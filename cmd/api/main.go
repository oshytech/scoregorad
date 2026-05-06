package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oshy/score-gorad/internal/api/handlers"
	"github.com/oshy/score-gorad/internal/api/routes"
	"github.com/oshy/score-gorad/internal/config"
	"github.com/oshy/score-gorad/internal/domain"
	pgrepo "github.com/oshy/score-gorad/internal/repository/postgres"
	redisr "github.com/oshy/score-gorad/internal/repository/redis"
	"github.com/oshy/score-gorad/internal/service"
	"github.com/oshy/score-gorad/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	// ── Infraestructura ──────────────────────────────────────────────────────

	db, err := pgrepo.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	// ── Repositories ─────────────────────────────────────────────────────────

	gameRepo := pgrepo.NewGameRepo(db)
	playerRepo := pgrepo.NewPlayerRepo(db)
	scoreRepo := pgrepo.NewScoreRepo(db)
	seasonRepo := pgrepo.NewSeasonRepo(db)
	_ = playerRepo

	var scoreRepoFinal domain.ScoreRepository = scoreRepo
	if cfg.RedisURL != "" {
		redisClient, err := redisr.NewClient(cfg.RedisURL)
		if err != nil {
			log.Printf("warning: could not connect to Redis (%v), running without cache", err)
		} else {
			scoreRepoFinal = redisr.NewCachedScoreRepository(scoreRepo, redisClient, 2*time.Minute)
			log.Println("leaderboard cache enabled (Redis)")
		}
	}

	// ── Worker pool ───────────────────────────────────────────────────────────

	pool := worker.New(4, 256)
	pool.Start(4, func(event worker.ScoreEvent) error {
		// Aquí irían los efectos secundarios: notificaciones, estadísticas, etc.
		// Por ahora solo logueamos el evento para demostrar el flujo.
		log.Printf("worker: processed score game=%s player=%s points=%d",
			event.GameID, event.PlayerID, event.Points)
		return nil
	})

	// ── Services ─────────────────────────────────────────────────────────────

	gameSvc := service.NewGameService(gameRepo)
	scoreSvc := service.NewScoreService(scoreRepoFinal, gameRepo, seasonRepo)
	scoreSvc.WithWorkerPool(pool)

	// ── HTTP server ───────────────────────────────────────────────────────────

	gameHandler := handlers.NewGameHandler(gameSvc)
	scoreHandler := handlers.NewScoreHandler(scoreSvc)
	router := routes.Setup(gameHandler, scoreHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: router,
	}

	// Lanzar el servidor en una goroutine separada para poder esperar señales
	// en el hilo principal sin bloquear.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("scoreGOard listening on :%s", cfg.ServerPort)
		serverErr <- server.ListenAndServe()
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────────
	//
	// Esperamos SIGINT (Ctrl+C) o SIGTERM (señal de Kubernetes / Docker).
	// El orden de shutdown importa:
	//  1. Servidor HTTP: deja de aceptar nuevas conexiones, drena las activas.
	//  2. Worker pool: espera que terminen los jobs en vuelo.
	//  3. Postgres: se cierra con defer al salir de main.
	//
	// Si cerramos Postgres antes de que los workers terminen, las operaciones
	// de DB en curso devolverán error. El orden correcto es fundamental.

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	case sig := <-quit:
		log.Printf("received signal %s, shutting down...", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Parar el servidor HTTP
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	log.Println("HTTP server stopped")

	// 2. Drenar el worker pool
	if err := pool.Shutdown(shutdownCtx); err != nil {
		log.Printf("worker pool shutdown error: %v", err)
	}
	log.Println("worker pool drained")

	// 3. Postgres se cierra con el defer db.Close() al final de main
	log.Println("scoreGOard stopped")
}
