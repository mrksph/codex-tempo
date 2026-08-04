package agent

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/mrksph/codex-tempo/internal/codex"
	"github.com/mrksph/codex-tempo/internal/localdb"
)

type Service struct {
	Store        *localdb.Store
	Parser       codex.Parser
	SessionsDir  string
	ScanInterval time.Duration
	Logger       *slog.Logger
}

func (s *Service) ScanOnce(ctx context.Context) error {
	return filepath.WalkDir(s.SessionsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			s.logger().Warn("cannot inspect transcript path", "path", path, "error", walkErr)
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		return s.ScanFile(ctx, path)
	})
}

func (s *Service) ScanFile(ctx context.Context, path string) error {
	cursor, err := s.Store.Cursor(ctx, path)
	if err != nil {
		return err
	}
	cursor, events, parseErrors, err := s.Parser.ParseFile(ctx, path, cursor)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	for _, parseErr := range parseErrors {
		_ = s.Store.RecordParserError(ctx, path, cursor.ByteOffset, parseErr)
		s.logger().Warn("invalid transcript record", "path", path, "error", parseErr)
	}
	if err := s.Store.CommitParsed(ctx, cursor, events); err != nil {
		return err
	}
	if len(events) > 0 {
		s.logger().Info("transcript parsed", "path", path, "events", len(events))
	}
	return nil
}

func (s *Service) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

func (s *Service) Run(ctx context.Context) error {
	if s.ScanInterval <= 0 {
		s.ScanInterval = 15 * time.Second
	}
	if s.Logger == nil {
		s.Logger = slog.Default()
	}
	if err := s.ScanOnce(ctx); err != nil {
		s.Logger.Error("initial scan failed", "error", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	addDirectories := func() {
		_ = filepath.WalkDir(s.SessionsDir, func(path string, entry fs.DirEntry, err error) error {
			if err == nil && entry.IsDir() {
				_ = watcher.Add(path)
			}
			return nil
		})
	}
	addDirectories()
	ticker := time.NewTicker(s.ScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create != 0 {
				addDirectories()
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				if err := s.ScanOnce(ctx); err != nil {
					s.Logger.Error("scan failed", "error", err)
				}
			}
		case err := <-watcher.Errors:
			s.Logger.Warn("watcher error", "error", err)
		case <-ticker.C:
			if err := s.ScanOnce(ctx); err != nil {
				s.Logger.Error("backup scan failed", "error", err)
			}
		}
	}
}
