package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"url-shortener/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, model.ErrorResponse{
		Error: err.Error(),
	})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrInvalidURL):
		writeError(w, http.StatusBadRequest, model.ErrInvalidURL)
	case errors.Is(err, model.ErrURLNotFound):
		writeError(w, http.StatusNotFound, model.ErrURLNotFound)
	default:
		writeError(w, http.StatusInternalServerError, model.ErrInternalServerError)
	}
}
