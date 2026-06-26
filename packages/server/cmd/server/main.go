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
	"co-review/server/internal/events"
	"co-review/server/internal/platform"
	"co-review/server/internal/provider"
	"co-review/server/internal/reviews"
	"co-review/server/internal/skills"
	skillassets "co-review/server/skills"
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

	loadedSkills, err := skills.LoadFS(skillassets.FS, ".")
	if err != nil {
		logger.Error("load review skills failed", "error", err)
		os.Exit(1)
	}
	gitlabClient, err := platform.NewGitLabClient(platform.GitLabConfig{TokenEnv: "CO_REVIEW_GITLAB_TOKEN"})
	if err != nil {
		logger.Error("configure GitLab client failed", "error", err)
		os.Exit(1)
	}
	broker := events.NewBroker()
	reviewService := &reviews.Service{Repo: reviews.NewRepository(database), Platform: gitlabClient, Provider: provider.DeterministicReviewProvider{}, Skills: loadedSkills, Broker: broker}

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           api.NewRouterWithDeps(api.RouterDeps{Reviews: reviewService, Broker: broker}),
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
