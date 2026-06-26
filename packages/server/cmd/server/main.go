package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"co-review/server/internal/api"
	"co-review/server/internal/bootstrap"
)

func main() {

	// Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Boostraping
	cfg, database, err := bootstrap.Init()
	if err != nil {
		logger.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           api.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
}
