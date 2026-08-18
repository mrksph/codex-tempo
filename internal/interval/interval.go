package interval

import (
	"sort"
	"time"

	"github.com/mrksph/codex-tempo/internal/domain"
)

type Interval struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type Summary struct {
	AgentTime          time.Duration            `json:"agent_time"`
	SessionAgentTime   time.Duration            `json:"session_agent_time"`
	WallClock          time.Duration            `json:"wall_clock"`
	ProjectSpan        map[string]time.Duration `json:"project_span"`
	ParallelismPeak    int                      `json:"parallelism_peak"`
	ParallelismAverage float64                  `json:"parallelism_average"`
	RunCount           int                      `json:"run_count"`
	InputTokens        int64                    `json:"input_tokens"`
	CachedInputTokens  int64                    `json:"cached_input_tokens"`
	OutputTokens       int64                    `json:"output_tokens"`
	ReasoningTokens    int64                    `json:"reasoning_tokens"`
}

func Clip(iv Interval, from, to time.Time) (Interval, bool) {
	if iv.Start.Before(from) {
		iv.Start = from
	}
	if iv.End.After(to) {
		iv.End = to
	}
	return iv, iv.End.After(iv.Start)
}

func Merge(intervals []Interval) []Interval {
	valid := make([]Interval, 0, len(intervals))
	for _, iv := range intervals {
		if iv.End.After(iv.Start) {
			valid = append(valid, iv)
		}
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Start.Equal(valid[j].Start) {
			return valid[i].End.Before(valid[j].End)
		}
		return valid[i].Start.Before(valid[j].Start)
	})
	merged := make([]Interval, 0, len(valid))
	for _, iv := range valid {
		if len(merged) == 0 || iv.Start.After(merged[len(merged)-1].End) {
			merged = append(merged, iv)
			continue
		}
		if iv.End.After(merged[len(merged)-1].End) {
			merged[len(merged)-1].End = iv.End
		}
	}
	return merged
}

func Duration(intervals []Interval) time.Duration {
	var total time.Duration
	for _, iv := range intervals {
		total += iv.End.Sub(iv.Start)
	}
	return total
}

func Summarize(runs []domain.Run, from, to, now time.Time) Summary {
	result := Summary{ProjectSpan: make(map[string]time.Duration)}
	all := make([]Interval, 0, len(runs))
	byProject := make(map[string][]Interval)
	type point struct {
		at    time.Time
		delta int
	}
	points := make([]point, 0, len(runs)*2)
	for _, run := range runs {
		end := now
		if run.EndedAt != nil {
			end = *run.EndedAt
		}
		iv, ok := Clip(Interval{Start: run.StartedAt, End: end}, from, to)
		if !ok {
			if run.StartedAt.Equal(end) && !run.StartedAt.Before(from) && run.StartedAt.Before(to) {
				result.InputTokens += run.InputTokens
				result.CachedInputTokens += run.CachedInputTokens
				result.OutputTokens += run.OutputTokens
				result.ReasoningTokens += run.ReasoningTokens
			}
			continue
		}
		result.RunCount++
		result.SessionAgentTime += iv.End.Sub(iv.Start)
		result.InputTokens += run.InputTokens
		result.CachedInputTokens += run.CachedInputTokens
		result.OutputTokens += run.OutputTokens
		result.ReasoningTokens += run.ReasoningTokens
		all = append(all, iv)
		byProject[run.ProjectID] = append(byProject[run.ProjectID], iv)
		points = append(points, point{iv.Start, 1}, point{iv.End, -1})
	}
	result.WallClock = Duration(Merge(all))
	for project, values := range byProject {
		result.ProjectSpan[project] = Duration(Merge(values))
		result.AgentTime += result.ProjectSpan[project]
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].at.Equal(points[j].at) {
			return points[i].delta < points[j].delta
		}
		return points[i].at.Before(points[j].at)
	})
	active := 0
	var weighted time.Duration
	for i, p := range points {
		if i > 0 {
			weighted += time.Duration(active) * p.at.Sub(points[i-1].at)
		}
		active += p.delta
		if active > result.ParallelismPeak {
			result.ParallelismPeak = active
		}
	}
	if result.WallClock > 0 {
		result.ParallelismAverage = float64(weighted) / float64(result.WallClock)
	}
	return result
}
