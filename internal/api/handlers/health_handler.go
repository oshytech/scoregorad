package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

// Pinger es cualquier dependencia que puede verificar su conectividad.
type Pinger interface {
	Ping(ctx context.Context) error
}

// sqlPinger adapta *sql.DB a Pinger usando PingContext.
type sqlPinger struct{ db *sql.DB }

func (s *sqlPinger) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// NewHealthHandler devuelve el handler de /health con checks reales.
//
// Este endpoint implementa un readiness check, no un liveness check:
//   - Liveness: el proceso está vivo (Kubernetes lo gestiona con el PID).
//   - Readiness: el proceso puede atender tráfico (DB y Redis responden).
//
// Si un check falla, devuelve 503 — Kubernetes dejará de enrutar tráfico
// a esta instancia hasta que se recupere, sin reiniciarla.
//
// El check usa un timeout corto (2s) para no bloquear el health check
// si una dependencia está lenta.
func NewHealthHandler(db *sql.DB, redisPinger Pinger) http.HandlerFunc {
	dbPinger := &sqlPinger{db: db}

	return func(w http.ResponseWriter, r *http.Request) {
		type checkResult struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		type response struct {
			Status string                 `json:"status"`
			Checks map[string]checkResult `json:"checks"`
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		checks := make(map[string]checkResult)
		overall := "ok"

		// Check Postgres
		if err := dbPinger.Ping(ctx); err != nil {
			checks["database"] = checkResult{Status: "error", Error: err.Error()}
			overall = "degraded"
		} else {
			checks["database"] = checkResult{Status: "ok"}
		}

		// Check Redis (solo si está configurado)
		if redisPinger != nil {
			if err := redisPinger.Ping(ctx); err != nil {
				checks["redis"] = checkResult{Status: "error", Error: err.Error()}
				overall = "degraded"
			} else {
				checks["redis"] = checkResult{Status: "ok"}
			}
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
