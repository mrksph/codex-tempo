package domain

import (
	"encoding/json"
	"time"
)

const ProjectionVersion = 1

type EventKind string

const (
	EventSessionStarted  EventKind = "session_started"
	EventSessionClosed   EventKind = "session_closed"
	EventPromptSubmitted EventKind = "prompt_submitted"
	EventRunStarted      EventKind = "run_started"
	EventToolStarted     EventKind = "tool_started"
	EventToolFinished    EventKind = "tool_finished"
	EventAssistant       EventKind = "assistant_message"
	EventRunCompleted    EventKind = "run_completed"
	EventRunInterrupted  EventKind = "run_interrupted"
	EventRunFailed       EventKind = "run_failed"
	EventLease           EventKind = "lease"
	EventTokenUsage      EventKind = "token_usage"
	EventProjectChanged  EventKind = "project_changed"
)

type RunStatus string

const (
	RunRunning     RunStatus = "running"
	RunCompleted   RunStatus = "completed"
	RunInterrupted RunStatus = "interrupted"
	RunFailed      RunStatus = "failed"
	RunSuperseded  RunStatus = "superseded"
	RunAbandoned   RunStatus = "abandoned"
)

type Event struct {
	ID         string          `json:"id"`
	MachineID  string          `json:"machine_id,omitempty"`
	SessionID  string          `json:"session_id"`
	RunID      string          `json:"run_id,omitempty"`
	Sequence   int64           `json:"sequence"`
	OccurredAt time.Time       `json:"occurred_at"`
	Kind       EventKind       `json:"kind"`
	Source     string          `json:"source"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type Project struct {
	ID          string    `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	Name        string    `json:"name"`
	RootPath    string    `json:"root_path,omitempty"`
	RemoteHash  string    `json:"remote_hash,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Session struct {
	ID             string     `json:"id"`
	MachineID      string     `json:"machine_id"`
	ProjectID      string     `json:"project_id"`
	CWD            string     `json:"cwd,omitempty"`
	WorktreeName   string     `json:"worktree_name,omitempty"`
	WorktreePath   string     `json:"worktree_path,omitempty"`
	IsWorktree     bool       `json:"is_worktree"`
	Source         string     `json:"source"`
	CodexVersion   string     `json:"codex_version,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	RunCount       int64      `json:"run_count"`
}

type Run struct {
	ID                string     `json:"id"`
	SessionID         string     `json:"session_id"`
	ProjectID         string     `json:"project_id"`
	StartedAt         time.Time  `json:"started_at"`
	EndedAt           *time.Time `json:"ended_at,omitempty"`
	LastActivityAt    time.Time  `json:"last_activity_at"`
	Status            RunStatus  `json:"status"`
	Model             string     `json:"model,omitempty"`
	ReasoningEffort   string     `json:"reasoning_effort,omitempty"`
	InputTokens       int64      `json:"input_tokens"`
	CachedInputTokens int64      `json:"cached_input_tokens"`
	OutputTokens      int64      `json:"output_tokens"`
	ReasoningTokens   int64      `json:"reasoning_tokens"`
	CloseReason       string     `json:"close_reason,omitempty"`
	ProjectionVersion int        `json:"projection_version"`
}

type TokenUsage struct {
	InputTokens       int64 `json:"input_tokens"`
	CachedInputTokens int64 `json:"cached_input_tokens"`
	OutputTokens      int64 `json:"output_tokens"`
	ReasoningTokens   int64 `json:"reasoning_tokens"`
}
