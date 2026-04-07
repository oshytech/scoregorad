package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/oshy/score-gorad/internal/domain"
)

func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, err error) {
	type errBody struct {
		Error string `json:"error"`
	}
	switch err {
	case domain.ErrGameNotFound, domain.ErrPlayerNotFound, domain.ErrSeasonNotFound, domain.ErrScoreNotFound:
		respondJSON(w, http.StatusNotFound, errBody{err.Error()})
	case domain.ErrSlugAlreadyExists:
		respondJSON(w, http.StatusConflict, errBody{err.Error()})
	case domain.ErrSeasonClosed, domain.ErrInvalidScore:
		respondJSON(w, http.StatusUnprocessableEntity, errBody{err.Error()})
	case domain.ErrInvalidInput:
		respondJSON(w, http.StatusBadRequest, errBody{err.Error()})
	default:
		respondJSON(w, http.StatusInternalServerError, errBody{"internal server error"})
	}
}
