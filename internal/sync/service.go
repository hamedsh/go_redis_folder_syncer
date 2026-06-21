// Package sync provides the Redis-backed file synchronisation service.
package sync

import (
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sanay/go_redis_folder_syncer/internal/config"
)

// RedisClient is the subset of the Redis client API used by Service.
// Defining an interface makes the service trivially testable with a mock.
type RedisClient interface {
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
}

// Service encapsulates all Redis sync operations.
type Service struct {
	cfg    *config.Config
	logger *slog.Logger
	redis  RedisClient
	ctx    context.Context
}

// New constructs a Service with its dependencies injected.
func New(cfg *config.Config, logger *slog.Logger, client RedisClient) *Service {
	return &Service{
		cfg:    cfg,
		logger: logger,
		redis:  client,
		ctx:    context.Background(),
	}
}

// redisKey returns the Redis key for the given file path.
func (s *Service) redisKey(filePath string) string {
	return s.cfg.RedisKeyPrefix + filepath.Clean(filepath.ToSlash(filePath))
}

// SyncFile upserts file metadata and base64-encoded content into Redis.
func (s *Service) SyncFile(filePath string) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		s.logger.Error("failed to resolve path", "path", filePath, "err", err)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return // file already gone — nothing to sync
		}
		s.logger.Error("stat failed", "path", absPath, "err", err)
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		s.logger.Error("sync failed — cannot read file", "path", absPath, "err", err)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	key := s.redisKey(absPath)

	mtime := strconv.FormatFloat(float64(info.ModTime().Unix()), 'f', 1, 64)
	if err := s.redis.HSet(s.ctx, key,
		"filename", filepath.Base(absPath),
		"size_bytes", strconv.FormatInt(info.Size(), 10),
		"mtime", mtime,
		"content_stub", encoded,
	).Err(); err != nil {
		s.logger.Error("redis HSet failed", "path", absPath, "err", err)
		return
	}

	s.logger.Info("synced", "path", absPath)
}

// RemoveFile deletes the Redis key associated with a locally-removed file.
func (s *Service) RemoveFile(filePath string) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		s.logger.Error("failed to resolve path", "path", filePath, "err", err)
		return
	}

	key := s.redisKey(absPath)
	if err := s.redis.Del(s.ctx, key).Err(); err != nil {
		s.logger.Error("redis Del failed", "path", absPath, "err", err)
		return
	}

	s.logger.Info("purged from redis", "path", absPath)
}

// LoadAndRestore restores every cached file from Redis back to disk on startup.
func (s *Service) LoadAndRestore() {
	s.logger.Info("restoring files from redis…")

	if err := os.MkdirAll(s.cfg.WatchDir, 0o755); err != nil {
		s.logger.Error("failed to create watch dir", "path", s.cfg.WatchDir, "err", err)
		return
	}

	pattern := s.cfg.RedisKeyPrefix + "*"
	keys, err := s.redis.Keys(s.ctx, pattern).Result()
	if err != nil {
		s.logger.Error("redis Keys failed", "pattern", pattern, "err", err)
		return
	}

	if len(keys) == 0 {
		s.logger.Info("no cached files found — starting fresh")
		return
	}

	for _, key := range keys {
		physicalPath := key[len(s.cfg.RedisKeyPrefix):]
		s.restoreOne(key, physicalPath)
	}

	s.logger.Info("restore complete")
}

func (s *Service) restoreOne(key, physicalPath string) {
	metadata, err := s.redis.HGetAll(s.ctx, key).Result()
	if err != nil || len(metadata) == 0 {
		return
	}

	cachedSize, _ := strconv.ParseInt(metadata["size_bytes"], 10, 64)

	// Skip if the on-disk file already matches the cached size.
	if info, err := os.Stat(physicalPath); err == nil {
		if info.Size() == cachedSize {
			s.logger.Info("up-to-date, skipping", "path", physicalPath)
			return
		}
	}

	if err := os.MkdirAll(filepath.Dir(physicalPath), 0o755); err != nil {
		s.logger.Error("failed to create parent dirs", "path", physicalPath, "err", err)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(metadata["content_stub"])
	if err != nil {
		s.logger.Error("base64 decode failed", "path", physicalPath, "err", err)
		return
	}

	if err := os.WriteFile(physicalPath, decoded, 0o644); err != nil {
		s.logger.Error("failed to restore file", "path", physicalPath, "err", err)
		return
	}

	s.logger.Info("restored", "path", physicalPath)

	// Restore the original modification time.
	if mtimeStr, ok := metadata["mtime"]; ok {
		if mtimeF, err := strconv.ParseFloat(mtimeStr, 64); err == nil {
			mtime := time.Unix(int64(mtimeF), 0)
			_ = os.Chtimes(physicalPath, time.Now(), mtime)
		}
	}
}
