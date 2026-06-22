// Package config loads and exposes application settings.
// Values are read from environment variables (with .env file support),
// and can be overridden by flags passed via the CLI layer.
package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds every tuneable for the syncer.
type Config struct {
	WatchDir       string // directory to monitor (required)
	RedisHost      string // Redis hostname
	RedisPort      int    // Redis port
	RedisPassword  string // Redis password
	RedisDb        int    // Redis DB num
	RedisKeyPrefix string // prefix for every Redis key
	PIDFile        string // path to the PID file
	LogFile        string // path to the log file (empty = stdout only)
}

// Load reads settings from the environment (and an optional .env file).
// Explicit non-zero values in overrides take precedence over env vars.
func Load(envFile string, overrides *Config) (*Config, error) {
	// best-effort: ignore "file not found"
	_ = godotenv.Load(envFile)

	env := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	envInt := func(key string, fallback int) int {
		if v := os.Getenv(key); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return fallback
	}

	cfg := &Config{
		WatchDir:       env("SYNC_WATCH_DIR", ""),
		RedisHost:      env("SYNC_REDIS_HOST", "localhost"),
		RedisPort:      envInt("SYNC_REDIS_PORT", 6379),
		RedisPassword:  env("SYNC_REDIS_PASSWORD", ""),
		RedisDb:        envInt("SYNC_REDIS_DB", 0),
		RedisKeyPrefix: env("SYNC_REDIS_KEY_PREFIX", "file_cache:"),
		PIDFile:        env("SYNC_PID_FILE", "/tmp/folder_syncer.pid"),
		LogFile:        env("SYNC_LOG_FILE", "/tmp/folder_syncer.log"),
	}

	mergeOverrides(cfg, overrides)
	return cfg, nil
}

// mergeOverrides copies non-zero fields from src into dst.
func mergeOverrides(dst, src *Config) {
	if src == nil {
		return
	}
	if src.WatchDir != "" {
		dst.WatchDir = src.WatchDir
	}
	if src.RedisHost != "" {
		dst.RedisHost = src.RedisHost
	}
	if src.RedisPort != 0 {
		dst.RedisPort = src.RedisPort
	}
	if src.RedisPassword != "" {
		dst.RedisPassword = src.RedisPassword
	}
	if src.RedisDb != 0 {
		dst.RedisDb = src.RedisDb
	}
	if src.RedisKeyPrefix != "" {
		dst.RedisKeyPrefix = src.RedisKeyPrefix
	}
	if src.PIDFile != "" {
		dst.PIDFile = src.PIDFile
	}
	if src.LogFile != "" {
		dst.LogFile = src.LogFile
	}
}
