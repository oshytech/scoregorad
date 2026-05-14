package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/oshy/score-gorad/internal/observability"
)

type requestIDKey struct{}

// RequestID genera un UUID por petición, lo inyecta en el context y lo
// añade a la respuesta como X-Request-ID.
//
// Usamos un tipo propio (requestIDKey{}) como clave de context en lugar
// de un string. Esto evita colisiones con otras librerías que usen el
// mismo string como clave — es el patrón recomendado en la documentación
// de context.WithValue.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		w.Header().Set("X-Request-ID", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extrae el request ID del context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// RequestIDLogger añade el request_id al logger del context.
// Debe ejecutarse después de RequestID en la cadena de middlewares.
func RequestIDLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := GetRequestID(r.Context())
		logger := observability.FromContext(r.Context()).With("request_id", reqID)
		ctx := observability.WithLogger(r.Context(), logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
