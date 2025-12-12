package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"go-channel/internal/api/httpapi"
	"go-channel/internal/config"
	"go-channel/internal/logging"
	"go-channel/internal/queue"
	"go-channel/internal/types"
	"go-channel/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	logger, err := logging.New(cfg.Env)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	q := queue.NewMemoryQueue(cfg.QueueSize)

	pool := worker.NewPool(q, demoHandler(logger), worker.Config{
		Concurrency: cfg.WorkerCount,
		JobTimeout:  cfg.JobTimeout,
		Backoff: worker.BackoffConfig{
			Base:   500 * time.Millisecond,
			Max:    5 * time.Second,
			Jitter: 250 * time.Millisecond,
		},
	}, logger)
	if err := pool.Start(ctx); err != nil {
		logger.Fatal("failed to start worker pool", zap.Error(err))
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpapi.New(q, logger),
	}

	go func() {
		logger.Info("http server starting", zap.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown error", zap.Error(err))
	}
	pool.Stop()
	q.Close()
	logger.Info("shutdown complete")
}

// demoHandler shows how a worker might process a job.
func demoHandler(log *zap.Logger) worker.Handler {
	return func(ctx context.Context, job types.Job) error {
		// Simple demo: log payload and simulate work.
		log.Info("processing job", zap.String("job_id", job.ID), zap.String("payload", job.Payload), zap.Int("attempt", job.Attempts))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(750 * time.Millisecond):
			return nil
		}
	}
}

