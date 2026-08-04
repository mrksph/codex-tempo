package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mrksph/codex-tempo/internal/domain"
	"github.com/mrksph/codex-tempo/internal/projector"
)

type Store struct{ Pool *pgxpool.Pool }
type IngestResult struct {
	Accepted   int         `json:"accepted"`
	Duplicates int         `json:"duplicates"`
	Rejected   []Rejection `json:"rejected"`
}
type Rejection struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{Pool: pool}, nil
}
func (s *Store) Close() { s.Pool.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, schema)
	return err
}

func (s *Store) RegisterMachine(ctx context.Context, requestedID, name string) (string, string, error) {
	id := requestedID
	if id == "" {
		id = uuid.NewString()
	}
	if _, err := uuid.Parse(id); err != nil {
		return "", "", errors.New("invalid machine id")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(secret)
	hash := tokenHash(token)
	_, err := s.Pool.Exec(ctx, `INSERT INTO machines(id,name,token_hash) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET name=excluded.name,token_hash=excluded.token_hash,last_seen_at=now()`, id, name, hash)
	return id, token, err
}

func (s *Store) AuthenticateMachine(ctx context.Context, machineID, token string) error {
	var stored []byte
	if err := s.Pool.QueryRow(ctx, "SELECT token_hash FROM machines WHERE id=$1", machineID).Scan(&stored); err != nil {
		return errors.New("invalid machine credentials")
	}
	if !equal(stored, tokenHash(token)) {
		return errors.New("invalid machine credentials")
	}
	return nil
}

func tokenHash(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func (s *Store) Ingest(ctx context.Context, machineID string, events []domain.Event) (IngestResult, error) {
	result := IngestResult{Rejected: make([]Rejection, 0)}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	touched := map[string]bool{}
	for _, event := range events {
		if event.SessionID == "" || event.ID == "" || event.OccurredAt.IsZero() {
			result.Rejected = append(result.Rejected, Rejection{event.ID, "invalid_event"})
			continue
		}
		if _, err := uuid.Parse(event.ID); err != nil {
			result.Rejected = append(result.Rejected, Rejection{event.ID, "invalid_event_id"})
			continue
		}
		if event.RunID != "" {
			if _, err := uuid.Parse(event.RunID); err != nil {
				result.Rejected = append(result.Rejected, Rejection{event.ID, "invalid_run_id"})
				continue
			}
		}
		event.MachineID = machineID
		tag, err := tx.Exec(ctx, `INSERT INTO events(id,machine_id,session_id,run_id,sequence,occurred_at,kind,source,payload) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9) ON CONFLICT(id) DO UPDATE SET occurred_at=excluded.occurred_at,payload=excluded.payload WHERE events.source='wakapi' AND (events.occurred_at IS DISTINCT FROM excluded.occurred_at OR events.payload IS DISTINCT FROM excluded.payload)`, event.ID, event.MachineID, event.SessionID, event.RunID, event.Sequence, event.OccurredAt, event.Kind, event.Source, event.Payload)
		if err != nil {
			return result, err
		}
		if tag.RowsAffected() == 0 {
			result.Duplicates++
		} else {
			result.Accepted++
			touched[event.SessionID] = true
		}
	}
	if _, err = tx.Exec(ctx, "UPDATE machines SET last_seen_at=now() WHERE id=$1", machineID); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	for sessionID := range touched {
		if err = s.RebuildSession(ctx, machineID, sessionID); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *Store) RebuildSession(ctx context.Context, machineID, sessionID string) error {
	events, err := s.sessionEvents(ctx, sessionID)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	runs := projector.Project(events)
	metadata := projectFromEvents(events)
	projectID, projectName := metadata.ProjectID, metadata.ProjectName
	fingerprint, remoteHash, codexVersion := metadata.ProjectFingerprint, metadata.RemoteHash, metadata.CodexVersion
	if projectID == "" && len(runs) > 0 {
		projectID = runs[0].ProjectID
	}
	if projectID == "" {
		return nil
	}
	if _, err = uuid.Parse(projectID); err != nil {
		return fmt.Errorf("invalid projected project id: %w", err)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if projectName == "" {
		projectName = "project-" + projectID[:8]
	}
	if fingerprint == "" {
		fingerprint = projectID
	}
	if projectName != "" {
		var existingID, existingFingerprint string
		lookupErr := tx.QueryRow(ctx, `SELECT id::text,fingerprint FROM projects WHERE name=$1 ORDER BY created_at,id LIMIT 1`, projectName).Scan(&existingID, &existingFingerprint)
		if lookupErr == nil {
			projectID, fingerprint = existingID, existingFingerprint
		} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
			return lookupErr
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO projects(id,fingerprint,name,remote_hash) VALUES($1,$2,$3,NULLIF($4,'')) ON CONFLICT(id) DO UPDATE SET name=CASE WHEN EXCLUDED.name NOT LIKE 'project-%' THEN EXCLUDED.name ELSE projects.name END,remote_hash=COALESCE(EXCLUDED.remote_hash,projects.remote_hash),updated_at=now()`, projectID, fingerprint, projectName, remoteHash); err != nil {
		return err
	}
	sessionSource := "codex"
	for _, event := range events {
		if event.Source == "wakapi" || event.Source == "hook" || event.Source == "metrics" {
			sessionSource = event.Source
			break
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO sessions(id,machine_id,project_id,cwd,worktree_name,worktree_path,is_worktree,source,codex_version,started_at) VALUES($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),$7,$8,NULLIF($9,''),$10) ON CONFLICT(id) DO UPDATE SET project_id=excluded.project_id,cwd=COALESCE(excluded.cwd,sessions.cwd),worktree_name=COALESCE(excluded.worktree_name,sessions.worktree_name),worktree_path=COALESCE(excluded.worktree_path,sessions.worktree_path),is_worktree=excluded.is_worktree,source=excluded.source,codex_version=COALESCE(excluded.codex_version,sessions.codex_version)`, sessionID, machineID, projectID, metadata.WorktreePath, metadata.WorktreeName, metadata.WorktreePath, metadata.IsWorktree, sessionSource, codexVersion, events[0].OccurredAt); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM runs WHERE session_id=$1", sessionID); err != nil {
		return err
	}
	for _, run := range runs {
		run.ProjectID = projectID
		_, err = tx.Exec(ctx, `INSERT INTO runs(id,session_id,project_id,started_at,ended_at,last_activity_at,status,model,reasoning_effort,input_tokens,cached_input_tokens,output_tokens,reasoning_tokens,close_reason,projection_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, run.ID, run.SessionID, run.ProjectID, run.StartedAt, run.EndedAt, run.LastActivityAt, run.Status, run.Model, run.ReasoningEffort, run.InputTokens, run.CachedInputTokens, run.OutputTokens, run.ReasoningTokens, run.CloseReason, run.ProjectionVersion)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type projectMetadata struct {
	ProjectID          string `json:"project_id"`
	ProjectName        string `json:"project_name"`
	ProjectFingerprint string `json:"project_fingerprint"`
	RemoteHash         string `json:"remote_hash"`
	WorktreeName       string `json:"worktree_name"`
	WorktreePath       string `json:"worktree_path"`
	IsWorktree         bool   `json:"is_worktree"`
	CodexVersion       string `json:"codex_version"`
}

func projectFromEvents(events []domain.Event) projectMetadata {
	for _, event := range events {
		var payload projectMetadata
		_ = json.Unmarshal(event.Payload, &payload)
		if payload.ProjectID != "" {
			return payload
		}
	}
	return projectMetadata{}
}

func (s *Store) sessionEvents(ctx context.Context, sessionID string) ([]domain.Event, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text,machine_id::text,session_id,COALESCE(run_id::text,''),sequence,occurred_at,kind,source,payload FROM events WHERE session_id=$1 ORDER BY sequence,occurred_at,id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Event, 0)
	for rows.Next() {
		var e domain.Event
		if err = rows.Scan(&e.ID, &e.MachineID, &e.SessionID, &e.RunID, &e.Sequence, &e.OccurredAt, &e.Kind, &e.Source, &e.Payload); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *Store) Runs(ctx context.Context, from, to time.Time, activeOnly bool, projectID string) ([]domain.Run, error) {
	if _, err := s.Pool.Exec(ctx, `UPDATE runs SET ended_at=LEAST(last_activity_at + interval '90 seconds', started_at + interval '12 hours'),last_activity_at=LEAST(last_activity_at + interval '90 seconds', started_at + interval '12 hours'),status='abandoned',close_reason='lease_expired',updated_at=now() WHERE ended_at IS NULL AND now() >= LEAST(last_activity_at + interval '90 seconds', started_at + interval '12 hours')`); err != nil {
		return nil, err
	}
	query := `SELECT id::text,session_id,project_id::text,started_at,ended_at,last_activity_at,status,COALESCE(model,''),COALESCE(reasoning_effort,''),input_tokens,cached_input_tokens,output_tokens,reasoning_tokens,COALESCE(close_reason,''),projection_version FROM runs WHERE started_at<$2 AND COALESCE(ended_at,now())>$1`
	if activeOnly {
		query += ` AND ended_at IS NULL`
	}
	args := []any{from, to}
	if projectID != "" {
		query += ` AND project_id=$3`
		args = append(args, projectID)
	}
	query += ` ORDER BY started_at DESC`
	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Run, 0)
	for rows.Next() {
		var r domain.Run
		if err = rows.Scan(&r.ID, &r.SessionID, &r.ProjectID, &r.StartedAt, &r.EndedAt, &r.LastActivityAt, &r.Status, &r.Model, &r.ReasoningEffort, &r.InputTokens, &r.CachedInputTokens, &r.OutputTokens, &r.ReasoningTokens, &r.CloseReason, &r.ProjectionVersion); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

type ProjectSummary struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	RunCount     int64      `json:"run_count"`
	AgentSeconds float64    `json:"agent_seconds"`
	LastActiveAt *time.Time `json:"last_active_at"`
}

func (s *Store) Projects(ctx context.Context) ([]ProjectSummary, error) {
	rows, err := s.Pool.Query(ctx, `SELECT p.id::text,p.name,count(r.id),COALESCE(sum(extract(epoch from (COALESCE(r.ended_at,r.last_activity_at)-r.started_at))),0),max(r.last_activity_at) FROM projects p LEFT JOIN runs r ON r.project_id=p.id GROUP BY p.id,p.name ORDER BY max(r.last_activity_at) DESC NULLS LAST`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ProjectSummary, 0)
	for rows.Next() {
		var p ProjectSummary
		if err = rows.Scan(&p.ID, &p.Name, &p.RunCount, &p.AgentSeconds, &p.LastActiveAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *Store) Sessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := s.Pool.Query(ctx, `SELECT s.id,s.machine_id::text,s.project_id::text,COALESCE(s.cwd,''),COALESCE(s.worktree_name,''),COALESCE(s.worktree_path,''),s.is_worktree,s.source,COALESCE(s.codex_version,''),s.started_at,s.ended_at,COALESCE(max(COALESCE(r.ended_at,r.last_activity_at)),s.ended_at,s.started_at),count(r.id) FROM sessions s LEFT JOIN runs r ON r.session_id=s.id WHERE s.source<>'wakapi' GROUP BY s.id ORDER BY COALESCE(max(COALESCE(r.ended_at,r.last_activity_at)),s.ended_at,s.started_at) DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Session, 0)
	for rows.Next() {
		var v domain.Session
		if err = rows.Scan(&v.ID, &v.MachineID, &v.ProjectID, &v.CWD, &v.WorktreeName, &v.WorktreePath, &v.IsWorktree, &v.Source, &v.CodexVersion, &v.StartedAt, &v.EndedAt, &v.LastActivityAt, &v.RunCount); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (s *Store) Machines(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text,name,created_at,last_seen_at FROM machines ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		var id, name string
		var created time.Time
		var seen *time.Time
		if err = rows.Scan(&id, &name, &created, &seen); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{"id": id, "name": name, "created_at": created, "last_seen_at": seen})
	}
	return result, rows.Err()
}

func (s *Store) RebuildAll(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, "SELECT DISTINCT machine_id::text,session_id FROM events")
	if err != nil {
		return err
	}
	defer rows.Close()
	type pair struct{ machine, session string }
	var pairs []pair
	for rows.Next() {
		var p pair
		if err = rows.Scan(&p.machine, &p.session); err != nil {
			return err
		}
		pairs = append(pairs, p)
	}
	for _, p := range pairs {
		if err = s.RebuildSession(ctx, p.machine, p.session); err != nil {
			return err
		}
	}
	return nil
}

var _ = pgx.ErrNoRows
