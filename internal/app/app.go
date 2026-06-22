// Package app wires all dependencies together and runs the core sync loop.
package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/sanay/go_redis_folder_syncer/internal/config"
	"github.com/sanay/go_redis_folder_syncer/internal/daemon"
	syncsvc "github.com/sanay/go_redis_folder_syncer/internal/sync"
)

func buildLogger(logFile string) (*slog.Logger, func(), error) {
	writers := []io.Writer{os.Stdout}
	var closeFuncs []func()

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		writers = append(writers, f)
		closeFuncs = append(closeFuncs, func() { _ = f.Close() })
	}

	handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	cleanup := func() {
		for _, fn := range closeFuncs {
			fn()
		}
	}
	return logger, cleanup, nil
}

func buildRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDb,
	})
}

// Run is the core sync loop: restore from Redis, then watch for changes.
// It blocks until a signal is received.
func Run(cfg *config.Config) error {
	logger, cleanup, err := buildLogger(cfg.LogFile)
	if err != nil {
		return err
	}
	defer cleanup()

	redisClient := buildRedisClient(cfg)
	defer redisClient.Close()

	// Verify Redis connectivity early.
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("cannot reach Redis at %s:%d: %w", cfg.RedisHost, cfg.RedisPort, err)
	}

	service := syncsvc.New(cfg, logger, redisClient)

	if err := daemon.WritePID(cfg.PIDFile); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	defer daemon.RemovePID(cfg.PIDFile)

	logger.Info("daemon started", "pid", os.Getpid())

	// Restore cached files before starting the watcher.
	service.LoadAndRestore()

	watcher, err := syncsvc.NewWatcher(service, logger)
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	// Register the watch dir and all sub-directories.
	if err := addRecursive(watcher, cfg.WatchDir); err != nil {
		return fmt.Errorf("watch dir: %w", err)
	}
	logger.Info("watching", "dir", cfg.WatchDir)

	done := make(chan struct{})

	// Handle termination signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		logger.Info("shutdown signal received", "signal", sig)
		close(done)
	}()

	watcher.Run(done)
	logger.Info("daemon stopped")
	return nil
}

// addRecursive adds a directory and every sub-directory to the watcher.
func addRecursive(w *syncsvc.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.Add(path)
		}
		return nil
	})
}
