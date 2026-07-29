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

	"github.com/Basith-08/tracklume-api/internal/app"
	"github.com/Basith-08/tracklume-api/internal/config"
	"github.com/Basith-08/tracklume-api/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := database.ConnectWithRetry(ctx, cfg.DatabaseDSN(), 30)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	server := &http.Server{Addr: ":" + cfg.AppPort, Handler: app.NewRouter(cfg, pool, logger), ReadTimeout: cfg.RequestTimeout, WriteTimeout: cfg.RequestTimeout, IdleTimeout: 60 * time.Second, ReadHeaderTimeout: 5 * time.Second}
	serverErr := make(chan error, 1)
	go func() { logger.Info("api listening", "port", cfg.AppPort); serverErr <- server.ListenAndServe() }()
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case err = <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	case <-signalCtx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		logger.Info("shutting down")
		if err = server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("shutdown complete")
	}
}
