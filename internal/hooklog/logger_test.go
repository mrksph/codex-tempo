package hooklog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenRotatesDailyAndRemovesLogsOlderThanRetention(t *testing.T) {
	dataDir := t.TempDir()
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	oldPath := filepath.Join(logDir, "hook-2026-07-20.jsonl")
	keptPath := filepath.Join(logDir, "hook-2026-07-25.jsonl")
	for _, path := range []string{oldPath, keptPath} {
		if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(oldPath, now.Add(-15*24*time.Hour), now.Add(-15*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(keptPath, now.Add(-10*24*time.Hour), now.Add(-10*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	logger, err := open(dataDir, 14*24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hook started", "session_id", "session")
	if err = logger.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expired log still exists: %v", err)
	}
	if _, err = os.Stat(keptPath); err != nil {
		t.Fatalf("recent log was removed: %v", err)
	}

	currentPath := filepath.Join(logDir, "hook-2026-08-04.jsonl")
	file, err := os.Open(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var record map[string]any
	if err = json.Unmarshal(readLastLine(t, file), &record); err != nil {
		t.Fatal(err)
	}
	if record["msg"] != "hook started" || record["session_id"] != "session" {
		t.Fatalf("record = %#v", record)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func readLastLine(t *testing.T, file *os.File) []byte {
	t.Helper()
	var last []byte
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		last = append(last[:0], scanner.Bytes()...)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return last
}
