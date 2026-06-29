package server

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"dnso/internal/repository"

	"github.com/miekg/dns"
)

type HandlerConfig struct {
	Client        *dns.Client
	UpstreamAddr  string
	ZoneStorage   repository.ZoneRepository
	RecordStorage repository.RecordRepository
	Cache         *DNSCache
}

type Handler struct {
	client        *dns.Client
	upstreamAddr  string
	zoneStorage   repository.ZoneRepository
	recordStorage repository.RecordRepository
	cache         *DNSCache

	mu         sync.RWMutex
	localZones map[string]struct{} // set of known local zone names (FQDN)
}

func NewHandler(config *HandlerConfig) *Handler {
	h := &Handler{
		client:        config.Client,
		upstreamAddr:  config.UpstreamAddr,
		zoneStorage:   config.ZoneStorage,
		recordStorage: config.RecordStorage,
		cache:         config.Cache,
		localZones:    make(map[string]struct{}),
	}
	h.refreshZones()
	return h
}

// refreshZones загружает список локальных зон из БД в in-memory кэш.
func (h *Handler) refreshZones() {
	zones, err := h.zoneStorage.GetAll(context.Background())
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.localZones = make(map[string]struct{}, len(zones))
	for _, z := range zones {
		h.localZones[strings.ToLower(z.Name)] = struct{}{}
	}
}

// isLocalZone проверяет, существует ли зона для указанного домена (FQDN).
// Поиск выполняется итеративно: сначала проверяется полное имя,
// затем отсекается левая часть (label) до тех пор, пока не останется
// корневая зона или зона не будет найдена.
func (h *Handler) isLocalZone(name string) bool {
	name = strings.ToLower(name)
	labels := dns.SplitDomainName(name)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for i := 0; i < len(labels); i++ {
		candidate := dns.Fqdn(strings.Join(labels[i:], "."))
		if _, ok := h.localZones[candidate]; ok {
			return true
		}
	}
	return false
}

// makeLocalRRs создаёт DNS Resource Records из локального хранилища записей.
// Возвращает все записи, соответствующие домену и типу.
func (h *Handler) makeLocalRRs(name string, qtype uint16) ([]dns.RR, error) {
	typeStr := dns.TypeToString[qtype]
	if typeStr == "" {
		return nil, fmt.Errorf("unsupported DNS type: %d", qtype)
	}

	records, err := h.recordStorage.Get(context.Background(), name, typeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no records found for %s type %s", name, typeStr)
	}

	rrs := make([]dns.RR, 0, len(records))
	for _, rec := range records {
		rrStr := fmt.Sprintf("%s %d %s %s", name, rec.TTL, typeStr, rec.Rdata)
		rr, err := dns.NewRR(rrStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RR %q: %w", rrStr, err)
		}
		rrs = append(rrs, rr)
	}

	return rrs, nil
}

