package app

import (
	"log/slog"
	"net/http"
	"sync/atomic"
)

type Application struct {
	Logger         *slog.Logger
	IsShuttingDown atomic.Bool
}

func NewApplication(logger *slog.Logger) *Application {
	return &Application{
		Logger: logger,
	}
}

func (app *Application) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/livez", app.livenessHandler)
	mux.HandleFunc("/readyz", app.readinessHandler)

	return mux
}
