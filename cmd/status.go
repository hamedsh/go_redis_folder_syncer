package cmd

import (
	"fmt"

	"github.com/sanay/go_redis_folder_syncer/internal/config"
	"github.com/sanay/go_redis_folder_syncer/internal/daemon"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check whether the daemon is running",
	RunE:  runStatus,
}

func runStatus(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(".env", nil)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pid, err := daemon.ReadPID(cfg.PIDFile)
	if err != nil {
		return fmt.Errorf("read PID file: %w", err)
	}

	if daemon.IsRunning(pid) {
		fmt.Printf("running (PID %d)\n", pid)
	} else {
		fmt.Println("not running")
	}
	return nil
}
