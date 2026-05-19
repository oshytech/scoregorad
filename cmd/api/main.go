package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oshy/score-gorad/internal/api/handlers"
	"github.com/oshy/score-gorad/internal/api/routes"
	"github.com/oshy/score-gorad/internal/config"
	"github.com/oshy/score-gorad/internal/domain"
	"github.com/oshy/score-gorad/internal/observability"
	pgrepo "github.com/oshy/score-gorad/internal/repository/postgres"
	redisr "github.com/oshy/score-gorad/internal/repository/redis"
	"github.com/oshy/score-gorad/internal/service"
	"github.com/oshy/score-gorad/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Environment)
	slog.SetDefault(logger)

	// ── Infraestructura ──────────────────────────────────────────────────────

	db, err := pgrepo.NewDB(cfg.DatabaseURL)
	if err != nil {
		logger.Error("connecting to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// ── Repositories ─────────────────────────────────────────────────────────

	gameRepo := pgrepo.NewGameRepo(db)
	playerRepo := pgrepo.NewPlayerRepo(db)
	scoreRepo := pgrepo.NewScoreRepo(db)
	seasonRepo := pgrepo.NewSeasonRepo(db)
	_ = playerRepo

	var scoreRepoFinal domain.ScoreRepository = scoreRepo
	var redisPinger handlers.Pinger

	if cfg.RedisURL != "" {
		rc, err := redisr.NewClient(cfg.RedisURL)
		if err != nil {
			logger.Warn("could not connect to Redis, running without cache", "error", err)
		} else {
			defer rc.Close()
			redisPinger = redisr.NewPinger(rc)
			scoreRepoFinal = redisr.NewCachedScoreRepository(scoreRepo, rc, 2*time.Minute)
			logger.Info("leaderboard cache enabled", "backend", "redis")
		}
	}

	// ── Worker pool ───────────────────────────────────────────────────────────

	pool := worker.New(256,
		worker.WithRetry(3, 200*time.Millisecond),
		worker.WithErrorHandler(func(event worker.ScoreEvent, err error, attempts int) {
			logger.Error("worker event permanently failed",
				"game_id", event.GameID,
				"player_id", event.PlayerID,
				"attempts", attempts,
				"error", err,
			)
		}),
	)
	pool.Start(4, func(event worker.ScoreEvent) error {
		logger.Info("worker: score processed",
			"game_id", event.GameID,
			"player_id", event.PlayerID,
			"points", event.Points,
		)
		return nil
	})

	// ── Services ─────────────────────────────────────────────────────────────

	gameSvc := service.NewGameService(gameRepo)
	scoreSvc := service.NewScoreService(scoreRepoFinal, gameRepo, seasonRepo)
	scoreSvc.WithWorkerPool(pool)

	// ── Handlers ─────────────────────────────────────────────────────────────

	gameHandler := handlers.NewGameHandler(gameSvc)
	scoreHandler := handlers.NewScoreHandler(scoreSvc)
	healthHandler := handlers.NewHealthHandler(db, redisPinger)

	// ── HTTP server ───────────────────────────────────────────────────────────

	router := routes.Setup(gameHandler, scoreHandler, healthHandler, cfg.APIKeys)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: router,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("scoreGOard started", "port", cfg.ServerPort, "env", cfg.Environment)
		serverErr <- server.ListenAndServe()
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────────

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
	logger.Info("HTTP server stopped")

	if err := pool.Shutdown(shutdownCtx); err != nil {
		logger.Error("worker pool shutdown error", "error", err)
	}
	logger.Info("worker pool drained")

	logger.Info("scoreGOard stopped")
}
