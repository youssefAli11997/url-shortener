//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"url-shortener/internal/app"
	"url-shortener/internal/config"
	"url-shortener/internal/model"
)

func setupApp(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	ctx := context.Background()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate integration test file")
	}

	migrationDir := filepath.Join(filepath.Dir(file), "..", "migrations")

	pg, err := postgres.Run(
		ctx,
		"postgres:17",
		postgres.WithDatabase("url_shortener"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatal(err)
	}

	dbURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	m, err := migrate.New(
		"file://"+migrationDir,
		dbURL,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatal(err)
	}

	cfg := config.Config{
		Port:        "8080",
		BaseURL:     "http://localhost:8080",
		DatabaseURL: dbURL,
	}

	a, err := app.NewApp(ctx, &cfg)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(a.Handler())

	cleanup := func() {
		server.Close()
		if err := a.Shutdown(ctx); err != nil {
			t.Errorf("failed to shut down app: %v", err)
		}
		if err := pg.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate postgres container: %v", err)
		}
	}

	return server, cleanup
}

func TestHealthz(t *testing.T) {
	server, cleanup := setupApp(t)
	defer cleanup()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
}

func TestEncodeDecode(t *testing.T) {
	server, cleanup := setupApp(t)
	defer cleanup()

	encodeReqBody := model.EncodeRequest{
		URL: "https://google.com",
	}

	body, err := json.Marshal(encodeReqBody)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(
		server.URL+"/encode",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected encode status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var encodeResp model.EncodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&encodeResp); err != nil {
		t.Fatal(err)
	}

	if encodeResp.ShortURL == "" {
		t.Fatal("expected short URL, got empty string")
	}

	decodeReqBody := model.DecodeRequest{
		ShortURL: encodeResp.ShortURL,
	}

	body, err = json.Marshal(decodeReqBody)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = http.Post(
		server.URL+"/decode",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected decode status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var decodeResp model.DecodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&decodeResp); err != nil {
		t.Fatal(err)
	}

	if decodeResp.URL != encodeReqBody.URL {
		t.Fatalf("expected %q, got %q", encodeReqBody.URL, decodeResp.URL)
	}
}

func TestEncodeIdempotency(t *testing.T) {
	server, cleanup := setupApp(t)
	defer cleanup()

	client := &http.Client{}

	encode := func() string {
		encodeReqBody := model.EncodeRequest{
			URL: "https://google.com",
		}

		body, err := json.Marshal(encodeReqBody)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.Post(
			server.URL+"/encode",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var res model.EncodeResponse

		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatal(err)
		}

		return res.ShortURL
	}

	first := encode()
	second := encode()

	if first != second {
		t.Fatalf(
			"expected idempotent encoding, got %s and %s",
			first,
			second,
		)
	}
}

func TestErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   string
		body       string
		statusCode int
	}{
		{
			name:       "invalid url",
			endpoint:   "/encode",
			body:       `{"url":"invalid"}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "decode not found",
			endpoint:   "/decode",
			body:       `{"short_url":"http://localhost:8080/abc"}`,
			statusCode: http.StatusNotFound,
		},
	}

	server, cleanup := setupApp(t)
	defer cleanup()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			resp, err := http.Post(
				server.URL+tt.endpoint,
				"application/json",
				strings.NewReader(tt.body),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.statusCode {
				t.Fatalf(
					"expected %d got %d",
					tt.statusCode,
					resp.StatusCode,
				)
			}

			var response map[string]string

			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}

			if response["error"] == "" {
				t.Fatal("expected error message")
			}
		})
	}
}
