package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/oshy/score-gorad/internal/domain"
)

type ScoreRepo struct {
	db *sql.DB
}

func NewScoreRepo(db *sql.DB) *ScoreRepo {
	return &ScoreRepo{db: db}
}

func (r *ScoreRepo) Create(ctx context.Context, s *domain.Score) error {
	meta, err := json.Marshal(s.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	q := `INSERT INTO scores (id, game_id, player_id, season_id, points, metadata, created_at)
	      VALUES ($1, $2, $3, NULLIF($4,''), $5, $6, $7)`
	_, err = r.db.ExecContext(ctx, q,
		s.ID, s.GameID, s.PlayerID, s.SeasonID, s.Points, meta, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting score: %w", err)
	}
	return nil
}

// GetLeaderboard devuelve la mejor puntuación de cada jugador para un juego.
//
// Usa una CTE con ROW_NUMBER() en lugar de DISTINCT ON porque DISTINCT ON no
// compone bien con LIMIT/OFFSET: el planificador puede aplicar el límite antes
// de ordenar el resultado exterior, produciendo páginas incorrectas.
//
// ROW_NUMBER() particiona por player_id y ordena por (points DESC, created_at ASC),
// lo que garantiza que rn=1 siempre sea el mejor score del jugador. La query
// exterior puede paginar de forma segura sobre el conjunto ya ordenado.
//
// Si seasonID es vacío, devuelve el ranking global (sin filtrar por temporada).
func (r *ScoreRepo) GetLeaderboard(ctx context.Context, gameID, seasonID string, limit, offset int) ([]domain.LeaderboardEntry, error) {
	var q string
	var args []any

	if seasonID == "" {
		q = `
		WITH ranked AS (
		    SELECT
		        s.player_id,
		        p.username,
		        s.points,
		        s.created_at AS achieved_at,
		        ROW_NUMBER() OVER (
		            PARTITION BY s.player_id
		            ORDER BY s.points DESC, s.created_at ASC
		        ) AS rn
		    FROM scores s
		    JOIN players p ON p.id = s.player_id
		    WHERE s.game_id = $1
		)
		SELECT player_id, username, points, achieved_at
		FROM ranked
		WHERE rn = 1
		ORDER BY points DESC, achieved_at ASC
		LIMIT $2 OFFSET $3`
		args = []any{gameID, limit, offset}
	} else {
		q = `
		WITH ranked AS (
		    SELECT
		        s.player_id,
		        p.username,
		        s.points,
		        s.created_at AS achieved_at,
		        ROW_NUMBER() OVER (
		            PARTITION BY s.player_id
		            ORDER BY s.points DESC, s.created_at ASC
		        ) AS rn
		    FROM scores s
		    JOIN players p ON p.id = s.player_id
		    WHERE s.game_id = $1 AND s.season_id = $2
		)
		SELECT player_id, username, points, achieved_at
		FROM ranked
		WHERE rn = 1
		ORDER BY points DESC, achieved_at ASC
		LIMIT $3 OFFSET $4`
		args = []any{gameID, seasonID, limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("querying leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []domain.LeaderboardEntry
	rank := offset + 1
	for rows.Next() {
		var e domain.LeaderboardEntry
		if err := rows.Scan(&e.PlayerID, &e.Username, &e.Points, &e.AchievedAt); err != nil {
			return nil, fmt.Errorf("scanning leaderboard entry: %w", err)
		}
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetPlayerRank calcula la posición de un jugador en el ranking.
// Reutiliza el mismo patrón CTE de GetLeaderboard para consistencia:
// primero obtiene el mejor score de cada jugador, luego cuenta cuántos
// tienen más puntos que el jugador consultado.
func (r *ScoreRepo) GetPlayerRank(ctx context.Context, gameID, playerID, seasonID string) (int, error) {
	var q string
	var args []any

	if seasonID == "" {
		q = `
		WITH best AS (
		    SELECT player_id, points
		    FROM (
		        SELECT player_id, points,
		               ROW_NUMBER() OVER (PARTITION BY player_id ORDER BY points DESC, created_at ASC) AS rn
		        FROM scores
		        WHERE game_id = $1
		    ) sub
		    WHERE rn = 1
		),
		player_best AS (
		    SELECT points FROM best WHERE player_id = $2
		)
		SELECT COUNT(*) + 1
		FROM best, player_best
		WHERE best.points > player_best.points
		   OR (best.points = player_best.points AND best.player_id < $2)`
		args = []any{gameID, playerID}
	} else {
		q = `
		WITH best AS (
		    SELECT player_id, points
		    FROM (
		        SELECT player_id, points,
		               ROW_NUMBER() OVER (PARTITION BY player_id ORDER BY points DESC, created_at ASC) AS rn
		        FROM scores
		        WHERE game_id = $1 AND season_id = $2
		    ) sub
		    WHERE rn = 1
		),
		player_best AS (
		    SELECT points FROM best WHERE player_id = $3
		)
		SELECT COUNT(*) + 1
		FROM best, player_best
		WHERE best.points > player_best.points
		   OR (best.points = player_best.points AND best.player_id < $3)`
		args = []any{gameID, seasonID, playerID}
	}

	var rank int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&rank)
	if err != nil {
		return 0, fmt.Errorf("querying player rank: %w", err)
	}
	return rank, nil
}

// GetPlayerScores devuelve el historial de puntuaciones de un jugador.
func (r *ScoreRepo) GetPlayerScores(ctx context.Context, playerID string, limit, offset int) ([]domain.Score, error) {
	q := `
	SELECT id, game_id, player_id, COALESCE(season_id,''), points, created_at
	FROM scores
	WHERE player_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, q, playerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("querying player scores: %w", err)
	}
	defer rows.Close()

	var scores []domain.Score
	for rows.Next() {
		var s domain.Score
		if err := rows.Scan(&s.ID, &s.GameID, &s.PlayerID, &s.SeasonID, &s.Points, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning score: %w", err)
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}
