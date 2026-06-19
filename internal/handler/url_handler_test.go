package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/model"
	"url-shortener/internal/service"
)

func setupHandler(service service.URLService) *URLHandler {
	return NewURLHandler(service)
}

// TODO(ya): consider using testify/require for simpler test functions

func TestURLHandler_Encode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		mockResponse   string
		mockError      error
		expectedStatus int
		expectShortURL string
	}{
		{
			name: "success",
			requestBody: model.EncodeRequest{
				URL: "https://google.com",
			},
			mockResponse:   "http://short/abc",
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectShortURL: "http://short/abc",
		},
		{
			name:           "invalid json",
			requestBody:    "invalid-json",
			mockResponse:   "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid url",
			requestBody: model.EncodeRequest{
				URL: "invalid",
			},
			mockResponse:   "",
			mockError:      model.ErrInvalidURL,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unexpected encode error",
			requestBody: model.EncodeRequest{
				URL: "https://google.com",
			},
			mockResponse:   "",
			mockError:      fmt.Errorf("failed to encode"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupHandler(&service.MockURLService{
				EncodeFunc: func(ctx context.Context, originalURL string) (string, error) {
					return tt.mockResponse, tt.mockError
				},
			})

			var body []byte
			var err error

			switch v := tt.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				body, err = json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/encode", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Encode(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected %d got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp model.EncodeResponse
				_ = json.NewDecoder(rec.Body).Decode(&resp)

				if resp.ShortURL != tt.expectShortURL {
					t.Errorf("expected %s got %s", tt.expectShortURL, resp.ShortURL)
				}
			}
		})
	}
}

func TestURLHandler_Decode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		mockResponse   string
		mockError      error
		expectedStatus int
		expectURL      string
	}{
		{
			name: "success",
			requestBody: model.DecodeRequest{
				ShortURL: "http://short/abc",
			},
			mockResponse:   "https://google.com",
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectURL:      "https://google.com",
		},
		{
			name:           "invalid json",
			requestBody:    "bad-json",
			mockResponse:   "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},

		{
			name: "invalid url",
			requestBody: model.DecodeRequest{
				ShortURL: "bad-short-url",
			},
			mockResponse:   "",
			mockError:      model.ErrInvalidURL,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "url not found",
			requestBody: model.DecodeRequest{
				ShortURL: "http://short/missing",
			},
			mockResponse:   "",
			mockError:      model.ErrURLNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "unexpected decode error",
			requestBody: model.DecodeRequest{
				ShortURL: "http://short/abc",
			},
			mockResponse:   "",
			mockError:      fmt.Errorf("failed to decode"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupHandler(&service.MockURLService{
				DecodeFunc: func(ctx context.Context, originalURL string) (string, error) {
					return tt.mockResponse, tt.mockError
				},
			})

			var body []byte
			var err error

			switch v := tt.requestBody.(type) {
			case string:
				body = []byte(v)
			default:
				body, err = json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/decode", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Decode(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected %d got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp model.DecodeResponse
				_ = json.NewDecoder(rec.Body).Decode(&resp)

				if resp.URL != tt.expectURL {
					t.Errorf("expected %s got %s", tt.expectURL, resp.URL)
				}
			}
		})
	}
}
