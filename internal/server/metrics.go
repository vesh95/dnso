package server

type HandlerMetrics interface {
	DnsIncRequestsTotal()
	DnsIncUpstreamsTotal()
	DnsRequestDuration(seconds float64)
	DnsCachedZonesCount(count int)
}

type RecCacheMetrics interface {
	DnsSetCacheSize(count int)
	DnsCacheIncMiss()
	DnsCacheIncHit()
	DnsCacheIncGets()
	DnsCacheIncSets()
}

type UpstreamMetrics interface {
	DnsIncExchangesTotal()
	DnsIncErrorsTotal()
	DnsSetUpstreamsCount(count int)
	DnsUpstreamDuration(seconds float64)
}

type DnsMetrics interface {
	HandlerMetrics
	RecCacheMetrics
	UpstreamExchanger
}
