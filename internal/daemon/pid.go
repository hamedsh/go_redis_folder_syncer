// Package daemon provides process management utilities (PID file, daemonization).
package daemon

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// WritePID writes the current process ID to pidFile, creating or
// overwriting it.
func WritePID(pidFile string) error {
	return os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// ReadPID reads and parses the PID stored in pidFile.
// Returns 0 and no error when the file does not exist or is empty.
func ReadPID(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	pidStr := strings.TrimSpace(string(data))
	if pidStr == "" {
		return 0, nil
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("corrupt PID file %q: %w", pidFile, err)
	}

	return pid, nil
}

// IsRunning returns true when a process with the given PID exists.
func IsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without sending a real signal.
	return proc.Signal(syscall.Signal(0)) == nil
}

// RemovePID removes the PID file, ignoring "not found" errors.
func RemovePID(pidFile string) {
	if err := os.Remove(pidFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		// non-fatal — log if a logger is handy, otherwise ignore
		_ = err
	}
}
