// Package daemon — daemonize.go re-executes the binary in a detached child
// process so it survives terminal close (Unix double-fork equivalent).
//
// Because Go's runtime is multi-threaded and fork(2) is unsafe after
// os.StartProcess, the idiomatic Go approach is to re-exec the binary with
// a special env flag rather than calling fork directly.
package daemon

import (
	"os"
	"os/exec"
)

const daemonEnvKey = "_FOLDER_SYNCER_DAEMON"

// IsDaemonChild returns true when the process was started by Daemonize.
func IsDaemonChild() bool {
	return os.Getenv(daemonEnvKey) == "1"
}

// Daemonize re-executes the current binary as a detached background process
// and exits the parent. The child continues running with the same arguments.
func Daemonize() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), daemonEnvKey+"=1")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// SysProcAttr is set in daemonize_unix.go (build-tag restricted).
	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}

	// Parent exits; child continues.
	os.Exit(0)
	return nil // unreachable
}
