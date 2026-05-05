package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/oshy/score-gorad/internal/domain"
	"github.com/oshy/score-gorad/internal/worker"
)

type ScoreService struct {
	scores  domain.ScoreRepository
	games   domain.GameRepository
	seasons domain.SeasonRepository
	pool    *worker.Pool // opcional: nil si el procesamiento asíncrono está desactivado
}

func NewScoreService(
	scores domain.ScoreRepository,
	games domain.GameRepository,
	seasons domain.SeasonRepository,
) *ScoreService {
	return &ScoreService{scores: scores, games: games, seasons: seasons}
}

// WithWorkerPool inyecta el pool de workers para procesamiento asíncrono.
// Si no se llama, SubmitScore funciona igual pero sin efectos secundarios async.
func (s *ScoreService) WithWorkerPool(pool *worker.Pool) {
	s.pool = pool
}

type SubmitScoreInput struct {
	GameID   string
	PlayerID string
	SeasonID string
	Points   int64
	Metadata map[string]any
}

func (s *ScoreService) SubmitScore(ctx context.Context, in SubmitScoreInput) (*domain.Score, error) {
	if in.Points < 0 {
		return nil, domain.ErrInvalidScore
	}

	// Verificar que el juego existe.
	if _, err := s.games.GetByID(ctx, in.GameID); err != nil {
		return nil, err
	}

	// Verificar que la temporada existe y no está cerrada.
	if in.SeasonID != "" {
		season, err := s.seasons.GetByID(ctx, in.SeasonID)
		if err != nil {
			return nil, err
		}
		if season.EndsAt != nil && time.Now().UTC().After(*season.EndsAt) {
			return nil, domain.ErrSeasonClosed
		}
	}

	score := &domain.Score{
		ID:        uuid.NewString(),
		GameID:    in.GameID,
		PlayerID:  in.PlayerID,
		SeasonID:  in.SeasonID,
		Points:    in.Points,
		Metadata:  in.Metadata,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.scores.Create(ctx, score); err != nil {
		return nil, err
	}

	// Enviar evento al pool para procesamiento asíncrono (efectos secundarios).
	// Si el pool está lleno se descarta: el cliente ya tiene su 201, no bloqueamos.
	if s.pool != nil {
		if err := s.pool.Submit(worker.ScoreEvent{
			GameID:   score.GameID,
			PlayerID: score.PlayerID,
			Points:   score.Points,
			SeasonID: score.SeasonID,
		}); err != nil {
			log.Printf("score event dropped (pool full): game=%s player=%s", score.GameID, score.PlayerID)
		}
	}

	return score, nil
}

func (s *ScoreService) GetLeaderboard(ctx context.Context, gameID, seasonID string, limit, offset int) ([]domain.LeaderboardEntry, error) {
	if _, err := s.games.GetByID(ctx, gameID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.scores.GetLeaderboard(ctx, gameID, seasonID, limit, offset)
}

func (s *ScoreService) GetPlayerRank(ctx context.Context, gameID, playerID, seasonID string) (int, error) {
	if _, err := s.games.GetByID(ctx, gameID); err != nil {
		return 0, err
	}
	rank, err := s.scores.GetPlayerRank(ctx, gameID, playerID, seasonID)
	if err != nil {
		return 0, fmt.Errorf("getting player rank: %w", err)
	}
	return rank, nil
}

func (s *ScoreService) GetPlayerScores(ctx context.Context, playerID string, limit, offset int) ([]domain.Score, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.scores.GetPlayerScores(ctx, playerID, limit, offset)
}
