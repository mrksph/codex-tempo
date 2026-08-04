package projector

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/mrksph/codex-tempo/internal/domain"
)

const inferredClosureGrace = 90 * time.Second

type eventPayload struct {
	ProjectID         string `json:"project_id"`
	Model             string `json:"model"`
	ReasoningEffort   string `json:"reasoning_effort"`
	InputTokens       int64  `json:"input_tokens"`
	CachedInputTokens int64  `json:"cached_input_tokens"`
	OutputTokens      int64  `json:"output_tokens"`
	ReasoningTokens   int64  `json:"reasoning_tokens"`
}

func Project(events []domain.Event) []domain.Run {
	ordered := append([]domain.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].SessionID != ordered[j].SessionID {
			return ordered[i].SessionID < ordered[j].SessionID
		}
		if ordered[i].Sequence != ordered[j].Sequence {
			return ordered[i].Sequence < ordered[j].Sequence
		}
		if !ordered[i].OccurredAt.Equal(ordered[j].OccurredAt) {
			return ordered[i].OccurredAt.Before(ordered[j].OccurredAt)
		}
		return ordered[i].ID < ordered[j].ID
	})
	seen := make(map[string]bool)
	runs := make(map[string]*domain.Run)
	active := make(map[string]*domain.Run)
	for _, event := range ordered {
		if event.ID == "" || seen[event.ID] {
			continue
		}
		seen[event.ID] = true
		var payload eventPayload
		_ = json.Unmarshal(event.Payload, &payload)
		current := active[event.SessionID]
		switch event.Kind {
		case domain.EventPromptSubmitted, domain.EventRunStarted:
			if current != nil && current.ID != event.RunID {
				closeInferredRun(current, event.OccurredAt, domain.RunSuperseded, "new_run")
			}
			if event.RunID == "" {
				continue
			}
			run := runs[event.RunID]
			if run == nil {
				run = &domain.Run{ID: event.RunID, SessionID: event.SessionID, ProjectID: payload.ProjectID, StartedAt: event.OccurredAt, LastActivityAt: event.OccurredAt, Status: domain.RunRunning, Model: payload.Model, ReasoningEffort: payload.ReasoningEffort, ProjectionVersion: domain.ProjectionVersion}
				runs[event.RunID] = run
			}
			active[event.SessionID] = run
		case domain.EventTokenUsage:
			if current == nil || (event.RunID != "" && current.ID != event.RunID) {
				current = runs[event.RunID]
			}
			if current != nil {
				current.InputTokens = payload.InputTokens
				current.CachedInputTokens = payload.CachedInputTokens
				current.OutputTokens = payload.OutputTokens
				current.ReasoningTokens = payload.ReasoningTokens
				touch(current, event.OccurredAt)
			}
		case domain.EventRunCompleted:
			closeMatching(runs, active, event, domain.RunCompleted, "explicit_completion")
		case domain.EventRunInterrupted:
			closeMatching(runs, active, event, domain.RunInterrupted, "interrupted")
		case domain.EventRunFailed:
			closeMatching(runs, active, event, domain.RunFailed, "failed")
		case domain.EventSessionClosed:
			if current != nil {
				closeInferredRun(current, event.OccurredAt, domain.RunInterrupted, "session_closed")
				delete(active, event.SessionID)
			}
		default:
			if current != nil {
				if payload.Model != "" {
					current.Model = payload.Model
				}
				if payload.ReasoningEffort != "" {
					current.ReasoningEffort = payload.ReasoningEffort
				}
				touch(current, event.OccurredAt)
			}
		}
	}
	result := make([]domain.Run, 0, len(runs))
	for _, run := range runs {
		result = append(result, *run)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartedAt.Before(result[j].StartedAt) })
	return result
}

func ReconcileOpen(runs []domain.Run, now time.Time, grace, maximum time.Duration) []domain.Run {
	result := append([]domain.Run(nil), runs...)
	for i := range result {
		if result[i].EndedAt != nil {
			continue
		}
		deadline := result[i].LastActivityAt.Add(grace)
		maximumDeadline := result[i].StartedAt.Add(maximum)
		if maximumDeadline.Before(deadline) {
			deadline = maximumDeadline
		}
		if !now.Before(deadline) {
			closeRun(&result[i], deadline, domain.RunAbandoned, "lease_expired")
		}
	}
	return result
}

func closeMatching(runs map[string]*domain.Run, active map[string]*domain.Run, event domain.Event, status domain.RunStatus, reason string) {
	run := active[event.SessionID]
	if event.RunID != "" {
		run = runs[event.RunID]
	}
	if run == nil {
		return
	}
	// An explicit terminal event may correct an inferred closure.
	if run.EndedAt == nil || run.Status == domain.RunSuperseded || run.Status == domain.RunAbandoned {
		closeRun(run, event.OccurredAt, status, reason)
	}
	if active[event.SessionID] == run {
		delete(active, event.SessionID)
	}
}

func closeInferredRun(run *domain.Run, at time.Time, status domain.RunStatus, reason string) {
	deadline := run.LastActivityAt.Add(inferredClosureGrace)
	if deadline.Before(at) {
		at = deadline
	}
	closeRun(run, at, status, reason)
}

func closeRun(run *domain.Run, at time.Time, status domain.RunStatus, reason string) {
	if at.Before(run.StartedAt) {
		at = run.StartedAt
	}
	run.EndedAt, run.LastActivityAt, run.Status, run.CloseReason = &at, at, status, reason
}

func touch(run *domain.Run, at time.Time) {
	if at.After(run.LastActivityAt) {
		run.LastActivityAt = at
	}
}
