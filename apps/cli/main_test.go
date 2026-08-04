package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mrksph/codex-tempo/internal/localdb"
	"github.com/mrksph/codex-tempo/internal/projector"
	"github.com/mrksph/codex-tempo/internal/wakapi"
)

func TestHookIntervalEventsCreateBoundedCompletedRun(t *testing.T) {
	start := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	events := hookIntervalEvents("machine", localdb.HookInterval{
		SessionID: "codex-session", ProjectID: "project", ProjectName: "tempo",
		WorktreeName: "feature", WorktreePath: "/workspace/feature", IsWorktree: true,
		StartedAt: start, EndedAt: start.Add(25 * time.Second), Model: "gpt-5",
	})
	runs := projector.Project(events)
	if len(events) != 3 || len(runs) != 1 {
		t.Fatalf("events = %d, runs = %d", len(events), len(runs))
	}
	run := runs[0]
	if run.EndedAt == nil || run.EndedAt.Sub(run.StartedAt) != 25*time.Second || run.Model != "gpt-5" {
		t.Fatalf("run = %#v", run)
	}
	for _, event := range events {
		if event.Source != "hook" {
			t.Fatalf("source = %q", event.Source)
		}
	}
	var payload struct {
		WorktreeName string `json:"worktree_name"`
		WorktreePath string `json:"worktree_path"`
		IsWorktree   bool   `json:"is_worktree"`
	}
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.WorktreeName != "feature" || payload.WorktreePath != "/workspace/feature" || !payload.IsWorktree {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestIntervalsBeforeCutoverDropsAndClipsFutureActivity(t *testing.T) {
	cutover := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	intervals := []wakapi.Interval{
		{Project: "a", StartedAt: cutover.Add(-time.Minute), EndedAt: cutover.Add(time.Minute)},
		{Project: "b", StartedAt: cutover, EndedAt: cutover.Add(time.Minute)},
	}
	got := intervalsBefore(intervals, cutover)
	if len(got) != 1 || !got[0].EndedAt.Equal(cutover) {
		t.Fatalf("intervals = %#v", got)
	}
}
