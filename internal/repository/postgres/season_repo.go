package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/oshy/score-gorad/internal/domain"
)

type SeasonRepo struct {
	db *sql.DB
}

func NewSeasonRepo(db *sql.DB) *SeasonRepo {
	return &SeasonRepo{db: db}
}

func (r *SeasonRepo) Create(ctx context.Context, s *domain.Season) error {
	q := `INSERT INTO seasons (id, game_id, name, starts_at, ends_at, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, q,
		s.ID, s.GameID, s.Name, s.StartsAt, s.EndsAt, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting season: %w", err)
	}
	return nil
}

func (r *SeasonRepo) GetByID(ctx context.Context, id string) (*domain.Season, error) {
	q := `SELECT id, game_id, name, starts_at, ends_at, created_at, updated_at
	      FROM seasons WHERE id = $1`
	s := &domain.Season{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&s.ID, &s.GameID, &s.Name, &s.StartsAt, &s.EndsAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSeasonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying season by id: %w", err)
	}
	return s, nil
}

func (r *SeasonRepo) ListByGame(ctx context.Context, gameID string) ([]domain.Season, error) {
	q := `SELECT id, game_id, name, starts_at, ends_at, created_at, updated_at
	      FROM seasons WHERE game_id = $1 ORDER BY starts_at DESC`
	rows, err := r.db.QueryContext(ctx, q, gameID)
	if err != nil {
		return nil, fmt.Errorf("listing seasons: %w", err)
	}
	defer rows.Close()

	var seasons []domain.Season
	for rows.Next() {
		var s domain.Season
		if err := rows.Scan(&s.ID, &s.GameID, &s.Name, &s.StartsAt, &s.EndsAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning season: %w", err)
		}
		seasons = append(seasons, s)
	}
	return seasons, rows.Err()
}
