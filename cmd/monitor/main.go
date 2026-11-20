package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"uptime-monitor/internal/api"
	"uptime-monitor/internal/checker"
	"uptime-monitor/internal/config"
	"uptime-monitor/internal/database"
	"uptime-monitor/internal/monitor"
	"uptime-monitor/internal/scheduler"
	"uptime-monitor/internal/worker"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get("http://127.0.0.1:8080/healthz")
		if err != nil {
			os.Exit(1)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	logger.Info(
		"database connected successfully",
		"environment", cfg.AppEnv,
	)

	repo := monitor.NewRepository(pool)
	workers, err := worker.NewPool(4, checker.NewHttpChecker())
	if err != nil {
		return err
	}
	runner, err := scheduler.New(repo, workers, scheduler.Config{
		PollInterval: time.Second, BatchSize: 4, LeaseDuration: time.Minute,
	}, func(_ context.Context, result worker.Result) error {
		// Persist completed results even when shutdown has canceled the scheduler.
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer saveCancel()
		c := result.Check
		return repo.Complete(saveCtx, result.Monitor.ID, c.StatusCode, c.ResponseTime, c.IsSuccess, c.ErrorType, c.ErrorMessage, c.StartedAt, c.FinishedAt)
	})
	if err != nil {
		return err
	}
	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: api.New(pool), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute}
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		server.Addr = addr
	}
	serverDone := make(chan struct{})
	var serverErr error
	go func() {
		defer close(serverDone)
		logger.Info("HTTP server started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr = err
			logger.Error("HTTP server failed", "error", err)
			stop()
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP shutdown failed", "error", err)
			_ = server.Close()
		}
		<-serverDone
		if runErr == nil {
			runErr = serverErr
		}
	}()
	logger.Info("uptime monitor started")
	if err := runner.Run(runCtx); err != nil {
		return err
	}

	return nil
}
