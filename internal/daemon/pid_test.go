package daemon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sanay/go_redis_folder_syncer/internal/daemon"
)

func TestWriteAndReadPID_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "test.pid")

	if err := daemon.WritePID(pidFile); err != nil {
		t.Fatalf("WritePID: %v", err)
	}

	pid, err := daemon.ReadPID(pidFile)
	if err != nil {
		t.Fatalf("ReadPID: %v", err)
	}
	if pid != os.Getpid() {
		t.Errorf("got PID %d, want %d", pid, os.Getpid())
	}
}

func TestReadPID_MissingFileReturnsZero(t *testing.T) {
	tmp := t.TempDir()
	pid, err := daemon.ReadPID(filepath.Join(tmp, "no.pid"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pid != 0 {
		t.Errorf("expected 0, got %d", pid)
	}
}

func TestReadPID_CorruptFileReturnsError(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "bad.pid")
	_ = os.WriteFile(pidFile, []byte("not-a-number"), 0o644)

	_, err := daemon.ReadPID(pidFile)
	if err == nil {
		t.Error("expected error for corrupt PID file")
	}
}

func TestWritePID_OverwritesExisting(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "overwrite.pid")
	_ = os.WriteFile(pidFile, []byte("99999"), 0o644)

	if err := daemon.WritePID(pidFile); err != nil {
		t.Fatalf("WritePID: %v", err)
	}
	pid, _ := daemon.ReadPID(pidFile)
	if pid != os.Getpid() {
		t.Errorf("got %d, want %d", pid, os.Getpid())
	}
}

func TestIsRunning_CurrentProcess(t *testing.T) {
	if !daemon.IsRunning(os.Getpid()) {
		t.Error("expected current process to be running")
	}
}

func TestIsRunning_ZeroPIDReturnsFalse(t *testing.T) {
	if daemon.IsRunning(0) {
		t.Error("PID 0 should not report as running")
	}
}

func TestIsRunning_NegativePIDReturnsFalse(t *testing.T) {
	if daemon.IsRunning(-1) {
		t.Error("negative PID should not report as running")
	}
}
