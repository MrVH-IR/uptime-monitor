package main

import (
	"context"
	"log/slog"
	"os"
	"time"
	"uptime-monitor/internal/config"
	"uptime-monitor/internal/database"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	logger.Info(
		"database connected successfully",
		"environment", cfg.AppEnv,
	)

	logger.Info("uptime monitor started")
}
