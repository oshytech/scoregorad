package routes

import (
	"net/http"
	"time"

	"github.com/oshy/score-gorad/internal/api/handlers"
	"github.com/oshy/score-gorad/internal/api/middleware"
)

func Setup(games *handlers.GameHandler, scores *handlers.ScoreHandler) http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", handlers.HealthHandler)

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

	// Orden de middlewares: Timeout envuelve a Logging envuelve al mux.
	// El timeout se aplica antes del logging para que la duración registrada
	// incluya el tiempo de espera si se excede.
	return middleware.Timeout(5 * time.Second)(middleware.Logging(mux))
}
