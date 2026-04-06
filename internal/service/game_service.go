package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oshy/score-gorad/internal/domain"
)

type GameService struct {
	games domain.GameRepository
}

func NewGameService(games domain.GameRepository) *GameService {
	return &GameService{games: games}
}

type CreateGameInput struct {
	Name string
	Slug string
}

func (s *GameService) CreateGame(ctx context.Context, in CreateGameInput) (*domain.Game, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(in.Slug) == "" {
		return nil, fmt.Errorf("%w: slug is required", domain.ErrInvalidInput)
	}

	now := time.Now().UTC()
	g := &domain.Game{
		ID:        uuid.NewString(),
		Name:      in.Name,
		Slug:      in.Slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.games.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *GameService) GetGame(ctx context.Context, id string) (*domain.Game, error) {
	return s.games.GetByID(ctx, id)
}

func (s *GameService) ListGames(ctx context.Context) ([]domain.Game, error) {
	return s.games.List(ctx)
}
