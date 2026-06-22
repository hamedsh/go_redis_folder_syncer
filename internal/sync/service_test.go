package sync_test

import (
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"github.com/redis/go-redis/v9"
	"github.com/sanay/go_redis_folder_syncer/internal/config"
	syncsvc "github.com/sanay/go_redis_folder_syncer/internal/sync"
)

// ---------------------------------------------------------------------------
// Mock Redis client
// ---------------------------------------------------------------------------

type mockRedis struct {
	store map[string]map[string]string
	keys_ []string // injected for Keys() responses
}

func newMockRedis() *mockRedis {
	return &mockRedis{store: make(map[string]map[string]string)}
}

func (m *mockRedis) HSet(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	if m.store[key] == nil {
		m.store[key] = make(map[string]string)
	}
	for i := 0; i+1 < len(values); i += 2 {
		m.store[key][values[i].(string)] = values[i+1].(string)
	}
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(1)
	return cmd
}

func (m *mockRedis) HGetAll(_ context.Context, key string) *redis.MapStringStringCmd {
	cmd := redis.NewMapStringStringCmd(context.Background())
	if v, ok := m.store[key]; ok {
		cmd.SetVal(v)
	} else {
		cmd.SetVal(map[string]string{})
	}
	return cmd
}

func (m *mockRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	for _, k := range keys {
		delete(m.store, k)
	}
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(int64(len(keys)))
	return cmd
}

func (m *mockRedis) Keys(_ context.Context, _ string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(context.Background())
	cmd.SetVal(m.keys_)
	return cmd
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newService(t *testing.T, watchDir string, mock *mockRedis) *syncsvc.Service {
	t.Helper()
	cfg := &config.Config{
		WatchDir:       watchDir,
		RedisKeyPrefix: "file_cache:",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return syncsvc.New(cfg, logger, mock)
}

// ---------------------------------------------------------------------------
// SyncFile
// ---------------------------------------------------------------------------

func TestSyncFile_StoresCorrectFields(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "hello.txt")
	content := []byte("hello world")
	if err := os.WriteFile(f, content, 0o644); err != nil {
		t.Fatal(err)
	}

	mock := newMockRedis()
	svc := newService(t, tmp, mock)
	svc.SyncFile(f)

	abs, _ := filepath.Abs(f)
	key := "file_cache:" + abs
	fields, ok := mock.store[key]
	if !ok {
		t.Fatal("expected Redis key to be set")
	}

	if fields["filename"] != "hello.txt" {
		t.Errorf("filename: got %q, want %q", fields["filename"], "hello.txt")
	}
	if fields["size_bytes"] != strconv.Itoa(len(content)) {
		t.Errorf("size_bytes: got %q, want %d", fields["size_bytes"], len(content))
	}
	if fields["content"] != base64.StdEncoding.EncodeToString(content) {
		t.Error("content mismatch")
	}
}

func TestSyncFile_SkipsNonexistentFile(t *testing.T) {
	tmp := t.TempDir()
	mock := newMockRedis()
	svc := newService(t, tmp, mock)
	svc.SyncFile(filepath.Join(tmp, "ghost.txt")) // must not panic or write
	if len(mock.store) != 0 {
		t.Error("expected no Redis writes for a nonexistent file")
	}
}

func TestSyncFile_BinaryContentPreserved(t *testing.T) {
	tmp := t.TempDir()
	binary := make([]byte, 256)
	for i := range binary {
		binary[i] = byte(i)
	}
	f := filepath.Join(tmp, "binary.bin")
	_ = os.WriteFile(f, binary, 0o644)

	mock := newMockRedis()
	svc := newService(t, tmp, mock)
	svc.SyncFile(f)

	abs, _ := filepath.Abs(f)
	key := "file_cache:" + abs
	encoded := mock.store[key]["content"]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(binary) {
		t.Error("binary round-trip failed")
	}
}

// ---------------------------------------------------------------------------
// RemoveFile
// ---------------------------------------------------------------------------

func TestRemoveFile_DeletesCorrectKey(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "gone.txt")

	mock := newMockRedis()
	abs, _ := filepath.Abs(filePath)
	mock.store["file_cache:"+abs] = map[string]string{"filename": "gone.txt"}

	svc := newService(t, tmp, mock)
	svc.RemoveFile(filePath)

	if _, ok := mock.store["file_cache:"+abs]; ok {
		t.Error("expected Redis key to be deleted")
	}
}

