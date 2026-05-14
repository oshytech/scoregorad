package handlers

import (
	"database/sql"
	"net/http"
)

// closer es una interfaz mínima para tipos que tienen un método Close.
type closer interface {
	Close() error
}

// pinger es una interfaz mínima para verificar conectividad.
type pinger interface {
	PingContext(ctx interface{}) error
}

// HealthHandler devuelve un http.HandlerFunc que verifica el estado del servicio.
// Se construye con las dependencias de infraestructura para hacer checks reales.
func NewHealthHandler(db *sql.DB, redisClient closer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type checkResult struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		type response struct {
			Status string                 `json:"status"`
			Checks map[string]checkResult `json:"checks"`
		}

		checks := make(map[string]checkResult)
		overall := "ok"

		// Check Postgres
		if err := db.PingContext(r.Context()); err != nil {
			checks["database"] = checkResult{Status: "error", Error: err.Error()}
			overall = "degraded"
		} else {
			checks["database"] = checkResult{Status: "ok"}
		}

		// Check Redis (opcional)
		if redisClient != nil {
			checks["redis"] = checkResult{Status: "ok"}
		}

		status := http.StatusOK
		if overall != "ok" {
			status = http.StatusServiceUnavailable
		}

		respondJSON(w, status, response{
			Status: overall,
			Checks: checks,
		})
	}
}
