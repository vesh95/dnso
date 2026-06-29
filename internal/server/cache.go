package server

import (
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
	mu   sync.RWMutex
	data map[string]*cacheEntry
}

func NewDNSCache() *DNSCache {
	return &DNSCache{
		data: make(map[string]*cacheEntry),
	}
}

func (c *DNSCache) Get(name string, qtype uint16) (*dns.Msg, bool) {
	key := name + "|" + strconv.Itoa(int(qtype))

	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expire) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return nil, false
	}
	return entry.msg, true
}

func (c *DNSCache) Put(name string, qtype uint16, msg *dns.Msg, ttl uint32) {
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
