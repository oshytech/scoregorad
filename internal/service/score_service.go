package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oshy/score-gorad/internal/domain"
)

type ScoreService struct {
	scores  domain.ScoreRepository
	games   domain.GameRepository
	seasons domain.SeasonRepository
}

func NewScoreService(
	scores domain.ScoreRepository,
	games domain.GameRepository,
	seasons domain.SeasonRepository,
) *ScoreService {
	return &ScoreService{scores: scores, games: games, seasons: seasons}
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
