package projector

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/mrksph/codex-tempo/internal/domain"
)

func TestProjectionIsIndependentAndIdempotent(t *testing.T) {
	base := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	payloadA, _ := json.Marshal(map[string]string{"project_id": "A"})
	payloadB, _ := json.Marshal(map[string]string{"project_id": "B"})
	events := []domain.Event{
		{ID: "a2", SessionID: "sa", RunID: "ra", Sequence: 2, OccurredAt: base.Add(20 * time.Minute), Kind: domain.EventRunCompleted},
		{ID: "b1", SessionID: "sb", RunID: "rb", Sequence: 1, OccurredAt: base.Add(5 * time.Minute), Kind: domain.EventRunStarted, Payload: payloadB},
		{ID: "a1", SessionID: "sa", RunID: "ra", Sequence: 1, OccurredAt: base, Kind: domain.EventRunStarted, Payload: payloadA},
		{ID: "b2", SessionID: "sb", RunID: "rb", Sequence: 2, OccurredAt: base.Add(25 * time.Minute), Kind: domain.EventRunCompleted},
	}
	withDuplicate := append(append([]domain.Event{}, events...), events[0])
	if got, want := Project(withDuplicate), Project(events); !reflect.DeepEqual(got, want) {
		t.Fatalf("projection changed with duplicate:\n%#v\n%#v", got, want)
	}
	if len(Project(events)) != 2 {
		t.Fatal("parallel sessions interfered")
	}
}

func TestLateCompletionCorrectsAbandonedRun(t *testing.T) {
	base := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	events := []domain.Event{{ID: "start", SessionID: "s", RunID: "r", Sequence: 1, OccurredAt: base, Kind: domain.EventRunStarted}}
	runs := ReconcileOpen(Project(events), base.Add(5*time.Minute), 90*time.Second, 12*time.Hour)
	if runs[0].Status != domain.RunAbandoned {
		t.Fatalf("status = %s", runs[0].Status)
	}
	events = append(events, domain.Event{ID: "end", SessionID: "s", RunID: "r", Sequence: 2, OccurredAt: base.Add(3 * time.Minute), Kind: domain.EventRunCompleted})
	if got := Project(events)[0]; got.Status != domain.RunCompleted || got.EndedAt.Sub(base) != 3*time.Minute {
		t.Fatalf("late completion not applied: %#v", got)
	}
}

func TestNewRunDoesNotCountLongInactiveGap(t *testing.T) {
	base := time.Date(2026, 8, 4, 1, 0, 0, 0, time.UTC)
	events := []domain.Event{
		{ID: "start-a", SessionID: "s", RunID: "a", Sequence: 1, OccurredAt: base, Kind: domain.EventRunStarted},
		{ID: "activity-a", SessionID: "s", RunID: "a", Sequence: 2, OccurredAt: base.Add(time.Minute), Kind: domain.EventLease},
		{ID: "start-b", SessionID: "s", RunID: "b", Sequence: 3, OccurredAt: base.Add(10 * time.Hour), Kind: domain.EventRunStarted},
	}
	runs := Project(events)
	if runs[0].EndedAt == nil || runs[0].EndedAt.Sub(base) != 2*time.Minute+30*time.Second {
		t.Fatalf("inferred end = %#v", runs[0].EndedAt)
	}
}
