package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrksph/codex-tempo/internal/interval"
	"github.com/mrksph/codex-tempo/internal/localdb"
	"github.com/mrksph/codex-tempo/internal/projector"
)

func TestMetricScannerEmitsTokensWithoutDuration(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout.jsonl")
	content := "" +
		"{\"timestamp\":\"2026-08-04T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":\"session\",\"cwd\":\"" + dir + "\"}}\n" +
		"{\"timestamp\":\"2026-08-04T10:00:01Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"task_started\",\"turn_id\":\"turn\"}}\n" +
		"{\"timestamp\":\"2026-08-04T10:00:02Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"info\":{\"total_token_usage\":{\"input_tokens\":100,\"cached_input_tokens\":40,\"output_tokens\":20,\"reasoning_output_tokens\":5}}}}\n" +
		"{\"timestamp\":\"2026-08-04T10:00:03Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"task_complete\",\"turn_id\":\"turn\",\"completed_at\":\"2026-08-04T10:00:03Z\"}}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := localdb.Open(filepath.Join(dir, "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cursor, err := store.MetricCursor(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	cursor, events, err := (MetricScanner{MachineID: "machine"}).Scan(ctx, path, cursor, time.Time{})
	if err != nil || len(events) != 4 {
		t.Fatalf("events = %d, err = %v", len(events), err)
	}
	runs := projector.Project(events)
	summary := interval.Summarize(runs, time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), time.Now())
	if summary.AgentTime != 0 || summary.InputTokens != 100 || summary.CachedInputTokens != 40 || summary.OutputTokens != 20 || summary.ReasoningTokens != 5 {
		t.Fatalf("summary = %#v", summary)
	}
	if err = store.CommitMetrics(ctx, cursor, events); err != nil {
		t.Fatal(err)
	}
}
