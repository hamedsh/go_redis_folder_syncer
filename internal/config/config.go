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

	cfg := &Config{
		WatchDir:       getEnv("SYNC_WATCH_DIR", ""),
		RedisHost:      getEnv("SYNC_REDIS_HOST", "localhost"),
		RedisPort:      getEnvInt("SYNC_REDIS_PORT", 6379),
		RedisPassword:  getEnv("SYNC_REDIS_PASSWORD", ""),
		RedisDb:        getEnvInt("SYNC_REDIS_DB", 0),
		RedisKeyPrefix: getEnv("SYNC_REDIS_KEY_PREFIX", "file_cache:"),
		PIDFile:        getEnv("SYNC_PID_FILE", "/tmp/folder_syncer.pid"),
		LogFile:        getEnv("SYNC_LOG_FILE", "/tmp/folder_syncer.log"),
	}

	if overrides == nil {
		return cfg, nil
	}

	// CLI-provided values win over env-file values.
	if overrides.WatchDir != "" {
		cfg.WatchDir = overrides.WatchDir
	}
	if overrides.RedisHost != "" {
		cfg.RedisHost = overrides.RedisHost
	}
	if overrides.RedisPort != 0 {
		cfg.RedisPort = overrides.RedisPort
	}
	if overrides.RedisDb != 0 {
		cfg.RedisDb = overrides.RedisDb
	}
	if overrides.RedisPassword != "" {
		cfg.RedisPassword = overrides.RedisPassword
	}
	if overrides.RedisKeyPrefix != "" {
		cfg.RedisKeyPrefix = overrides.RedisKeyPrefix
	}
	if overrides.PIDFile != "" {
		cfg.PIDFile = overrides.PIDFile
	}
	if overrides.LogFile != "" {
		cfg.LogFile = overrides.LogFile
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
