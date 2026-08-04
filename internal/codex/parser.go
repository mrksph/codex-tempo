package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/mrksph/codex-tempo/internal/domain"
	"github.com/mrksph/codex-tempo/internal/localdb"
	"github.com/mrksph/codex-tempo/internal/projectresolver"
)

type rawRecord struct {
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type envelope struct {
	Type        string       `json:"type"`
	ID          string       `json:"id"`
	SessionID   string       `json:"session_id"`
	TurnID      string       `json:"turn_id"`
	CWD         string       `json:"cwd"`
	Model       string       `json:"model"`
	Effort      string       `json:"effort"`
	CLIVersion  string       `json:"cli_version"`
	StartedAt   flexibleTime `json:"started_at"`
	CompletedAt flexibleTime `json:"completed_at"`
	Info        struct {
		Last  TokenUsage `json:"last_token_usage"`
		Total TokenUsage `json:"total_token_usage"`
	} `json:"info"`
}

type flexibleTime struct {
	time.Time
	Valid bool
}

func (t *flexibleTime) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return err
		}
		t.Time, t.Valid = parsed, true
		return nil
	}
	var unix float64
	if err := json.Unmarshal(data, &unix); err == nil {
		seconds, fraction := math.Modf(unix)
		t.Time = time.Unix(int64(seconds), int64(fraction*float64(time.Second))).UTC()
		t.Valid = true
	}
	return nil
}

type TokenUsage struct {
	InputTokens       int64 `json:"input_tokens"`
	CachedInputTokens int64 `json:"cached_input_tokens"`
	OutputTokens      int64 `json:"output_tokens"`
	ReasoningTokens   int64 `json:"reasoning_output_tokens"`
}

type Parser struct {
	MachineID  string
	StorePaths bool
}

func (p Parser) ParseFile(ctx context.Context, path string, cursor localdb.Cursor) (localdb.Cursor, []domain.Event, []error, error) {
	file, err := os.Open(path)
	if err != nil {
		return cursor, nil, nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return cursor, nil, nil, err
	}
	identity := fileIdentity(info)
	if cursor.FileIdentity != "" && cursor.FileIdentity != identity {
		cursor = localdb.Cursor{Path: path}
	}
	cursor.FileIdentity = identity
	if cursor.ByteOffset > info.Size() {
		cursor.ByteOffset = 0
		cursor.LastSequence = 0
	}
	if _, err = file.Seek(cursor.ByteOffset, io.SeekStart); err != nil {
		return cursor, nil, nil, err
	}
	reader := bufio.NewReader(file)
	var events []domain.Event
	var parseErrors []error
	for {
		lineStart := cursor.ByteOffset
		line, readErr := reader.ReadBytes('\n')
		if len(line) == 0 && readErr != nil {
			break
		}
		if readErr == io.EOF && len(line) > 0 && line[len(line)-1] != '\n' {
			break
		}
		cursor.ByteOffset += int64(len(line))
		cursor.LastModified = info.ModTime()
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if readErr != nil {
				break
			}
			continue
		}
		event, metadata, err := p.parseLine(ctx, line, cursor)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("offset %d: %w", lineStart, err))
			if readErr != nil {
				break
			}
			continue
		}
		if metadata != nil {
			if metadata.SessionID != "" {
				cursor.SessionID = metadata.SessionID
			}
			if metadata.CWD != "" {
				if p.StorePaths {
					cursor.CWD = metadata.CWD
				} else {
					cursor.CWD = ""
				}
				project := projectresolver.Resolve(ctx, metadata.CWD, p.MachineID, p.StorePaths)
				cursor.ProjectID = project.ID
				cursor.ProjectName = project.Name
				cursor.ProjectFingerprint = project.Fingerprint
				cursor.RemoteHash = project.RemoteHash
				cursor.WorktreeName = project.WorktreeName
				cursor.WorktreePath = project.WorktreePath
				cursor.IsWorktree = project.IsWorktree
			}
		}
		if event != nil && event.Kind == domain.EventSessionStarted {
			body, _ := json.Marshal(map[string]any{
				"project_id": cursor.ProjectID, "project_name": cursor.ProjectName,
				"project_fingerprint": cursor.ProjectFingerprint, "remote_hash": cursor.RemoteHash,
				"worktree_name": cursor.WorktreeName, "worktree_path": cursor.WorktreePath,
				"is_worktree": cursor.IsWorktree, "codex_version": metadata.CLIVersion,
			})
			event.Payload = body
		}
		if event != nil {
			cursor.LastSequence++
			event.Sequence = cursor.LastSequence
			event.MachineID = p.MachineID
			event.SessionID = cursor.SessionID
			event.ID = domain.DeterministicUUID(p.MachineID, event.SessionID, string(event.Kind), event.RunID, strconv.FormatInt(event.Sequence, 10))
			cursor.LastEventID = event.ID
			events = append(events, *event)
		}
		if readErr != nil {
			break
		}
	}
	return cursor, events, parseErrors, nil
}

