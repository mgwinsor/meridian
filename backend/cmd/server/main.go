package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mgwinsor/meridian/backend/internal/app"
)

func main() {
	const port = 8080
	const env = "dev"

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	app := app.NewApplication(logger)

	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: app.Routes(),
	}

	errChan := make(chan error, 1)

	logger.Info("Starting server", "port", port, "env", env)
	go func() {
		errChan <- httpServer.ListenAndServe()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server encountered a fatal runtime error", "error", err)
			os.Exit(1)
		}
	case s := <-signalChan:
		logger.Info("Captured shutdown signal", "signal", s.String())

		app.IsShuttingDown.Store(true)

		const loadBalancerDeregistrationWindow = 5 * time.Second
		time.Sleep(loadBalancerDeregistrationWindow)

		const serverDrainTimeout = 10 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), serverDrainTimeout)
		defer cancel()

		logger.Info("Shutting down HTTP server gracefully...")
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error("Graceful shutdown failed to clear active connections", "error", err)
			os.Exit(1)
		}

		logger.Info("Server stopped cleanly. Goodbye.")
	}
}
