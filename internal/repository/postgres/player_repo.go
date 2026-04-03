package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/oshy/score-gorad/internal/domain"
)

type PlayerRepo struct {
	db *sql.DB
}

func NewPlayerRepo(db *sql.DB) *PlayerRepo {
	return &PlayerRepo{db: db}
}

func (r *PlayerRepo) Create(ctx context.Context, p *domain.Player) error {
	q := `INSERT INTO players (id, username, external_id, created_at, updated_at)
	      VALUES ($1, $2, NULLIF($3,''), $4, $5)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Username, p.ExternalID, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting player: %w", err)
	}
	return nil
}

func (r *PlayerRepo) GetByID(ctx context.Context, id string) (*domain.Player, error) {
	q := `SELECT id, username, COALESCE(external_id,''), created_at, updated_at
	      FROM players WHERE id = $1`
	p := &domain.Player{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.Username, &p.ExternalID, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPlayerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying player by id: %w", err)
	}
	return p, nil
}
