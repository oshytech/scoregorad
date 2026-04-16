package tests

// Benchmarks para la capa de repositorio de scores.
//
// Estos benchmarks requieren una base de datos PostgreSQL en ejecución.
// Se pueden correr con:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5432/scoregorad?sslmode=disable" \
//	  go test -bench=. -benchmem ./tests/
//
// Si no hay base de datos disponible, los benchmarks se omiten (t.Skip).
//
// Para comparar antes/después de los índices de Fase 2:
//
//	go test -bench=BenchmarkLeaderboard -benchtime=5s -count=3 ./tests/ | tee before.txt
//	# aplica los índices...
//	go test -bench=BenchmarkLeaderboard -benchtime=5s -count=3 ./tests/ | tee after.txt
//	benchstat before.txt after.txt

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oshy/score-gorad/internal/domain"
	"github.com/oshy/score-gorad/internal/repository/postgres"
)

func setupDB(b *testing.B) (*postgres.ScoreRepo, *postgres.GameRepo, *postgres.PlayerRepo, func()) {
	b.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		b.Skip("DATABASE_URL not set, skipping benchmark")
	}

	db, err := postgres.NewDB(dsn)
	if err != nil {
		b.Fatalf("connecting to db: %v", err)
	}

	return postgres.NewScoreRepo(db),
		postgres.NewGameRepo(db),
		postgres.NewPlayerRepo(db),
		func() { db.Close() }
}

// seedLeaderboard inserta n jugadores con 5 puntuaciones cada uno para un juego.
func seedLeaderboard(b *testing.B, scoreRepo *postgres.ScoreRepo, gameRepo *postgres.GameRepo, playerRepo *postgres.PlayerRepo, n int) string {
	b.Helper()
	ctx := context.Background()

	gameID := uuid.NewString()
	now := time.Now().UTC()
	if err := gameRepo.Create(ctx, &domain.Game{
		ID: gameID, Name: "bench-game", Slug: fmt.Sprintf("bench-%s", gameID[:8]),
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		b.Fatalf("creating game: %v", err)
	}

	for i := range n {
		playerID := uuid.NewString()
		if err := playerRepo.Create(ctx, &domain.Player{
			ID: playerID, Username: fmt.Sprintf("player-%d", i),
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			b.Fatalf("creating player: %v", err)
		}
		for j := range 5 {
			if err := scoreRepo.Create(ctx, &domain.Score{
				ID: uuid.NewString(), GameID: gameID, PlayerID: playerID,
				Points:    int64((i+1)*100 + j),
				CreatedAt: now.Add(time.Duration(j) * time.Second),
			}); err != nil {
				b.Fatalf("inserting score: %v", err)
			}
		}
	}

	return gameID
}

func BenchmarkLeaderboard(b *testing.B) {
	scoreRepo, gameRepo, playerRepo, cleanup := setupDB(b)
	defer cleanup()

	gameID := seedLeaderboard(b, scoreRepo, gameRepo, playerRepo, 1000)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := scoreRepo.GetLeaderboard(context.Background(), gameID, "", 25, 0)
		if err != nil {
			b.Fatalf("leaderboard error: %v", err)
		}
	}
}

func BenchmarkLeaderboardPage5(b *testing.B) {
	scoreRepo, gameRepo, playerRepo, cleanup := setupDB(b)
	defer cleanup()

	gameID := seedLeaderboard(b, scoreRepo, gameRepo, playerRepo, 1000)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := scoreRepo.GetLeaderboard(context.Background(), gameID, "", 25, 100)
		if err != nil {
			b.Fatalf("leaderboard page 5 error: %v", err)
		}
	}
}

func BenchmarkPlayerRank(b *testing.B) {
	scoreRepo, gameRepo, playerRepo, cleanup := setupDB(b)
	defer cleanup()

	gameID := seedLeaderboard(b, scoreRepo, gameRepo, playerRepo, 500)

	// Buscar el playerID del jugador 250 (mitad del ranking).
	entries, err := scoreRepo.GetLeaderboard(context.Background(), gameID, "", 1, 249)
	if err != nil || len(entries) == 0 {
		b.Skip("could not get mid-ranking player")
	}
	midPlayerID := entries[0].PlayerID

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := scoreRepo.GetPlayerRank(context.Background(), gameID, midPlayerID, "")
		if err != nil {
			b.Fatalf("player rank error: %v", err)
		}
	}
}
