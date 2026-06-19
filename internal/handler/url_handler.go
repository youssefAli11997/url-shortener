package handler

import (
	"encoding/json"
	"net/http"

	"url-shortener/internal/model"
	"url-shortener/internal/service"
)

type URLHandler struct {
	service service.URLService
}

func NewURLHandler(service service.URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

func (h *URLHandler) Encode(w http.ResponseWriter, r *http.Request) {
	var req model.EncodeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrInvalidRequestBody)
		return
	}

	shortURL, err := h.service.Encode(r.Context(), req.URL)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.EncodeResponse{
		ShortURL: shortURL,
	})
}

func (h *URLHandler) Decode(w http.ResponseWriter, r *http.Request) {
	var req model.DecodeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrInvalidRequestBody)
		return
	}

	url, err := h.service.Decode(r.Context(), req.ShortURL)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.DecodeResponse{
		URL: url,
	})
}

func (h *URLHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(
		model.HealthzResponse{
			Status: "ok",
		},
	)
}
