package cmd

import (
	"fmt"
	"os"

	"github.com/sanay/go_redis_folder_syncer/internal/app"
	"github.com/sanay/go_redis_folder_syncer/internal/config"
	"github.com/sanay/go_redis_folder_syncer/internal/daemon"
	"github.com/spf13/cobra"
)

var (
	flagForeground     bool
	flagWatchDir       string
	flagRedisHost      string
	flagRedisPort      int
	flagRedisPassword  string
	flagRedisDb        int
	flagRedisKeyPrefix string
	flagPIDFile        string
	flagLogFile        string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the syncer (background daemon by default)",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().BoolVar(&flagForeground, "fg", false, "run in the foreground instead of daemonizing")
	startCmd.Flags().StringVar(&flagWatchDir, "watch-dir", "", "directory to watch (required)")
	startCmd.Flags().StringVar(&flagRedisHost, "redis-host", "", "Redis hostname")
	startCmd.Flags().IntVar(&flagRedisPort, "redis-port", 0, "Redis port")
	startCmd.Flags().StringVar(&flagRedisPassword, "redis-password", "", "Redis Password")
	startCmd.Flags().IntVar(&flagRedisDb, "redis-db-num", 0, "Redis DB Num")
	startCmd.Flags().StringVar(&flagRedisKeyPrefix, "redis-key-prefix", "", "Redis key prefix")
	startCmd.Flags().StringVar(&flagPIDFile, "pid-file", "", "PID file path")
	startCmd.Flags().StringVar(&flagLogFile, "log-file", "", "log file path")
}

func runStart(_ *cobra.Command, _ []string) error {
	overrides := &config.Config{
		WatchDir:       flagWatchDir,
		RedisHost:      flagRedisHost,
		RedisPort:      flagRedisPort,
		RedisPassword:  flagRedisPassword,
		RedisDb:        flagRedisDb,
		RedisKeyPrefix: flagRedisKeyPrefix,
		PIDFile:        flagPIDFile,
		LogFile:        flagLogFile,
	}

	cfg, err := config.Load(".env", overrides)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.WatchDir == "" {
		return fmt.Errorf("--watch-dir is required")
	}

	// Guard against a double-start.
	pid, err := daemon.ReadPID(cfg.PIDFile)
	if err == nil && daemon.IsRunning(pid) {
		fmt.Fprintf(os.Stderr, "already running (PID %d)\n", pid)
		os.Exit(1)
	}

	// Daemonize unless --fg was passed or we are already the daemon child.
	if !flagForeground && !daemon.IsDaemonChild() {
		if err := daemon.Daemonize(); err != nil {
			return fmt.Errorf("daemonize: %w", err)
		}
		// parent exits inside Daemonize(); child continues below
	}

	return app.Run(cfg)
}
