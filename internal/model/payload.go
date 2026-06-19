package model

type EncodeRequest struct {
	URL string `json:"url"`
}

type EncodeResponse struct {
	ShortURL string `json:"short_url"`
}

type DecodeRequest struct {
	ShortURL string `json:"short_url"`
}

type DecodeResponse struct {
	URL string `json:"url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthzResponse struct {
	Status string `json:"status"`
}
