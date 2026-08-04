package localdb

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mrksph/codex-tempo/internal/domain"
)

func TestQueueIsIdempotentAndDurable(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	machineID, err := store.MachineID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	event := domain.Event{ID: uuid.NewString(), MachineID: machineID, SessionID: "session", Sequence: 1, OccurredAt: time.Now(), Kind: domain.EventRunStarted, Source: "test"}
	cursor := Cursor{Path: "fixture", FileIdentity: "1:1", ByteOffset: 42, LastSequence: 1, LastModified: time.Now()}
	if err = store.CommitParsed(ctx, cursor, []domain.Event{event, event}); err != nil {
		t.Fatal(err)
	}
	pending, err := store.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending = %d", len(pending))
	}
	if err = store.MarkAcknowledged(ctx, []string{event.ID}); err != nil {
		t.Fatal(err)
	}
	pending, err = store.Pending(ctx, 10)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending after ack = %d, %v", len(pending), err)
	}
}

func TestResetCursorsAlsoClearsParserErrors(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.CommitParsed(ctx, Cursor{Path: "fixture", FileIdentity: "1:1", LastModified: time.Now()}, nil); err != nil {
		t.Fatal(err)
	}
	if err = store.RecordParserError(ctx, "fixture", 42, context.DeadlineExceeded); err != nil {
		t.Fatal(err)
	}
	if err = store.ResetCursors(ctx); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := store.Diagnostics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics["cursors"] != 0 || diagnostics["parser_errors"] != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestEnqueueIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	event := domain.Event{ID: uuid.NewString(), MachineID: uuid.NewString(), SessionID: "wakapi-session", Sequence: 1, OccurredAt: time.Now(), Kind: domain.EventRunStarted, Source: "wakapi"}
	if err = store.Enqueue(ctx, []domain.Event{event, event}); err != nil {
		t.Fatal(err)
	}
	pending, err := store.Pending(ctx, 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %d, %v", len(pending), err)
	}
}

func TestEnqueueUpdatesAcknowledgedWakapiEvent(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	event := domain.Event{ID: uuid.NewString(), MachineID: uuid.NewString(), SessionID: "wakapi-session", Sequence: 3, OccurredAt: time.Now().UTC(), Kind: domain.EventRunCompleted, Source: "wakapi"}
	if err = store.Enqueue(ctx, []domain.Event{event}); err != nil {
		t.Fatal(err)
	}
	if err = store.MarkAcknowledged(ctx, []string{event.ID}); err != nil {
		t.Fatal(err)
	}
	event.OccurredAt = event.OccurredAt.Add(time.Minute)
	if err = store.Enqueue(ctx, []domain.Event{event}); err != nil {
		t.Fatal(err)
	}
	pending, err := store.Pending(ctx, 10)
	if err != nil || len(pending) != 1 || !pending[0].OccurredAt.Equal(event.OccurredAt) {
		t.Fatalf("pending = %#v, %v", pending, err)
	}
}

func TestHookHeartbeatOnlyCreatesIntervalsWithinTimeout(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	base := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	heartbeat := HookHeartbeat{
		SessionID: "session", ProjectID: "project-a", ProjectName: "alpha",
		WorktreeName: "feature", WorktreePath: "/workspace/feature", IsWorktree: true,
		OccurredAt: base,
	}

	interval, err := store.AdvanceHookHeartbeat(ctx, heartbeat, 90*time.Second)
	if err != nil || interval != nil {
		t.Fatalf("first heartbeat = %#v, %v", interval, err)
	}
	heartbeat.OccurredAt = base.Add(30 * time.Second)
	interval, err = store.AdvanceHookHeartbeat(ctx, heartbeat, 90*time.Second)
	if err != nil || interval == nil || interval.EndedAt.Sub(interval.StartedAt) != 30*time.Second ||
		interval.WorktreeName != "feature" || interval.WorktreePath != "/workspace/feature" || !interval.IsWorktree {
		t.Fatalf("active interval = %#v, %v", interval, err)
	}

	heartbeat.ProjectID, heartbeat.ProjectName = "project-b", "beta"
	heartbeat.OccurredAt = base.Add(3 * time.Minute)
	interval, err = store.AdvanceHookHeartbeat(ctx, heartbeat, 90*time.Second)
	if err != nil || interval != nil {
		t.Fatalf("idle gap = %#v, %v", interval, err)
	}
	heartbeat.OccurredAt = base.Add(3*time.Minute + 20*time.Second)
	interval, err = store.AdvanceHookHeartbeat(ctx, heartbeat, 90*time.Second)
	if err != nil || interval == nil || interval.ProjectID != "project-b" || interval.EndedAt.Sub(interval.StartedAt) != 20*time.Second {
		t.Fatalf("resumed interval = %#v, %v", interval, err)
	}
}

func TestOpenKeepsSQLiteWaitBelowHookTimeout(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var timeoutMS int
	if err = store.db.QueryRow("PRAGMA busy_timeout").Scan(&timeoutMS); err != nil {
		t.Fatal(err)
	}
	if timeoutMS != 1000 {
		t.Fatalf("busy timeout = %dms", timeoutMS)
	}
	var version int
	if err = store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %d", version)
	}
}

func TestOpenDoesNotMigrateCurrentSchemaDuringAnotherWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tempo.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO metadata(key,value) VALUES('write-lock','held')"); err != nil {
		t.Fatal(err)
	}

	other, err := Open(path)
	if err != nil {
		t.Fatalf("open current schema while write is active: %v", err)
	}
	other.Close()
}

func TestConcurrentHeartbeatsSerializeWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tempo.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()

	const workers = 30
	start := make(chan struct{})
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			workerStore, openErr := Open(path)
			if openErr != nil {
				errors <- openErr
				return
			}
			defer workerStore.Close()
			<-start
			_, heartbeatErr := workerStore.AdvanceHookHeartbeat(context.Background(), HookHeartbeat{
				SessionID: uuid.NewString(), ProjectID: "project", ProjectName: "tempo", OccurredAt: time.Now().UTC(),
			}, 90*time.Second)
			if heartbeatErr != nil {
				errors <- heartbeatErr
			}
		}()
	}
	close(start)
	group.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent heartbeat: %v", err)
	}
}

func TestTimeMetadataKeepsFirstCutoverAndUpdatesSyncTime(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	got, err := store.EnsureTimeMetadata(ctx, "cutover", first)
	if err != nil || !got.Equal(first) {
		t.Fatalf("first = %v, %v", got, err)
	}
	got, err = store.EnsureTimeMetadata(ctx, "cutover", first.Add(time.Hour))
	if err != nil || !got.Equal(first) {
		t.Fatalf("preserved = %v, %v", got, err)
	}
	if err = store.SetTimeMetadata(ctx, "sync", first); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.TimeMetadata(ctx, "sync")
	if err != nil || !ok || !got.Equal(first) {
		t.Fatalf("sync = %v, %t, %v", got, ok, err)
	}
}