// resolveLocal пытается разрешить имя через локальное хранилище.
// Если найдена CNAME-запись, следует по цепочке (до maxCNAMEFollow).
func (h *Handler) resolveLocal(name string, qtype uint16, depth int) ([]dns.RR, error) {
	const maxCNAMEFollow = 8
	if depth > maxCNAMEFollow {
		return nil, fmt.Errorf("CNAME loop detected for %s", name)
	}

	// Сначала ищем запись запрошенного типа
	rrs, err := h.makeLocalRRs(name, qtype)
	if err == nil {
		return rrs, nil
	}

	// Если не нашли — ищем CNAME
	cnameRRs, err := h.makeLocalRRs(name, dns.TypeCNAME)
	if err != nil {
		return nil, fmt.Errorf("no records found for %s", name)
	}

	// Следуем по CNAME
	var result []dns.RR
	for _, cnameRR := range cnameRRs {
		cname := cnameRR.(*dns.CNAME)
		target := dns.Fqdn(cname.Target)

		result = append(result, cnameRR)

		targetRRs, err := h.resolveLocal(target, qtype, depth+1)
		if err == nil {
			result = append(result, targetRRs...)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no records found for %s", name)
	}

	return result, nil
}

func (h *Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = false

	var proxyQuestions []dns.Question
	var anyLocal bool

	for _, q := range r.Question {
		name := dns.Fqdn(q.Name)

		// 1. Проверяем кэш
		cached, found := h.cache.Get(name, q.Qtype)
		if found {
			for _, rr := range cached.Answer {
				if rr.Header().Name == name && rr.Header().Rrtype == q.Qtype {
					m.Answer = append(m.Answer, rr)
				}
			}
			// Если в кэше был NXDOMAIN, копируем rcode
			if cached.Rcode == dns.RcodeNameError {
				m.SetRcode(r, dns.RcodeNameError)
			}
			continue
		}

		// 2. Проверяем локальную зону
		if h.isLocalZone(name) {
			anyLocal = true

			rrs, err := h.resolveLocal(name, q.Qtype, 0)
			if err == nil {
				m.Answer = append(m.Answer, rrs...)
				m.Authoritative = true
				continue
			}

			// Запись не найдена в локальной зоне — NODATA (NOERROR + пустой Answer)
			// Не возвращаем NXDOMAIN, так как зона существует
			continue
		}

		proxyQuestions = append(proxyQuestions, q)
	}

	// 3. Отправляем upstream-запрос для нелокальных вопросов
	if len(proxyQuestions) > 0 {
		req := new(dns.Msg)
		req.SetReply(r)
		req.Question = proxyQuestions
		req.Rcode = dns.RcodeSuccess
		req.RecursionDesired = r.RecursionDesired

		if opt := r.IsEdns0(); opt != nil {
			req.SetEdns0(opt.UDPSize(), false)
		}

		resp, _, err := h.client.Exchange(req, h.upstreamAddr)
		if err != nil || resp == nil {
			m.SetRcode(r, dns.RcodeServerFailure)
			w.WriteMsg(m)
			return
		}

		// Копируем весь ответ upstream (Answer, Ns, Extra)
		m.Answer = append(m.Answer, resp.Answer...)
		m.Ns = resp.Ns
		m.Extra = resp.Extra
		m.Rcode = resp.Rcode
	} else if !anyLocal && len(m.Answer) == 0 && len(r.Question) > 0 {
		// Все вопросы были из кэша, но ответов нет — ServerFailure
		m.SetRcode(r, dns.RcodeServerFailure)
	}

	// 4. Кэшируем результат для каждого вопроса
	for _, q := range r.Question {
		name := dns.Fqdn(q.Name)
		var foundRRs []dns.RR

		for _, rr := range m.Answer {
			if rr.Header().Name == name && rr.Header().Rrtype == q.Qtype {
				foundRRs = append(foundRRs, rr)
			}
		}

		if len(foundRRs) > 0 {
			ttl := uint32(60)
			for _, rr := range foundRRs {
				if rr.Header().Ttl < ttl {
					ttl = rr.Header().Ttl
				}
			}

			miniMsg := new(dns.Msg)
			miniMsg.SetReply(r)
			miniMsg.Answer = foundRRs
			miniMsg.Rcode = m.Rcode
			h.cache.Put(name, q.Qtype, miniMsg, ttl)
		} else if m.Rcode == dns.RcodeNameError {
			// Для NXDOMAIN используем TTL из SOA (RFC 2308) или 60 по умолчанию
			ttl := uint32(60)
			for _, rr := range m.Ns {
				if soa, ok := rr.(*dns.SOA); ok {
					ttl = soa.Minttl
					break
				}
			}

			miniMsg := new(dns.Msg)
			miniMsg.SetReply(r)
			miniMsg.SetRcode(r, dns.RcodeNameError)
			h.cache.Put(name, q.Qtype, miniMsg, ttl)
		}
	}

	w.WriteMsg(m)
}
