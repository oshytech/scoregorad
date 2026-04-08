package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/oshy/score-gorad/internal/api/handlers"
	"github.com/oshy/score-gorad/internal/api/routes"
	"github.com/oshy/score-gorad/internal/config"
	"github.com/oshy/score-gorad/internal/repository/postgres"
	"github.com/oshy/score-gorad/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	// Infraestructura
	db, err := postgres.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	// Repositories
	gameRepo := postgres.NewGameRepo(db)
	playerRepo := postgres.NewPlayerRepo(db)
	scoreRepo := postgres.NewScoreRepo(db)
	seasonRepo := postgres.NewSeasonRepo(db)

	// Evitar "declared and not used" en playerRepo hasta que se use en un handler futuro.
	_ = playerRepo

	// Services
	gameSvc := service.NewGameService(gameRepo)
	scoreSvc := service.NewScoreService(scoreRepo, gameRepo, seasonRepo)

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
