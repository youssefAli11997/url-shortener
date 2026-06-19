package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"url-shortener/internal/config"
	"url-shortener/internal/handler"
	"url-shortener/internal/middleware"
	"url-shortener/internal/repository"
	"url-shortener/internal/service"
)

type App struct {
	server *http.Server
	db     *pgxpool.Pool
}

func NewApp(ctx context.Context, config *config.Config) (*App, error) {

	db, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}

	repository := repository.NewPostgresURLRepository(db)
	service := service.NewURLService(repository, config.BaseURL)
	handler := handler.NewURLHandler(service)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /encode", handler.Encode)
	mux.HandleFunc("POST /decode", handler.Decode)
	mux.HandleFunc("GET /healthz", handler.Healthz)

	serverHandler := middleware.Recover(
		middleware.Logging(mux),
	)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Port),
		Handler: serverHandler,
	}

	return &App{
		server: server,
		db:     db,
	}, nil
}

func (a *App) Start() error {
	log.Printf("server listening on %s", a.server.Addr)

	err := a.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	defer a.db.Close()
	return a.server.Shutdown(ctx)
}

func (a *App) Handler() http.Handler {
	return a.server.Handler
}
