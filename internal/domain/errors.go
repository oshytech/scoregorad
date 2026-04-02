package domain

import "errors"

var (
	ErrGameNotFound     = errors.New("game not found")
	ErrPlayerNotFound   = errors.New("player not found")
	ErrScoreNotFound    = errors.New("score not found")
	ErrSeasonNotFound   = errors.New("season not found")
	ErrSeasonClosed     = errors.New("season is closed, scores are no longer accepted")
	ErrInvalidScore     = errors.New("score must be a non-negative value")
	ErrInvalidInput     = errors.New("invalid input")
	ErrSlugAlreadyExists = errors.New("slug already exists")
)
