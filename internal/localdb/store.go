package localdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/mrksph/codex-tempo/internal/domain"
)

type Store struct{ db *sql.DB }

const schemaVersion = 1

type Cursor struct {
	Path, FileIdentity              string
	ByteOffset                      int64
	LastEventID, SessionID          string
	ProjectID, ProjectName          string
	ProjectFingerprint, RemoteHash  string
	CWD, WorktreeName, WorktreePath string
	IsWorktree                      bool
	LastSequence                    int64
	LastModified                    time.Time
}

type HookHeartbeat struct {
	SessionID, ProjectID, ProjectName string
	ProjectFingerprint, RemoteHash    string
	WorktreeName, WorktreePath, Model string
	IsWorktree                        bool
	OccurredAt                        time.Time
}

type HookInterval struct {
	StartedAt, EndedAt                time.Time
	SessionID, ProjectID, ProjectName string
	ProjectFingerprint, RemoteHash    string
	WorktreeName, WorktreePath, Model string
	IsWorktree                        bool
}

type MetricCursor struct {
	Path, FileIdentity string
	ByteOffset         int64
	State              json.RawMessage
	LastModified       time.Time
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	query := dsn.Query()
	query.Set("_txlock", "immediate")
	query.Add("_pragma", "busy_timeout(1000)")
	query.Add("_pragma", "foreign_keys(ON)")
	query.Add("_pragma", "synchronous(NORMAL)")
	dsn.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	var currentVersion int
	if err = db.QueryRow("PRAGMA user_version").Scan(&currentVersion); err != nil {
		db.Close()
		return nil, err
	}
	if currentVersion < schemaVersion {
		if _, err = db.Exec("PRAGMA journal_mode=WAL"); err == nil {
			err = s.migrate()
		}
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS transcript_cursors (
 path TEXT PRIMARY KEY, file_identity TEXT NOT NULL, byte_offset INTEGER NOT NULL DEFAULT 0,
 last_event_id TEXT NOT NULL DEFAULT '', last_sequence INTEGER NOT NULL DEFAULT 0,
 session_id TEXT NOT NULL DEFAULT '', project_id TEXT NOT NULL DEFAULT '', project_name TEXT NOT NULL DEFAULT '',
	project_fingerprint TEXT NOT NULL DEFAULT '', remote_hash TEXT NOT NULL DEFAULT '', cwd TEXT NOT NULL DEFAULT '',
	worktree_name TEXT NOT NULL DEFAULT '', worktree_path TEXT NOT NULL DEFAULT '', is_worktree INTEGER NOT NULL DEFAULT 0,
 last_modified TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
 id TEXT PRIMARY KEY, machine_id TEXT NOT NULL, session_id TEXT NOT NULL, run_id TEXT NOT NULL DEFAULT '',
 sequence INTEGER NOT NULL, occurred_at TEXT NOT NULL, kind TEXT NOT NULL, source TEXT NOT NULL,
 payload BLOB NOT NULL DEFAULT '{}', sync_state TEXT NOT NULL DEFAULT 'pending',
 error TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(machine_id, session_id, sequence)
);
CREATE INDEX IF NOT EXISTS events_sync_idx ON events(sync_state, occurred_at);
CREATE TABLE IF NOT EXISTS parser_errors (
 id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, byte_offset INTEGER NOT NULL,
 message TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS hook_heartbeats (
 session_id TEXT PRIMARY KEY, occurred_at TEXT NOT NULL, project_id TEXT NOT NULL,
 project_name TEXT NOT NULL, project_fingerprint TEXT NOT NULL, remote_hash TEXT NOT NULL DEFAULT '',
	worktree_name TEXT NOT NULL DEFAULT '', worktree_path TEXT NOT NULL DEFAULT '', is_worktree INTEGER NOT NULL DEFAULT 0,
	model TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS metric_cursors (
 path TEXT PRIMARY KEY, file_identity TEXT NOT NULL, byte_offset INTEGER NOT NULL DEFAULT 0,
 state BLOB NOT NULL DEFAULT '{}', last_modified TEXT NOT NULL
);`)
	if err != nil {
		return err
	}
	for _, statement := range []string{
		"ALTER TABLE transcript_cursors ADD COLUMN project_name TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE transcript_cursors ADD COLUMN project_fingerprint TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE transcript_cursors ADD COLUMN remote_hash TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE transcript_cursors ADD COLUMN worktree_name TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE transcript_cursors ADD COLUMN worktree_path TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE transcript_cursors ADD COLUMN is_worktree INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE hook_heartbeats ADD COLUMN worktree_name TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE hook_heartbeats ADD COLUMN worktree_path TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE hook_heartbeats ADD COLUMN is_worktree INTEGER NOT NULL DEFAULT 0",
	} {
		if _, err = s.db.Exec(statement); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	_, err = s.db.Exec(fmt.Sprintf("PRAGMA user_version=%d", schemaVersion))
	return err
}

// AdvanceHookHeartbeat returns a work interval only when two consecutive hooks
// for the same Codex session are close enough to represent active work.
func (s *Store) AdvanceHookHeartbeat(ctx context.Context, heartbeat HookHeartbeat, timeout time.Duration) (*HookInterval, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var previous HookHeartbeat
	var occurredAt string
	err = tx.QueryRowContext(ctx, `SELECT session_id,occurred_at,project_id,project_name,project_fingerprint,remote_hash,worktree_name,worktree_path,is_worktree,model FROM hook_heartbeats WHERE session_id=?`, heartbeat.SessionID).
		Scan(&previous.SessionID, &occurredAt, &previous.ProjectID, &previous.ProjectName, &previous.ProjectFingerprint, &previous.RemoteHash, &previous.WorktreeName, &previous.WorktreePath, &previous.IsWorktree, &previous.Model)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	hadPrevious := err == nil
	if hadPrevious {
		previous.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
		if err != nil {
			return nil, err
		}
		if !heartbeat.OccurredAt.After(previous.OccurredAt) {
			return nil, tx.Commit()
		}
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO hook_heartbeats(session_id,occurred_at,project_id,project_name,project_fingerprint,remote_hash,worktree_name,worktree_path,is_worktree,model) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(session_id) DO UPDATE SET occurred_at=excluded.occurred_at,project_id=excluded.project_id,project_name=excluded.project_name,project_fingerprint=excluded.project_fingerprint,remote_hash=excluded.remote_hash,worktree_name=excluded.worktree_name,worktree_path=excluded.worktree_path,is_worktree=excluded.is_worktree,model=excluded.model`, heartbeat.SessionID, heartbeat.OccurredAt.UTC().Format(time.RFC3339Nano), heartbeat.ProjectID, heartbeat.ProjectName, heartbeat.ProjectFingerprint, heartbeat.RemoteHash, heartbeat.WorktreeName, heartbeat.WorktreePath, heartbeat.IsWorktree, heartbeat.Model)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if !hadPrevious {
		return nil, nil
	}
	gap := heartbeat.OccurredAt.Sub(previous.OccurredAt)
	if gap <= 0 || gap > timeout {
		return nil, nil
	}
	return &HookInterval{
		StartedAt: previous.OccurredAt, EndedAt: heartbeat.OccurredAt,
		SessionID: previous.SessionID, ProjectID: previous.ProjectID, ProjectName: previous.ProjectName,
		ProjectFingerprint: previous.ProjectFingerprint, RemoteHash: previous.RemoteHash,
		WorktreeName: previous.WorktreeName, WorktreePath: previous.WorktreePath, IsWorktree: previous.IsWorktree,
		Model: previous.Model,
	}, nil
}

func (s *Store) MachineID(ctx context.Context) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key='machine_id'").Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	id = uuid.NewString()
	_, err = s.db.ExecContext(ctx, "INSERT INTO metadata(key,value) VALUES('machine_id',?)", id)
	return id, err
}

func (s *Store) EnsureTimeMetadata(ctx context.Context, key string, fallback time.Time) (time.Time, error) {
	value := fallback.UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, "INSERT INTO metadata(key,value) VALUES(?,?) ON CONFLICT(key) DO NOTHING", key, value); err != nil {
		return time.Time{}, err
	}
	var stored string
	if err := s.db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key=?", key).Scan(&stored); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, stored)
}

