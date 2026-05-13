package middleware

import (
	"net/http"
	"strings"
)

type contextKey string

const APIKeyContextKey contextKey = "api_key"

// Auth valida la cabecera X-API-Key contra un mapa de claves permitidas.
//
// En producción las claves deberían estar en Postgres o en un secreto
// de Kubernetes, no en un mapa en memoria. Este enfoque es suficiente para
// el MVP y demuestra el patrón sin añadir infraestructura extra.
//
// Rutas excluidas de autenticación: /health y /metrics son internas
// y no deben requerir API key (se protegen a nivel de red, no de aplicación).
func Auth(validKeys map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// /health y /metrics son endpoints internos sin autenticación.
			if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("X-API-Key")
			if key == "" {
				// Compatibilidad con Authorization: Bearer <key>
				if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
					key = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if key == "" {
				http.Error(w, `{"error":"missing API key"}`, http.StatusUnauthorized)
				return
			}

			if _, ok := validKeys[key]; !ok {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
