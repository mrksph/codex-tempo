package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mrksph/codex-tempo/internal/domain"
	"github.com/mrksph/codex-tempo/internal/localdb"
)

func TestSyncAcknowledgesAcceptedEvents(t *testing.T) {
	ctx := context.Background()
	store, err := localdb.Open(filepath.Join(t.TempDir(), "tempo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	machineID, _ := store.MachineID(ctx)
	event := domain.Event{ID: uuid.NewString(), MachineID: machineID, SessionID: "session", Sequence: 1, OccurredAt: time.Now(), Kind: domain.EventLease, Source: "test"}
	if err = store.CommitParsed(ctx, localdb.Cursor{Path: "fixture", FileIdentity: "1", LastModified: time.Now()}, []domain.Event{event}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		var request ingestRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.MachineID != machineID || len(request.Events) != 1 {
			t.Errorf("request = %#v", request)
		}
		_ = json.NewEncoder(w).Encode(ingestResponse{Accepted: 1})
	}))
	defer server.Close()
	client := Client{Store: store, ServerURL: server.URL, Token: "secret", MachineID: machineID}
	if err = client.SyncOnce(ctx); err != nil {
		t.Fatal(err)
	}
	pending, err := store.Pending(ctx, 10)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending = %d, %v", len(pending), err)
	}
}
