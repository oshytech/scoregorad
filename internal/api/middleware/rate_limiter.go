package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implementa un token bucket por API key.
//
// Usa sync.Map en lugar de map + sync.RWMutex porque el patrón de acceso
// es mayoritariamente lecturas (requests de claves ya vistas) con escrituras
// esporádicas (primera vez que aparece una clave). sync.Map está optimizado
// exactamente para este caso — está documentado en el código fuente de Go
// como "amortized-constant-time loads, stores, and deletes".
//
// Con un map + RWMutex las escrituras bloquean todas las lecturas.
// Con sync.Map los limiters existentes se leen sin ningún lock.
type RateLimiter struct {
	mu       sync.Map // map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
	cleanupInterval time.Duration
}

// NewRateLimiter crea un RateLimiter con r requests/segundo y burst como
// tamaño máximo del bucket.
//
// Ejemplo: NewRateLimiter(10, 20) permite 10 req/s con ráfagas de hasta 20.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rps:             rate.Limit(rps),
		burst:           burst,
		cleanupInterval: 10 * time.Minute,
	}
	return rl
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	// LoadOrStore: si ya existe devuelve el existente; si no, guarda el nuevo.
	// La operación es atómica — no hay race entre la comprobación y la escritura.
	val, _ := rl.mu.LoadOrStore(key, rate.NewLimiter(rl.rps, rl.burst))
	return val.(*rate.Limiter)
}

// RateLimit devuelve el middleware de rate limiting.
// Claves no autenticadas (cuando Auth está desactivado) usan "anonymous" como key.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	rl := NewRateLimiter(rps, burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Identificar al cliente por API key o por IP si no hay key.
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.RemoteAddr
			}

			limiter := rl.getLimiter(key)
			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
