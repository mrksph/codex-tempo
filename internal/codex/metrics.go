package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/mrksph/codex-tempo/internal/domain"
	"github.com/mrksph/codex-tempo/internal/localdb"
	"github.com/mrksph/codex-tempo/internal/projectresolver"
)

type metricState struct {
	SessionID, ProjectID, ProjectName, ProjectFingerprint, RemoteHash, WorktreeName, WorktreePath string
	ActiveTurn, Model, ReasoningEffort                                                            string
	IsWorktree                                                                                    bool
	Baseline, Current, Last                                                                       TokenUsage
}

type MetricScanner struct {
	MachineID  string
	StorePaths bool
}

// Scan reads only structural metadata and token counters. It never stores
// prompt, response, or tool content, and emits zero-duration metric runs.
func (s MetricScanner) Scan(ctx context.Context, path string, cursor localdb.MetricCursor, cutover time.Time) (localdb.MetricCursor, []domain.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return cursor, nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return cursor, nil, err
	}
	identity := fileIdentity(info)
	if cursor.FileIdentity != "" && cursor.FileIdentity != identity {
		cursor = localdb.MetricCursor{Path: path}
	}
	cursor.Path, cursor.FileIdentity, cursor.LastModified = path, identity, info.ModTime()
	if cursor.ByteOffset > info.Size() {
		cursor.ByteOffset = 0
	}
	var state metricState
	_ = json.Unmarshal(cursor.State, &state)
	if _, err = file.Seek(cursor.ByteOffset, io.SeekStart); err != nil {
		return cursor, nil, err
	}
	reader := bufio.NewReader(file)
	var events []domain.Event
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) == 0 && readErr != nil {
			break
		}
		if readErr == io.EOF && len(line) > 0 && line[len(line)-1] != '\n' {
			break
		}
		cursor.ByteOffset += int64(len(line))
		var record rawRecord
		if json.Unmarshal(bytes.TrimSpace(line), &record) == nil {
			var payload envelope
			if json.Unmarshal(record.Payload, &payload) == nil {
				switch record.Type {
				case "session_meta":
					if payload.SessionID == "" {
						payload.SessionID = payload.ID
					}
					state.SessionID = payload.SessionID
					if payload.CWD != "" {
						project := projectresolver.Resolve(ctx, payload.CWD, s.MachineID, s.StorePaths)
						state.ProjectID, state.ProjectName = project.ID, project.Name
						state.ProjectFingerprint, state.RemoteHash = project.Fingerprint, project.RemoteHash
						state.WorktreeName, state.WorktreePath = project.WorktreeName, project.WorktreePath
						state.IsWorktree = project.IsWorktree
					}
				case "turn_context":
					state.Model, state.ReasoningEffort = payload.Model, payload.Effort
				case "event_msg":
					switch payload.Type {
					case "task_started":
						state.ActiveTurn = payload.TurnID
						state.Baseline = state.Current
					case "token_count":
						state.Current = payload.Info.Total
						state.Last = payload.Info.Last
					case "task_complete":
						at := record.Timestamp
						if payload.CompletedAt.Valid {
							at = payload.CompletedAt.Time
						}
						if state.ActiveTurn == "" {
							state.ActiveTurn = payload.TurnID
						}
						if state.ActiveTurn != "" && state.ProjectID != "" && (cutover.IsZero() || !at.Before(cutover)) {
							usage := tokenDelta(state.Current, state.Baseline)
							if usage == (TokenUsage{}) {
								usage = state.Last
							}
							events = append(events, metricEvents(s.MachineID, state, at, usage)...)
						}
						state.ActiveTurn = ""
						state.Baseline = state.Current
					}
				}
			}
		}
		if readErr != nil {
			break
		}
	}
	cursor.State, _ = json.Marshal(state)
	return cursor, events, nil
}

func tokenDelta(current, baseline TokenUsage) TokenUsage {
	return TokenUsage{
		InputTokens:       max64(0, current.InputTokens-baseline.InputTokens),
		CachedInputTokens: max64(0, current.CachedInputTokens-baseline.CachedInputTokens),
		OutputTokens:      max64(0, current.OutputTokens-baseline.OutputTokens),
		ReasoningTokens:   max64(0, current.ReasoningTokens-baseline.ReasoningTokens),
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func metricEvents(machineID string, state metricState, at time.Time, usage TokenUsage) []domain.Event {
	runID := domain.DeterministicUUID("metric-run", state.SessionID, state.ActiveTurn)
	sessionID := "metrics:" + runID
	projectPayload, _ := json.Marshal(map[string]any{
		"project_id": state.ProjectID, "project_name": state.ProjectName,
		"project_fingerprint": state.ProjectFingerprint, "remote_hash": state.RemoteHash,
		"worktree_name": state.WorktreeName, "worktree_path": state.WorktreePath,
		"is_worktree": state.IsWorktree,
		"model":       state.Model, "reasoning_effort": state.ReasoningEffort,
	})
	tokenPayload, _ := json.Marshal(map[string]int64{
		"input_tokens": usage.InputTokens, "cached_input_tokens": usage.CachedInputTokens,
		"output_tokens": usage.OutputTokens, "reasoning_tokens": usage.ReasoningTokens,
	})
	return []domain.Event{
		{ID: domain.DeterministicUUID("metric-event", runID, "session"), MachineID: machineID, SessionID: sessionID, Sequence: 1, OccurredAt: at, Kind: domain.EventSessionStarted, Source: "metrics", Payload: projectPayload},
		{ID: domain.DeterministicUUID("metric-event", runID, "start"), MachineID: machineID, SessionID: sessionID, RunID: runID, Sequence: 2, OccurredAt: at, Kind: domain.EventRunStarted, Source: "metrics", Payload: projectPayload},
		{ID: domain.DeterministicUUID("metric-event", runID, "tokens"), MachineID: machineID, SessionID: sessionID, RunID: runID, Sequence: 3, OccurredAt: at, Kind: domain.EventTokenUsage, Source: "metrics", Payload: tokenPayload},
		{ID: domain.DeterministicUUID("metric-event", runID, "complete"), MachineID: machineID, SessionID: sessionID, RunID: runID, Sequence: 4, OccurredAt: at, Kind: domain.EventRunCompleted, Source: "metrics", Payload: json.RawMessage("{}")},
	}
}
