package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/oshy/score-gorad/internal/api/handlers"
	"github.com/oshy/score-gorad/internal/api/routes"
	"github.com/oshy/score-gorad/internal/config"
	"github.com/oshy/score-gorad/internal/domain"
	pgrepo "github.com/oshy/score-gorad/internal/repository/postgres"
	redisr "github.com/oshy/score-gorad/internal/repository/redis"
	"github.com/oshy/score-gorad/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	// Infraestructura — Postgres
	db, err := pgrepo.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	// Repositories (capa postgres)
	gameRepo := pgrepo.NewGameRepo(db)
	playerRepo := pgrepo.NewPlayerRepo(db)
	scoreRepo := pgrepo.NewScoreRepo(db)
	seasonRepo := pgrepo.NewSeasonRepo(db)

	_ = playerRepo

	// Si REDIS_URL está configurado, envolvemos el scoreRepo con la caché.
	// ScoreService sigue recibiendo domain.ScoreRepository — no sabe nada de Redis.
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

	// Services
	gameSvc := service.NewGameService(gameRepo)
	scoreSvc := service.NewScoreService(scoreRepoFinal, gameRepo, seasonRepo)

	// Handlers
	gameHandler := handlers.NewGameHandler(gameSvc)
	scoreHandler := handlers.NewScoreHandler(scoreSvc)

	// Router
	router := routes.Setup(gameHandler, scoreHandler)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("scoreGOard listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
