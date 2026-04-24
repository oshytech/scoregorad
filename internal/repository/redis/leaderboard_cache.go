package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/oshy/score-gorad/internal/domain"
)

// CachedScoreRepository implementa domain.ScoreRepository envolviendo otra
// implementación con una capa de caché Redis.
//
// Este es el patrón decorator: CachedScoreRepository satisface exactamente la
// misma interfaz que el repositorio postgres que envuelve. El servicio no sabe
// si está hablando con Postgres directamente o con Redis + Postgres.
// La decisión de qué implementación usar se toma en main.go al cablear las
// dependencias — la lógica de negocio no cambia en absoluto.
type CachedScoreRepository struct {
	db    domain.ScoreRepository
	cache *redis.Client
	ttl   time.Duration
}

func NewCachedScoreRepository(db domain.ScoreRepository, cache *redis.Client, ttl time.Duration) *CachedScoreRepository {
	return &CachedScoreRepository{db: db, cache: cache, ttl: ttl}
}

// Create delega directamente en el repositorio subyacente.
// La invalidación de caché se gestiona aquí para mantener la coherencia.
func (r *CachedScoreRepository) Create(ctx context.Context, s *domain.Score) error {
	if err := r.db.Create(ctx, s); err != nil {
		return err
	}
	// Tras insertar un score nuevo, la caché del leaderboard de ese juego
	// ya no es válida. La invalidamos para forzar una recarga desde Postgres.
	r.invalidateGame(ctx, s.GameID)
	return nil
}

// GetLeaderboard implementa cache-aside:
//  1. Busca en Redis con una clave derivada de los parámetros.
//  2. Si hay hit, deserializa y devuelve.
//  3. Si hay miss, consulta Postgres, guarda en Redis con TTL y devuelve.
func (r *CachedScoreRepository) GetLeaderboard(ctx context.Context, gameID, seasonID string, limit, offset int) ([]domain.LeaderboardEntry, error) {
	key := leaderboardKey(gameID, seasonID, limit, offset)

	// Intento de cache hit
	data, err := r.cache.Get(ctx, key).Bytes()
	if err == nil {
		var entries []domain.LeaderboardEntry
		if jsonErr := json.Unmarshal(data, &entries); jsonErr == nil {
			return entries, nil
		}
		// Si la deserialización falla ignoramos la caché y vamos a Postgres.
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		// Redis está caído o hay un error real — seguimos sin caché,
		// no propagamos el error para no degradar el servicio.
		_ = err
	}

	// Cache miss: consultar Postgres
	entries, err := r.db.GetLeaderboard(ctx, gameID, seasonID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Guardar en caché. Si falla, no es crítico — el servicio sigue funcionando.
	if payload, jsonErr := json.Marshal(entries); jsonErr == nil {
		_ = r.cache.Set(ctx, key, payload, r.ttl).Err()
	}

	return entries, nil
}

// GetPlayerRank no se cachea en esta implementación: los ranks individuales
// cambian con cada nuevo score y el coste de invalidarlos selectivamente
// supera el beneficio. Se delega directamente a Postgres.
func (r *CachedScoreRepository) GetPlayerRank(ctx context.Context, gameID, playerID, seasonID string) (int, error) {
	return r.db.GetPlayerRank(ctx, gameID, playerID, seasonID)
}

// GetPlayerScores tampoco se cachea: el historial personal cambia con frecuencia
// y suele consultarse menos que el leaderboard global.
func (r *CachedScoreRepository) GetPlayerScores(ctx context.Context, playerID string, limit, offset int) ([]domain.Score, error) {
	return r.db.GetPlayerScores(ctx, playerID, limit, offset)
}

// invalidateGame elimina todas las entradas de caché de leaderboard para un juego.
// Usamos KEYS con patrón para encontrar todas las claves del juego y borrarlas.
//
// ADVERTENCIA: KEYS bloquea Redis mientras escanea todo el keyspace.
// En producción con millones de claves esto puede causar latencias de segundos.
// Lo corregiremos en el siguiente commit.
func (r *CachedScoreRepository) invalidateGame(ctx context.Context, gameID string) {
	pattern := fmt.Sprintf("lb:%s:*", gameID)
	keys, err := r.cache.Keys(ctx, pattern).Result()
	if err != nil || len(keys) == 0 {
		return
	}
	_ = r.cache.Del(ctx, keys...).Err()
}

// leaderboardKey genera la clave de caché para un leaderboard paginado.
// Incluye todos los parámetros que afectan al resultado.
func leaderboardKey(gameID, seasonID string, limit, offset int) string {
	season := "global"
	if seasonID != "" {
		season = seasonID
	}
	return fmt.Sprintf("lb:%s:%s:%d:%d", gameID, season, limit, offset)
}

// gameTrackKey devuelve la clave del Set que registra las claves de caché de un juego.
func gameTrackKey(gameID string) string {
	return fmt.Sprintf("lb:track:%s", gameID)
}
