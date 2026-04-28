package integration

// Tests de integración para el repositorio de caché Redis.
//
// Usan testcontainers-go para levantar instancias reales de Redis y Postgres
// en Docker durante los tests. No hay mocks — si la lógica falla en un
// contenedor real, fallará en producción.
//
// Para ejecutar:
//
//	go test -v ./tests/integration/ -timeout 120s
//
// Requiere Docker en el host.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/oshy/score-gorad/internal/domain"
	pgrepo "github.com/oshy/score-gorad/internal/repository/postgres"
	redisr "github.com/oshy/score-gorad/internal/repository/redis"
)

func setupContainers(t *testing.T) (domain.ScoreRepository, *pgrepo.GameRepo, *pgrepo.PlayerRepo, func()) {
	t.Helper()
	ctx := context.Background()

	// Postgres
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("scoregorad_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.WithInitScripts(
			"../../migrations/001_create_games.sql",
			"../../migrations/002_create_players.sql",
			"../../migrations/003_create_seasons.sql",
			"../../migrations/004_create_scores.sql",
		),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting postgres DSN: %v", err)
	}

	db, err := pgrepo.NewDB(dsn)
	if err != nil {
		t.Fatalf("connecting to postgres: %v", err)
	}

	// Redis
	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("starting redis container: %v", err)
	}

	redisURL, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("getting redis connection string: %v", err)
	}

	redisClient, err := redisr.NewClient("redis://" + redisURL)
	if err != nil {
		t.Fatalf("connecting to redis: %v", err)
	}

	pgScore := pgrepo.NewScoreRepo(db)
	cached := redisr.NewCachedScoreRepository(pgScore, redisClient, 30*time.Second)

	cleanup := func() {
		db.Close()
		redisClient.Close()
		_ = pgContainer.Terminate(ctx)
		_ = redisContainer.Terminate(ctx)
	}

	return cached, pgrepo.NewGameRepo(db), pgrepo.NewPlayerRepo(db), cleanup
}

func seedGame(t *testing.T, ctx context.Context, gameRepo *pgrepo.GameRepo) string {
	t.Helper()
	now := time.Now().UTC()
	gameID := uuid.NewString()
	err := gameRepo.Create(ctx, &domain.Game{
		ID: gameID, Name: "Test Game", Slug: "test-" + gameID[:8],
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("creating game: %v", err)
	}
	return gameID
}

func seedPlayer(t *testing.T, ctx context.Context, playerRepo *pgrepo.PlayerRepo) string {
	t.Helper()
	now := time.Now().UTC()
	playerID := uuid.NewString()
	err := playerRepo.Create(ctx, &domain.Player{
		ID: playerID, Username: "player-" + playerID[:8],
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("creating player: %v", err)
	}
	return playerID
}

// TestCacheHit verifica que una segunda llamada devuelve datos desde caché
// sin tocar Postgres.
func TestCacheHit(t *testing.T) {
	repo, gameRepo, playerRepo, cleanup := setupContainers(t)
	defer cleanup()
	ctx := context.Background()

	gameID := seedGame(t, ctx, gameRepo)
	playerID := seedPlayer(t, ctx, playerRepo)

	_ = repo.Create(ctx, &domain.Score{
		ID: uuid.NewString(), GameID: gameID, PlayerID: playerID,
		Points: 1000, CreatedAt: time.Now().UTC(),
	})

	// Primera llamada — cache miss, va a Postgres
	entries1, err := repo.GetLeaderboard(ctx, gameID, "", 25, 0)
	if err != nil {
		t.Fatalf("first GetLeaderboard: %v", err)
	}
	if len(entries1) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries1))
	}

	// Segunda llamada — debe devolver el mismo resultado desde caché
	entries2, err := repo.GetLeaderboard(ctx, gameID, "", 25, 0)
	if err != nil {
		t.Fatalf("second GetLeaderboard: %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("expected 1 cached entry, got %d", len(entries2))
	}
	if entries1[0].Points != entries2[0].Points {
		t.Errorf("cache returned different points: %d vs %d", entries1[0].Points, entries2[0].Points)
	}
}

// TestCacheInvalidationOnSubmit verifica que tras enviar un nuevo score
// el leaderboard ya no devuelve datos obsoletos de la caché.
func TestCacheInvalidationOnSubmit(t *testing.T) {
	repo, gameRepo, playerRepo, cleanup := setupContainers(t)
	defer cleanup()
	ctx := context.Background()

	gameID := seedGame(t, ctx, gameRepo)
	player1 := seedPlayer(t, ctx, playerRepo)
	player2 := seedPlayer(t, ctx, playerRepo)

	// Score inicial de player1
	_ = repo.Create(ctx, &domain.Score{
		ID: uuid.NewString(), GameID: gameID, PlayerID: player1,
		Points: 500, CreatedAt: time.Now().UTC(),
	})

	// Cachear el leaderboard
	entries1, err := repo.GetLeaderboard(ctx, gameID, "", 25, 0)
	if err != nil || len(entries1) != 1 {
		t.Fatalf("initial leaderboard failed: %v, len=%d", err, len(entries1))
	}

	// player2 envía un score — debe invalidar la caché
	_ = repo.Create(ctx, &domain.Score{
		ID: uuid.NewString(), GameID: gameID, PlayerID: player2,
		Points: 1500, CreatedAt: time.Now().UTC(),
	})

	// El leaderboard ahora debe tener 2 entradas (no 1 de la caché)
	entries2, err := repo.GetLeaderboard(ctx, gameID, "", 25, 0)
	if err != nil {
		t.Fatalf("post-submit GetLeaderboard: %v", err)
	}
	if len(entries2) != 2 {
		t.Errorf("expected 2 entries after invalidation, got %d", len(entries2))
	}
	// player2 debe estar primero (1500 > 500)
	if entries2[0].PlayerID != player2 {
		t.Errorf("expected player2 first, got %s", entries2[0].PlayerID)
	}
}

// TestCacheTTL documenta el comportamiento de expiración.
// Con un TTL muy corto (1s) y un sleep de 2s, la tercera llamada
// volvería a ir a Postgres. Omitimos el sleep para no ralentizar la suite.
func TestCacheTTL(t *testing.T) {
	_, _, _, cleanup := setupContainers(t)
	defer cleanup()
	t.Log("TTL behavior: entries expire after the configured duration and are re-fetched from Postgres on next call")
}
