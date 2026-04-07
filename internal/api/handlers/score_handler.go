package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/oshy/score-gorad/internal/domain"
	"github.com/oshy/score-gorad/internal/service"
)

type ScoreHandler struct {
	svc *service.ScoreService
}

func NewScoreHandler(svc *service.ScoreService) *ScoreHandler {
	return &ScoreHandler{svc: svc}
}

func (h *ScoreHandler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("gameId")

	var body struct {
		PlayerID string         `json:"playerId"`
		SeasonID string         `json:"seasonId"`
		Points   int64          `json:"score"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.PlayerID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "playerId is required"})
		return
	}

	score, err := h.svc.SubmitScore(r.Context(), service.SubmitScoreInput{
		GameID:   gameID,
		PlayerID: body.PlayerID,
		SeasonID: body.SeasonID,
		Points:   body.Points,
		Metadata: body.Metadata,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, score)
}

func (h *ScoreHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("gameId")
	seasonID := r.PathValue("seasonId") // vacío si la ruta es /leaderboard sin temporada

	limit, offset := parsePagination(r)

	entries, err := h.svc.GetLeaderboard(r.Context(), gameID, seasonID, limit, offset)
	if err != nil {
		respondError(w, err)
		return
	}
	if entries == nil {
		entries = []domain.LeaderboardEntry{}
	}
	respondJSON(w, http.StatusOK, entries)
}

func (h *ScoreHandler) GetPlayerRank(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("gameId")
	playerID := r.PathValue("playerId")
	seasonID := r.URL.Query().Get("season")

	rank, err := h.svc.GetPlayerRank(r.Context(), gameID, playerID, seasonID)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]int{"rank": rank})
}

func (h *ScoreHandler) GetPlayerScores(w http.ResponseWriter, r *http.Request) {
	playerID := r.PathValue("playerId")
	limit, offset := parsePagination(r)

	scores, err := h.svc.GetPlayerScores(r.Context(), playerID, limit, offset)
	if err != nil {
		respondError(w, err)
		return
	}
	if scores == nil {
		scores = []domain.Score{}
	}
	respondJSON(w, http.StatusOK, scores)
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 25
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}
