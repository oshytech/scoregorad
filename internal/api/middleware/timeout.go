package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout envuelve cada petición en un context con deadline.
//
// Cuando el contexto expira, las llamadas a sql.QueryContext y cualquier
// operación que acepte context.Context se cancelan automáticamente.
// Esto es el payoff de haber propagado context desde el principio:
// un único middleware protege todas las capas sin tocar ninguna de ellas.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