func (p Parser) parseLine(ctx context.Context, line []byte, cursor localdb.Cursor) (*domain.Event, *envelope, error) {
	var record rawRecord
	if err := json.Unmarshal(line, &record); err != nil {
		return nil, nil, err
	}
	var payload envelope
	if err := json.Unmarshal(record.Payload, &payload); err != nil {
		return nil, nil, err
	}
	at := record.Timestamp
	switch record.Type {
	case "session_meta":
		if payload.SessionID == "" {
			payload.SessionID = payload.ID
		}
		if payload.SessionID == "" {
			return nil, nil, errors.New("session_meta without session id")
		}
		body, _ := json.Marshal(map[string]any{"project_id": cursor.ProjectID, "codex_version": payload.CLIVersion})
		return &domain.Event{OccurredAt: at, Kind: domain.EventSessionStarted, Source: "transcript", Payload: body}, &payload, nil
	case "turn_context":
		if payload.TurnID == "" {
			return nil, &payload, nil
		}
		body, _ := json.Marshal(map[string]any{"project_id": cursor.ProjectID, "model": payload.Model, "reasoning_effort": payload.Effort})
		return &domain.Event{RunID: runID(cursor.SessionID, payload.TurnID), OccurredAt: at, Kind: domain.EventLease, Source: "transcript", Payload: body}, &payload, nil
	case "event_msg":
		switch payload.Type {
		case "task_started":
			if payload.StartedAt.Valid {
				at = payload.StartedAt.Time
			}
			body, _ := json.Marshal(map[string]any{"project_id": cursor.ProjectID})
			return &domain.Event{RunID: runID(cursor.SessionID, payload.TurnID), OccurredAt: at, Kind: domain.EventRunStarted, Source: "transcript", Payload: body}, nil, nil
		case "task_complete":
			if payload.CompletedAt.Valid {
				at = payload.CompletedAt.Time
			}
			return &domain.Event{RunID: runID(cursor.SessionID, payload.TurnID), OccurredAt: at, Kind: domain.EventRunCompleted, Source: "transcript", Payload: json.RawMessage(`{}`)}, nil, nil
		case "token_count":
			body, _ := json.Marshal(map[string]int64{"input_tokens": payload.Info.Last.InputTokens, "cached_input_tokens": payload.Info.Last.CachedInputTokens, "output_tokens": payload.Info.Last.OutputTokens, "reasoning_tokens": payload.Info.Last.ReasoningTokens})
			return &domain.Event{OccurredAt: at, Kind: domain.EventTokenUsage, Source: "transcript", Payload: body}, nil, nil
		}
	}
	return nil, nil, nil
}

func runID(sessionID, native string) string {
	if native == "" {
		return ""
	}
	return domain.DeterministicUUID("run", sessionID, native)
}

func fileIdentity(info os.FileInfo) string {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
	}
	return fmt.Sprintf("%s:%d", info.Name(), info.ModTime().UnixNano())
}
