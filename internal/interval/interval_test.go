package interval

import (
	"testing"
	"time"

	"github.com/mrksph/codex-tempo/internal/domain"
)

func TestParallelRuns(t *testing.T) {
	base := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	endA, endB := base.Add(20*time.Minute), base.Add(25*time.Minute)
	runs := []domain.Run{
		{ID: "a", ProjectID: "A", StartedAt: base, EndedAt: &endA},
		{ID: "b", ProjectID: "B", StartedAt: base.Add(5 * time.Minute), EndedAt: &endB},
	}
	got := Summarize(runs, base, base.Add(time.Hour), base.Add(time.Hour))
	if got.AgentTime != 40*time.Minute {
		t.Fatalf("agent time = %s", got.AgentTime)
	}
	if got.WallClock != 25*time.Minute {
		t.Fatalf("wall clock = %s", got.WallClock)
	}
	if got.ProjectSpan["A"] != 20*time.Minute || got.ProjectSpan["B"] != 20*time.Minute {
		t.Fatalf("project spans = %#v", got.ProjectSpan)
	}
	if got.ParallelismPeak != 2 {
		t.Fatalf("peak = %d", got.ParallelismPeak)
	}
	if got.ParallelismAverage != 1.6 {
		t.Fatalf("average = %f", got.ParallelismAverage)
	}
}

func TestHalfOpenIntervalsDoNotOverlap(t *testing.T) {
	base := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	a, b := base.Add(time.Minute), base.Add(2*time.Minute)
	runs := []domain.Run{{StartedAt: base, EndedAt: &a}, {StartedAt: a, EndedAt: &b}}
	got := Summarize(runs, base, b, b)
	if got.ParallelismPeak != 1 || got.WallClock != 2*time.Minute {
		t.Fatalf("unexpected summary: %#v", got)
	}
}
