package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/oshy/score-gorad/internal/observability"
)

// Metrics registra HTTPRequestsTotal y HTTPRequestDuration por petición.
// Se coloca dentro de Auth/RateLimit para no contabilizar requests rechazadas
// como tráfico legítimo en las métricas de latencia.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.status)

		observability.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		observability.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}
