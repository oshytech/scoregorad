package domain

import "context"

// GameRepository define las operaciones de persistencia para Game.
// La interfaz vive en el dominio; la implementación, en el paquete postgres.
type GameRepository interface {
	Create(ctx context.Context, game *Game) error
	GetByID(ctx context.Context, id string) (*Game, error)
	GetBySlug(ctx context.Context, slug string) (*Game, error)
	List(ctx context.Context) ([]Game, error)
}

// PlayerRepository define las operaciones de persistencia para Player.
type PlayerRepository interface {
	Create(ctx context.Context, player *Player) error
	GetByID(ctx context.Context, id string) (*Player, error)
}

// ScoreRepository define las operaciones de persistencia para Score y Leaderboard.
type ScoreRepository interface {
	Create(ctx context.Context, score *Score) error
	GetLeaderboard(ctx context.Context, gameID string, seasonID string, limit, offset int) ([]LeaderboardEntry, error)
	GetPlayerRank(ctx context.Context, gameID string, playerID string, seasonID string) (int, error)
	GetPlayerScores(ctx context.Context, playerID string, limit, offset int) ([]Score, error)
}

// SeasonRepository define las operaciones de persistencia para Season.
type SeasonRepository interface {
	Create(ctx context.Context, season *Season) error
	GetByID(ctx context.Context, id string) (*Season, error)
	ListByGame(ctx context.Context, gameID string) ([]Season, error)
}
