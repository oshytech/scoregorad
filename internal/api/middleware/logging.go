package middleware

import (
	"net/http"
	"time"

	"github.com/oshy/score-gorad/internal/observability"
)

// responseWriter captura el status code que escribe el handler.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging registra cada petición con slog estructurado.
// Los campos incluyen method, path, status, duration_ms y request_id
// (si RequestID se ha ejecutado antes en la cadena).
//
// Al usar slog con JSON handler en producción, estos campos son directamente
// indexables en cualquier sistema de logs (Datadog, Loki, CloudWatch).
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		logger := observability.FromContext(r.Context())
		logger.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
