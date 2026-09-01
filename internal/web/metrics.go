package web

type WebServerMetrics interface {
	WebIncRequestsTotal()
	WebAddRequestDuration(seconds float64)
}
