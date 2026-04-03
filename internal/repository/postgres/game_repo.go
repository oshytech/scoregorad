package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/oshy/score-gorad/internal/domain"
)

type GameRepo struct {
	db *sql.DB
}

func NewGameRepo(db *sql.DB) *GameRepo {
	return &GameRepo{db: db}
}

func (r *GameRepo) Create(ctx context.Context, g *domain.Game) error {
	q := `INSERT INTO games (id, name, slug, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q, g.ID, g.Name, g.Slug, g.CreatedAt, g.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrSlugAlreadyExists
		}
		return fmt.Errorf("inserting game: %w", err)
	}
	return nil
}

func (r *GameRepo) GetByID(ctx context.Context, id string) (*domain.Game, error) {
	q := `SELECT id, name, slug, created_at, updated_at FROM games WHERE id = $1`
	g := &domain.Game{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&g.ID, &g.Name, &g.Slug, &g.CreatedAt, &g.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrGameNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying game by id: %w", err)
	}
	return g, nil
}

func (r *GameRepo) GetBySlug(ctx context.Context, slug string) (*domain.Game, error) {
	q := `SELECT id, name, slug, created_at, updated_at FROM games WHERE slug = $1`
	g := &domain.Game{}
	err := r.db.QueryRowContext(ctx, q, slug).Scan(
		&g.ID, &g.Name, &g.Slug, &g.CreatedAt, &g.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrGameNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying game by slug: %w", err)
	}
	return g, nil
}

func (r *GameRepo) List(ctx context.Context) ([]domain.Game, error) {
	q := `SELECT id, name, slug, created_at, updated_at FROM games ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("listing games: %w", err)
	}
	defer rows.Close()

	var games []domain.Game
	for rows.Next() {
		var g domain.Game
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning game: %w", err)
		}
		games = append(games, g)
	}
	return games, rows.Err()
}
