package domain

import "time"

// Game representa un videojuego o modo competitivo.
type Game struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Player representa a un jugador.
type Player struct {
	ID         string
	Username   string
	ExternalID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Season representa una temporada competitiva.
type Season struct {
	ID        string
	GameID    string
	Name      string
	StartsAt  time.Time
	EndsAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Score representa una puntuación enviada por un jugador.
type Score struct {
	ID       string
	GameID   string
	PlayerID string
	SeasonID string
	Points   int64
	Metadata map[string]any
	CreatedAt time.Time
}

// LeaderboardEntry representa una entrada en el ranking, calculada a partir de scores.
// No es una tabla física — es una proyección.
type LeaderboardEntry struct {
	Rank      int
	PlayerID  string
	Username  string
	Points    int64
	AchievedAt time.Time
}
