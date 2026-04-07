package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/oshy/score-gorad/internal/domain"
	"github.com/oshy/score-gorad/internal/service"
)

type GameHandler struct {
	svc *service.GameService
}

func NewGameHandler(svc *service.GameService) *GameHandler {
	return &GameHandler{svc: svc}
}

func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	game, err := h.svc.CreateGame(r.Context(), service.CreateGameInput{
		Name: body.Name,
		Slug: body.Slug,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, game)
}

func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("gameId")
	game, err := h.svc.GetGame(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, game)
}

func (h *GameHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	games, err := h.svc.ListGames(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	if games == nil {
		games = []domain.Game{}
	}
	respondJSON(w, http.StatusOK, games)
}
