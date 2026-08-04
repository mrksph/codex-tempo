package hooklog

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const filePrefix = "hook-"

type Logger struct {
	*slog.Logger
	file *os.File
}

func Open(dataDir string, retention time.Duration) (*Logger, error) {
	return open(dataDir, retention, time.Now())
}

func open(dataDir string, retention time.Duration, now time.Time) (*Logger, error) {
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, fmt.Errorf("create hook log directory: %w", err)
	}
	if retention > 0 {
		removeExpired(logDir, retention, now)
	}
	path := filepath.Join(logDir, filePrefix+now.Format(time.DateOnly)+".jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open hook log: %w", err)
	}
	if err = file.Chmod(0o600); err != nil {
		file.Close()
		return nil, fmt.Errorf("secure hook log: %w", err)
	}
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})),
		file:   file,
	}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func removeExpired(logDir string, retention time.Duration, now time.Time) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := now.Add(-retention)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), filePrefix) || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(logDir, entry.Name()))
		}
	}
}
