package codex

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrksph/codex-tempo/internal/localdb"
)

func TestParseFileIgnoresContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout.jsonl")
	data := `{"timestamp":"2026-08-04T10:00:00Z","type":"session_meta","payload":{"id":"session-1","cwd":"` + dir + `","cli_version":"1.0"}}` + "\n" +
		`{"timestamp":"2026-08-04T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-1","started_at":"2026-08-04T10:00:01Z"}}` + "\n" +
		`{"timestamp":"2026-08-04T10:00:02Z","type":"event_msg","payload":{"type":"user_message","message":"SECRET"}}` + "\n" +
		`{"timestamp":"2026-08-04T10:01:00Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1","completed_at":"2026-08-04T10:01:00Z"}}` + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cursor, events, errs, err := (Parser{MachineID: "machine"}).ParseFile(context.Background(), path, localdb.Cursor{Path: path})
	if err != nil || len(errs) != 0 {
		t.Fatalf("parse errors: %v %v", err, errs)
	}
	if len(events) != 3 {
		t.Fatalf("events = %d", len(events))
	}
	if cursor.SessionID != "session-1" || cursor.LastSequence != 3 {
		t.Fatalf("cursor = %#v", cursor)
	}
	for _, event := range events {
		if bytes.Contains(event.Payload, []byte("SECRET")) {
			t.Fatal("content was retained")
		}
	}
	_, next, _, err := (Parser{MachineID: "machine"}).ParseFile(context.Background(), path, cursor)
	if err != nil || len(next) != 0 {
		t.Fatalf("incremental parse returned %d events: %v", len(next), err)
	}
}

func TestParseFileAcceptsUnixTaskTimestamps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout.jsonl")
	data := `{"timestamp":"2026-08-04T10:00:00Z","type":"session_meta","payload":{"id":"session-1","cwd":"` + dir + `"}}` + "\n" +
		`{"timestamp":"2026-08-04T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-1","started_at":1785837601}}` + "\n" +
		`{"timestamp":"2026-08-04T10:01:00Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1","completed_at":1785837660.25}}` + "\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	_, events, errs, err := (Parser{MachineID: "machine"}).ParseFile(context.Background(), path, localdb.Cursor{Path: path})
	if err != nil || len(errs) != 0 {
		t.Fatalf("parse errors: %v %v", err, errs)
	}
	if len(events) != 3 {
		t.Fatalf("events = %d", len(events))
	}
	if got, want := events[1].OccurredAt, time.Unix(1785837601, 0).UTC(); !got.Equal(want) {
		t.Fatalf("started at = %s, want %s", got, want)
	}
	if got, want := events[2].OccurredAt, time.Unix(1785837660, 250_000_000).UTC(); !got.Equal(want) {
		t.Fatalf("completed at = %s, want %s", got, want)
	}
}
