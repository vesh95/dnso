package web

type WebServerMetrics interface {
	WebIncRequestsTotal()
	WebAddRequestDuration(method, path string, seconds float64)
}
