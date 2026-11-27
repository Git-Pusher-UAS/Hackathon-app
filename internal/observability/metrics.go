package observability

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequests        = promauto.NewCounterVec(prometheus.CounterOpts{Name: "sai_http_requests_total", Help: "Total HTTP requests"}, []string{"method", "route", "status"})
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "sai_http_request_duration_seconds", Help: "HTTP request duration", Buckets: prometheus.DefBuckets}, []string{"method", "route", "status"})
	RateLimitBlocked    = promauto.NewCounterVec(prometheus.CounterOpts{Name: "sai_rate_limit_blocked_total", Help: "Rate limit blocks"}, []string{"key_prefix"})
	OTPChallenges       = promauto.NewCounterVec(prometheus.CounterOpts{Name: "sai_otp_challenges_total", Help: "OTP challenges created"}, []string{"kind"})
	OTPVerifications    = promauto.NewCounterVec(prometheus.CounterOpts{Name: "sai_otp_verifications_total", Help: "OTP verifications"}, []string{"kind", "result"})
	ContactReleases     = promauto.NewCounter(prometheus.CounterOpts{Name: "sai_contact_releases_total", Help: "Contact releases"})
	AuditAppendErrors   = promauto.NewCounter(prometheus.CounterOpts{Name: "sai_audit_append_errors_total", Help: "Audit append errors"})
)

func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		route := c.Route().Path
		if route == "" {
			route = c.Path()
		}
		method := c.Method()
		labels := prometheus.Labels{"method": method, "route": route, "status": strconv.Itoa(status)}
		HTTPRequests.With(labels).Inc()
		HTTPRequestDuration.With(labels).Observe(time.Since(start).Seconds())
		return err
	}
}
