package server

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type cacheEntry struct {
	msg    *dns.Msg
	expire time.Time
}

type DNSCache struct {
	mu      sync.RWMutex
	data    map[string]*cacheEntry
	metrics RecCacheMetrics

	stopBackground func()
}

func NewDNSCache(metrics RecCacheMetrics) *DNSCache {
	ctx, cancel := context.WithCancel(context.Background())
	cache := &DNSCache{
		data:           make(map[string]*cacheEntry),
		metrics:        metrics,
		stopBackground: cancel,
	}

	go func() {
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				cache.metrics.DnsSetCacheSize(len(cache.data))
			}
		}
	}()

	return cache
}

// Shutdown останавливает фоновые задачи кэша по сбору метрик
func (c *DNSCache) Shutdown() {
	c.stopBackground()
}

// Get делает попытку получения значения и возвращает `*dns.Msg` и флаг присутствия значения в кэше
func (c *DNSCache) Get(name string, qtype uint16) (*dns.Msg, bool) {
	defer c.metrics.DnsCacheIncGets()
	key := name + "|" + strconv.Itoa(int(qtype))

	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		c.metrics.DnsCacheIncMiss()
		return nil, false
	}
	if time.Now().After(entry.expire) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		c.metrics.DnsCacheIncMiss()
		return nil, false
	}
	c.metrics.DnsCacheIncHit()
	return entry.msg, true
}

// Put производит вставку значения в кэш или заменяет значение в существующей записи
func (c *DNSCache) Put(name string, qtype uint16, msg *dns.Msg, ttl uint32) {
	defer c.metrics.DnsCacheIncSets()
	if ttl == 0 {
		ttl = 60
	}

	key := name + "|" + strconv.Itoa(int(qtype))
	exp := time.Now().Add(time.Duration(ttl) * time.Second)

	c.mu.Lock()
	c.data[key] = &cacheEntry{
		msg:    msg,
		expire: exp,
	}
	c.mu.Unlock()
}
