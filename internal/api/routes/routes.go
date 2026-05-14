package routes

import (
	"net/http"
	"time"

	"github.com/oshy/score-gorad/internal/api/handlers"
	"github.com/oshy/score-gorad/internal/api/middleware"
)

// Setup registra las rutas y aplica la cadena de middlewares.
//
// Orden de middlewares (de fuera hacia dentro):
//  1. RequestID    — genera el ID antes de que nada más se ejecute.
//  2. RequestIDLogger — añade request_id al logger del context.
//  3. Timeout      — deadline antes del logging para medir tiempo real.
//  4. Logging      — registra la petición con slog estructurado.
//  5. Auth         — rechaza peticiones sin API key válida.
//  6. RateLimit    — limita requests por key.
//  7. mux          — enrutado a handlers.
func Setup(
	games *handlers.GameHandler,
	scores *handlers.ScoreHandler,
	health http.HandlerFunc,
	validKeys map[string]struct{},
) http.Handler {
	mux := http.NewServeMux()

	// /health y /metrics no requieren auth (ver Auth middleware)
	mux.HandleFunc("GET /health", health)

	// Games
	mux.HandleFunc("POST /games", games.CreateGame)
	mux.HandleFunc("GET /games", games.ListGames)
	mux.HandleFunc("GET /games/{gameId}", games.GetGame)

	// Scores y leaderboard
	mux.HandleFunc("POST /games/{gameId}/scores", scores.SubmitScore)
	mux.HandleFunc("GET /games/{gameId}/leaderboard", scores.GetLeaderboard)
	mux.HandleFunc("GET /games/{gameId}/seasons/{seasonId}/leaderboard", scores.GetLeaderboard)
	mux.HandleFunc("GET /games/{gameId}/players/{playerId}/rank", scores.GetPlayerRank)

	// Historial del jugador
	mux.HandleFunc("GET /players/{playerId}/scores", scores.GetPlayerScores)

	chain := middleware.RequestID(
		middleware.RequestIDLogger(
			middleware.Timeout(5*time.Second)(
				middleware.Logging(
					middleware.Auth(validKeys)(
						middleware.RateLimit(20, 40)(
							mux,
						),
					),
				),
			),
		),
	)

	return chain
}