func TestRemoveFile_FiresEvenWhenFileMissingLocally(t *testing.T) {
	tmp := t.TempDir()
	mock := newMockRedis()
	svc := newService(t, tmp, mock)

	// Inject a key manually as if it existed previously.
	ghost := filepath.Join(tmp, "never_existed.txt")
	abs, _ := filepath.Abs(ghost)
	mock.store["file_cache:"+abs] = map[string]string{}

	svc.RemoveFile(ghost)
	if _, ok := mock.store["file_cache:"+abs]; ok {
		t.Error("expected key to be deleted even for locally-absent file")
	}
}

// ---------------------------------------------------------------------------
// LoadAndRestore
// ---------------------------------------------------------------------------

func TestLoadAndRestore_RestoresMissingFile(t *testing.T) {
	tmp := t.TempDir()
	watch := filepath.Join(tmp, "watch")
	physicalPath := filepath.Join(watch, "restored.txt")
	content := []byte("restored content")

	mock := newMockRedis()
	mock.keys_ = []string{"file_cache:" + physicalPath}
	mock.store["file_cache:"+physicalPath] = map[string]string{
		"filename":   "restored.txt",
		"size_bytes": strconv.Itoa(len(content)),
		"mtime":      "1700000000",
		"content":    base64.StdEncoding.EncodeToString(content),
	}

	svc := newService(t, watch, mock)
	svc.LoadAndRestore()

	got, err := os.ReadFile(physicalPath)
	if err != nil {
		t.Fatalf("restored file not found: %v", err)
	}
	if string(got) != string(content) {
		t.Error("content mismatch after restore")
	}
}

func TestLoadAndRestore_SkipsFileWithMatchingSize(t *testing.T) {
	tmp := t.TempDir()
	watch := filepath.Join(tmp, "watch")
	_ = os.MkdirAll(watch, 0o755)
	existing := filepath.Join(watch, "existing.txt")
	content := []byte("existing content")
	_ = os.WriteFile(existing, content, 0o644)
	originalMtime := mustStat(t, existing).ModTime()

	mock := newMockRedis()
	mock.keys_ = []string{"file_cache:" + existing}
	mock.store["file_cache:"+existing] = map[string]string{
		"filename":     "existing.txt",
		"size_bytes":   strconv.Itoa(len(content)),
		"mtime":        strconv.FormatFloat(float64(originalMtime.Unix()), 'f', -1, 64),
		"content_stub": base64.StdEncoding.EncodeToString(content),
	}

	svc := newService(t, watch, mock)
	svc.LoadAndRestore()

	// mtime must be unchanged (file was skipped).
	if mustStat(t, existing).ModTime() != originalMtime {
		t.Error("mtime changed for an up-to-date file")
	}
}

func TestLoadAndRestore_OverwritesFileWithDifferentSize(t *testing.T) {
	tmp := t.TempDir()
	watch := filepath.Join(tmp, "watch")
	_ = os.MkdirAll(watch, 0o755)
	target := filepath.Join(watch, "changed.txt")
	_ = os.WriteFile(target, []byte("old"), 0o644)
	newContent := []byte("brand new content")

	mock := newMockRedis()
	mock.keys_ = []string{"file_cache:" + target}
	mock.store["file_cache:"+target] = map[string]string{
		"filename":   "changed.txt",
		"size_bytes": strconv.Itoa(len(newContent)),
		"mtime":      "1700000000",
		"content":    base64.StdEncoding.EncodeToString(newContent),
	}

	svc := newService(t, watch, mock)
	svc.LoadAndRestore()

	got, _ := os.ReadFile(target)
	if string(got) != string(newContent) {
		t.Errorf("got %q, want %q", got, newContent)
	}
}

func TestLoadAndRestore_NoKeysDoesNothing(t *testing.T) {
	tmp := t.TempDir()
	mock := newMockRedis()
	mock.keys_ = []string{}
	svc := newService(t, tmp, mock)
	svc.LoadAndRestore() // must not panic
}

func TestLoadAndRestore_SetsMtimeAfterRestore(t *testing.T) {
	tmp := t.TempDir()
	watch := filepath.Join(tmp, "watch")
	target := filepath.Join(watch, "ts.txt")
	content := []byte("timestamped")
	const expectedMtime int64 = 1_700_000_000

	mock := newMockRedis()
	mock.keys_ = []string{"file_cache:" + target}
	mock.store["file_cache:"+target] = map[string]string{
		"filename":   "ts.txt",
		"size_bytes": strconv.Itoa(len(content)),
		"mtime":      strconv.FormatInt(expectedMtime, 10),
		"content":    base64.StdEncoding.EncodeToString(content),
	}

	svc := newService(t, watch, mock)
	svc.LoadAndRestore()

	gotMtime := mustStat(t, target).ModTime().Unix()
	diff := gotMtime - expectedMtime
	if diff < -1 || diff > 1 {
		t.Errorf("mtime %d differs from expected %d by more than 1s", gotMtime, expectedMtime)
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	return info
}
