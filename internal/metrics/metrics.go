package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	Registry *prometheus.Registry
	metrics  *metrics
}

type metrics struct {
	webRequestDuration  prometheus.Histogram
	webRequestsTotal    prometheus.Counter
	dnsRequestTotal     prometheus.Counter
	dnsCachedZonesCount prometheus.Gauge
	dnsCacheGets        prometheus.Counter
	dnsCacheSets        prometheus.Counter
	dnsCacheHits        prometheus.Counter
	dnsCacheMiss        prometheus.Counter
	dnsUpstreamsTotal   prometheus.Counter
	dnsRequestsDuration prometheus.Histogram
	dnsCacheRecSize     prometheus.Gauge
}

func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()
	m := &metrics{
		webRequestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "dnso_web_request_duration",
			Help:    "Длительности запросов REST API",
			Buckets: prometheus.DefBuckets,
		}),
		webRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dnso_web_requests_count",
			Help: "Количество запросов REST API",
		}),
		dnsCachedZonesCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "dnso_dns_cached_zonesCount",
			Help: "Количество закэшированных зон DNS",
		}),
		dnsCacheGets: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dnso_dns_cache_gets",
			Help: "Количество запросов из кэша",
		}),
		dnsCacheSets: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dnso_dns_cache_sets",
			Help: "Количество записей в кэш",
		}),
		dnsCacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dnso_dns_cache_hits",
			Help: "Количество кэш-попаданий",
		}),
		dnsCacheMiss: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dnso_dns_cache_miss",
			Help: "Количество кэш-промахов",
		}),
		dnsUpstreamsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dnso_dns_upstreams_total",
			Help: "Количество обращений ко внешним DNS",
		}),
		dnsRequestsDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "dnso_dns_requests_duration",
			Help:    "Длительность отвентов на DNS запросы",
			Buckets: prometheus.DefBuckets,
		}),
		dnsCacheRecSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "dnso_dns_cache_rec_size",
			Help: "Размер кэша записей DNS",
		}),
	}
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.webRequestDuration,
		m.webRequestsTotal,
		m.dnsCachedZonesCount,
		m.dnsCacheGets,
		m.dnsCacheSets,
		m.dnsCacheHits,
		m.dnsCacheMiss,
		m.dnsUpstreamsTotal,
		m.dnsRequestsDuration,
		m.dnsCacheRecSize,
	)

	return &Registry{
		Registry: reg,
		metrics:  m,
	}
}

func (reg *Registry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.HandlerFor(reg.Registry, promhttp.HandlerOpts{}))

	m.ServeHTTP(w, r)
}

// WebAddRequestDuration implements [web.WebServerMetrics].
func (r *Registry) WebAddRequestDuration(seconds float64) {
	r.metrics.webRequestDuration.Observe(seconds)
}

// WebIncRequestsTotal implements [web.WebServerMetrics].
func (r *Registry) WebIncRequestsTotal() {
	r.metrics.webRequestsTotal.Inc()
}

// DnsCachedZonesCount implements [server.DnsMetrics].
func (r *Registry) DnsCachedZonesCount(count int) {
	r.metrics.dnsCachedZonesCount.Set(float64(count))
}

// DnsIncGets implements [server.DnsMetrics].
func (r *Registry) DnsCacheIncGets() {
	r.metrics.dnsCacheGets.Inc()
}

// DnsIncHit implements [server.DnsMetrics].
func (r *Registry) DnsCacheIncHit() {
	r.metrics.dnsCacheHits.Inc()
}

// DnsIncMiss implements [server.DnsMetrics].
func (r *Registry) DnsCacheIncMiss() {
	r.metrics.dnsCacheMiss.Inc()
}

// DnsIncRequestsTotal implements [server.DnsMetrics].
func (r *Registry) DnsIncRequestsTotal() {
	r.metrics.dnsRequestTotal.Inc()
}

// DnsIncSets implements [server.DnsMetrics].
func (r *Registry) DnsCacheIncSets() {
	r.metrics.dnsCacheSets.Inc()
}

// DnsIncUpstreamsTotal implements [server.DnsMetrics].
func (r *Registry) DnsIncUpstreamsTotal() {
	r.metrics.dnsUpstreamsTotal.Inc()
}

// DnsRequestDuration implements [server.DnsMetrics].
func (r *Registry) DnsRequestDuration(seconds float64) {
	r.metrics.dnsRequestsDuration.Observe(seconds)
}

// DnsSetCacheSize implements [server.DnsMetrics].
func (r *Registry) DnsSetCacheSize(count int) {
	r.metrics.dnsCacheRecSize.Set(float64(count))
}
