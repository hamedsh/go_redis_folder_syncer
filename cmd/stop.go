package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sanay/go_redis_folder_syncer/internal/config"
	"github.com/sanay/go_redis_folder_syncer/internal/daemon"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	RunE:  runStop,
}

func runStop(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(".env", nil)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pid, err := daemon.ReadPID(cfg.PIDFile)
	if err != nil {
		return fmt.Errorf("read PID file: %w", err)
	}

	if !daemon.IsRunning(pid) {
		fmt.Fprintln(os.Stderr, "daemon is not running")
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	fmt.Printf("sent SIGTERM to PID %d\n", pid)
	return nil
}