func (s *Store) MetricCursor(ctx context.Context, path string) (MetricCursor, error) {
	var cursor MetricCursor
	var modified string
	err := s.db.QueryRowContext(ctx, "SELECT path,file_identity,byte_offset,state,last_modified FROM metric_cursors WHERE path=?", path).
		Scan(&cursor.Path, &cursor.FileIdentity, &cursor.ByteOffset, &cursor.State, &modified)
	if err == sql.ErrNoRows {
		cursor.Path = path
		return cursor, nil
	}
	if err != nil {
		return cursor, err
	}
	cursor.LastModified, _ = time.Parse(time.RFC3339Nano, modified)
	return cursor, nil
}

func (s *Store) CommitMetrics(ctx context.Context, cursor MetricCursor, events []domain.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, event := range events {
		payload := event.Payload
		if len(payload) == 0 {
			payload = json.RawMessage("{}")
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO events(id,machine_id,session_id,run_id,sequence,occurred_at,kind,source,payload) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING", event.ID, event.MachineID, event.SessionID, event.RunID, event.Sequence, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.Kind, event.Source, []byte(payload)); err != nil {
			return err
		}
	}
	state := cursor.State
	if len(state) == 0 {
		state = json.RawMessage("{}")
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO metric_cursors(path,file_identity,byte_offset,state,last_modified) VALUES(?,?,?,?,?) ON CONFLICT(path) DO UPDATE SET file_identity=excluded.file_identity,byte_offset=excluded.byte_offset,state=excluded.state,last_modified=excluded.last_modified", cursor.Path, cursor.FileIdentity, cursor.ByteOffset, []byte(state), cursor.LastModified.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) TimeMetadata(ctx context.Context, key string) (time.Time, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key=?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, true, err
}

func (s *Store) SetTimeMetadata(ctx context.Context, key string, value time.Time) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO metadata(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) Cursor(ctx context.Context, path string) (Cursor, error) {
	var c Cursor
	var modified string
	err := s.db.QueryRowContext(ctx, `SELECT path,file_identity,byte_offset,last_event_id,last_sequence,session_id,project_id,project_name,project_fingerprint,remote_hash,cwd,worktree_name,worktree_path,is_worktree,last_modified FROM transcript_cursors WHERE path=?`, path).
		Scan(&c.Path, &c.FileIdentity, &c.ByteOffset, &c.LastEventID, &c.LastSequence, &c.SessionID, &c.ProjectID, &c.ProjectName, &c.ProjectFingerprint, &c.RemoteHash, &c.CWD, &c.WorktreeName, &c.WorktreePath, &c.IsWorktree, &modified)
	if err == sql.ErrNoRows {
		c.Path = path
		return c, nil
	}
	if err != nil {
		return c, err
	}
	c.LastModified, _ = time.Parse(time.RFC3339Nano, modified)
	return c, nil
}

func (s *Store) CommitParsed(ctx context.Context, cursor Cursor, events []domain.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, event := range events {
		payload := event.Payload
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO events(id,machine_id,session_id,run_id,sequence,occurred_at,kind,source,payload) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, event.ID, event.MachineID, event.SessionID, event.RunID, event.Sequence, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.Kind, event.Source, []byte(payload))
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO transcript_cursors(path,file_identity,byte_offset,last_event_id,last_sequence,session_id,project_id,project_name,project_fingerprint,remote_hash,cwd,worktree_name,worktree_path,is_worktree,last_modified) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(path) DO UPDATE SET file_identity=excluded.file_identity,byte_offset=excluded.byte_offset,last_event_id=excluded.last_event_id,last_sequence=excluded.last_sequence,session_id=excluded.session_id,project_id=excluded.project_id,project_name=excluded.project_name,project_fingerprint=excluded.project_fingerprint,remote_hash=excluded.remote_hash,cwd=excluded.cwd,worktree_name=excluded.worktree_name,worktree_path=excluded.worktree_path,is_worktree=excluded.is_worktree,last_modified=excluded.last_modified`, cursor.Path, cursor.FileIdentity, cursor.ByteOffset, cursor.LastEventID, cursor.LastSequence, cursor.SessionID, cursor.ProjectID, cursor.ProjectName, cursor.ProjectFingerprint, cursor.RemoteHash, cursor.CWD, cursor.WorktreeName, cursor.WorktreePath, cursor.IsWorktree, cursor.LastModified.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Enqueue(ctx context.Context, events []domain.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, event := range events {
		payload := event.Payload
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO events(id,machine_id,session_id,run_id,sequence,occurred_at,kind,source,payload) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET occurred_at=excluded.occurred_at,payload=excluded.payload,sync_state='pending' WHERE events.source='wakapi' AND (events.occurred_at<>excluded.occurred_at OR events.payload<>excluded.payload)`, event.ID, event.MachineID, event.SessionID, event.RunID, event.Sequence, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.Kind, event.Source, []byte(payload)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RecordParserError(ctx context.Context, path string, offset int64, parseErr error) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO parser_errors(path,byte_offset,message) VALUES(?,?,?)", path, offset, parseErr.Error())
	return err
}

func (s *Store) Events(ctx context.Context, from, to time.Time) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,machine_id,session_id,run_id,sequence,occurred_at,kind,source,payload FROM events WHERE occurred_at>=? AND occurred_at<? ORDER BY occurred_at,sequence`, from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Event, 0)
	for rows.Next() {
		var event domain.Event
		var occurred string
		if err := rows.Scan(&event.ID, &event.MachineID, &event.SessionID, &event.RunID, &event.Sequence, &occurred, &event.Kind, &event.Source, &event.Payload); err != nil {
			return nil, err
		}
		event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred)
		if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func (s *Store) Pending(ctx context.Context, limit int) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,machine_id,session_id,run_id,sequence,occurred_at,kind,source,payload FROM events WHERE sync_state='pending' ORDER BY occurred_at LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Event, 0)
	for rows.Next() {
		var e domain.Event
		var at string
		if err := rows.Scan(&e.ID, &e.MachineID, &e.SessionID, &e.RunID, &e.Sequence, &at, &e.Kind, &e.Source, &e.Payload); err != nil {
			return nil, err
		}
		e.OccurredAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *Store) MarkAcknowledged(ctx context.Context, ids []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err = tx.ExecContext(ctx, "UPDATE events SET sync_state='acknowledged',error='' WHERE id=?", id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Diagnostics(ctx context.Context) (map[string]int64, error) {
	result := map[string]int64{}
	for _, query := range []struct{ name, sql string }{{"pending", "SELECT count(*) FROM events WHERE sync_state='pending'"}, {"acknowledged", "SELECT count(*) FROM events WHERE sync_state='acknowledged'"}, {"parser_errors", "SELECT count(*) FROM parser_errors"}, {"cursors", "SELECT count(*) FROM transcript_cursors"}} {
		var count int64
		if err := s.db.QueryRowContext(ctx, query.sql).Scan(&count); err != nil {
			return nil, fmt.Errorf("%s: %w", query.name, err)
		}
		result[query.name] = count
	}
	return result, nil
}

func (s *Store) ResetCursors(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DELETE FROM transcript_cursors"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM parser_errors"); err != nil {
		return err
	}
	return tx.Commit()
}
