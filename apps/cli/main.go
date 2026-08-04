package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mrksph/codex-tempo/internal/codex"
	"github.com/mrksph/codex-tempo/internal/config"
	"github.com/mrksph/codex-tempo/internal/domain"
	"github.com/mrksph/codex-tempo/internal/hooklog"
	"github.com/mrksph/codex-tempo/internal/interval"
	"github.com/mrksph/codex-tempo/internal/localdb"
	"github.com/mrksph/codex-tempo/internal/projector"
	"github.com/mrksph/codex-tempo/internal/projectresolver"
	ledgersync "github.com/mrksph/codex-tempo/internal/sync"
	"github.com/mrksph/codex-tempo/internal/wakapi"
)

func main() {
	configPath := flag.String("config", "", "configuration file")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
	}
	cfg, err := config.Load(*configPath)
	check(err)
	if args[0] == "hook" {
		runHookCommand(cfg)
		return
	}
	store, err := localdb.Open(filepath.Join(cfg.DataDir, "tempo.db"))
	check(err)
	defer store.Close()
	ctx := context.Background()
	switch args[0] {
	case "report":
		period := "today"
		if len(args) > 1 {
			period = args[1]
		}
		from, to := rangeFor(period, time.Now())
		events, err := store.Events(ctx, from.Add(-12*time.Hour), to)
		check(err)
		runs := projector.ReconcileOpen(projector.Project(events), time.Now(), 90*time.Second, 12*time.Hour)
		summary := interval.Summarize(runs, from, to, time.Now())
		projectSpan := make(map[string]string, len(summary.ProjectSpan))
		for projectID, duration := range summary.ProjectSpan {
			projectSpan[projectID] = duration.String()
		}
		printJSON(map[string]any{
			"agent_time": summary.AgentTime.String(), "wall_clock": summary.WallClock.String(),
			"project_span": projectSpan, "parallelism_peak": summary.ParallelismPeak,
			"parallelism_average": summary.ParallelismAverage, "run_count": summary.RunCount,
			"input_tokens": summary.InputTokens, "cached_input_tokens": summary.CachedInputTokens,
			"output_tokens": summary.OutputTokens, "reasoning_tokens": summary.ReasoningTokens,
		})
	case "timeline":
		from, to := rangeFor("today", time.Now())
		events, err := store.Events(ctx, from.Add(-12*time.Hour), to)
		check(err)
		printJSON(projector.Project(events))
	case "sessions":
		events, err := store.Events(ctx, time.Unix(0, 0), time.Now().Add(time.Second))
		check(err)
		runs := projector.Project(events)
		active := runs[:0]
		for _, r := range runs {
			if r.EndedAt == nil {
				active = append(active, r)
			}
		}
		printJSON(active)
	case "projects":
		events, err := store.Events(ctx, time.Unix(0, 0), time.Now().Add(time.Second))
		check(err)
		runs := projector.Project(events)
		seen := map[string]bool{}
		var ids []string
		for _, r := range runs {
			if !seen[r.ProjectID] {
				seen[r.ProjectID] = true
				ids = append(ids, r.ProjectID)
			}
		}
		sort.Strings(ids)
		printJSON(ids)
	case "doctor":
		diagnostics, err := store.Diagnostics(ctx)
		check(err)
		machineID, err := store.MachineID(ctx)
		check(err)
		printJSON(map[string]any{
			"machine_id": machineID, "database": filepath.Join(cfg.DataDir, "tempo.db"), "counts": diagnostics,
			"hook_log_dir": filepath.Join(cfg.DataDir, "logs"), "log_retention": cfg.LogRetention.String(),
		})
	case "reparse":
		check(store.ResetCursors(ctx))
		fmt.Println("cursors reset; run codex-tempo-agent --once")
	case "import":
		if len(args) < 2 || args[1] != "wakapi" {
			usage()
		}
		importWakapi(ctx, cfg, store, args[2:])
	default:
		usage()
	}
}

type hookInput struct {
	SessionID      string  `json:"session_id"`
	TurnID         string  `json:"turn_id"`
	CWD            string  `json:"cwd"`
	Model          string  `json:"model"`
	EventName      string  `json:"hook_event_name"`
	TranscriptPath *string `json:"transcript_path"`
}

func runHookCommand(cfg config.Config) {
	startedAt := time.Now()
	fileLogger, logErr := hooklog.Open(cfg.DataDir, cfg.LogRetention)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	if logErr == nil {
		defer fileLogger.Close()
		logger = fileLogger.Logger
	}

	var input hookInput
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 1<<20)).Decode(&input); err != nil {
		logger.Error("hook completed", "status", "error", "stage", "decode_input", "error", err, "duration_ms", time.Since(startedAt).Milliseconds())
		return
	}
	if input.SessionID == "" {
		logger.Error("hook completed", "status", "error", "stage", "validate_input", "error", "session_id is required", "duration_ms", time.Since(startedAt).Milliseconds())
		return
	}
	logger.Info("hook started", "hook_event", input.EventName, "session_id", input.SessionID, "turn_id", input.TurnID, "pid", os.Getpid())
	store, err := localdb.Open(filepath.Join(cfg.DataDir, "tempo.db"))
	if err != nil {
		logger.Error("hook completed", "status", "error", "stage", "open_store", "error", err, "hook_event", input.EventName, "session_id", input.SessionID, "duration_ms", time.Since(startedAt).Milliseconds())
		return
	}
	defer store.Close()
	runHook(cfg, store, logger, input, startedAt)
}

func runHook(cfg config.Config, store *localdb.Store, logger *slog.Logger, input hookInput, startedAt time.Time) {
	status, stage := "ok", "machine_id"
	var finalErr error
	var project projectresolver.Result
	eventsEnqueued, metricEvents := 0, 0
	syncAttempted, syncSucceeded := false, false
	defer func() {
		attributes := []any{
			"status", status, "stage", stage, "hook_event", input.EventName,
			"session_id", input.SessionID, "turn_id", input.TurnID,
			"project_id", project.ID, "project_name", project.Name,
			"worktree_name", project.WorktreeName, "is_worktree", project.IsWorktree,
			"events_enqueued", eventsEnqueued, "metric_events", metricEvents,
			"sync_attempted", syncAttempted, "sync_succeeded", syncSucceeded,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		}
		if cfg.Privacy.StorePaths {
			attributes = append(attributes, "worktree_path", project.WorktreePath)
		}
		if finalErr != nil {
			attributes = append(attributes, "error", finalErr)
			logger.Error("hook completed", attributes...)
			return
		}
		logger.Info("hook completed", attributes...)
	}()
	fail := func(failedStage string, err error) {
		status, stage, finalErr = "error", failedStage, err
	}
	warn := func(failedStage string, err error) {
		status = "degraded"
		logger.Warn("hook step failed", "stage", failedStage, "error", err, "hook_event", input.EventName, "session_id", input.SessionID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	machineID, err := store.MachineID(ctx)
	if err != nil {
		fail("machine_id", err)
		return
	}
	if input.CWD == "" {
		input.CWD, _ = os.Getwd()
	}
	now := time.Now().UTC()
	cutover, err := store.EnsureTimeMetadata(ctx, "hook_cutover_at", now)
	if err != nil {
		fail("ensure_cutover", err)
		return
	}
	project = projectresolver.Resolve(ctx, input.CWD, machineID, cfg.Privacy.StorePaths)
	stage = "heartbeat"
	item, err := store.AdvanceHookHeartbeat(ctx, localdb.HookHeartbeat{
		SessionID: input.SessionID, ProjectID: project.ID, ProjectName: project.Name,
		ProjectFingerprint: project.Fingerprint, RemoteHash: project.RemoteHash,
		WorktreeName: project.WorktreeName, WorktreePath: project.WorktreePath, IsWorktree: project.IsWorktree,
		Model: input.Model, OccurredAt: now,
	}, cfg.HookActivityTimeout)
	if err != nil {
		fail("heartbeat", err)
		return
	}
	if item != nil {
		events := hookIntervalEvents(machineID, *item)
		if err = store.Enqueue(ctx, events); err != nil {
			fail("enqueue", err)
			return
		}
		eventsEnqueued = len(events)
	}
	if input.TranscriptPath != nil && *input.TranscriptPath != "" {
		cursor, cursorErr := store.MetricCursor(ctx, *input.TranscriptPath)
		if cursorErr != nil {
			warn("metric_cursor", cursorErr)
		} else {
			scanner := codex.MetricScanner{MachineID: machineID, StorePaths: cfg.Privacy.StorePaths}
			cursor, events, scanErr := scanner.Scan(ctx, *input.TranscriptPath, cursor, cutover)
			if scanErr != nil {
				warn("metric_scan", scanErr)
			} else if commitErr := store.CommitMetrics(ctx, cursor, events); commitErr != nil {
				warn("metric_commit", commitErr)
			} else {
				metricEvents = len(events)
			}
		}
	}
	if cfg.ServerURL == "" || cfg.MachineToken == "" {
		stage = "complete"
		return
	}
	lastSync, hasLastSync, err := store.TimeMetadata(ctx, "hook_last_sync_at")
	if err != nil {
		fail("sync_metadata", err)
		return
	}
	if hasLastSync && now.Sub(lastSync) < cfg.HookSyncInterval {
		stage = "complete"
		return
	}
	stage, syncAttempted = "sync", true
	client := &ledgersync.Client{Store: store, ServerURL: cfg.ServerURL, Token: cfg.MachineToken, MachineID: machineID, HTTP: &http.Client{Timeout: time.Second}}
	if err = client.SyncOnce(ctx); err != nil {
		fail("sync", err)
		return
	}
	syncSucceeded = true
	if err = store.SetTimeMetadata(ctx, "hook_last_sync_at", now); err != nil {
		fail("sync_metadata_write", err)
		return
	}
	stage = "complete"
}

func hookIntervalEvents(machineID string, item localdb.HookInterval) []domain.Event {
	runID := domain.DeterministicUUID("hook-run", item.SessionID, item.ProjectID, item.StartedAt.Format(time.RFC3339Nano), item.EndedAt.Format(time.RFC3339Nano))
	sessionID := "hook:" + runID
	payload, _ := json.Marshal(map[string]any{
		"project_id": item.ProjectID, "project_name": item.ProjectName,
		"project_fingerprint": item.ProjectFingerprint, "remote_hash": item.RemoteHash,
		"worktree_name": item.WorktreeName, "worktree_path": item.WorktreePath,
		"is_worktree": item.IsWorktree, "model": item.Model,
	})
	return []domain.Event{
		{ID: domain.DeterministicUUID("hook-event", runID, "session_started"), MachineID: machineID, SessionID: sessionID, Sequence: 1, OccurredAt: item.StartedAt, Kind: domain.EventSessionStarted, Source: "hook", Payload: payload},
		{ID: domain.DeterministicUUID("hook-event", runID, "run_started"), MachineID: machineID, SessionID: sessionID, RunID: runID, Sequence: 2, OccurredAt: item.StartedAt, Kind: domain.EventRunStarted, Source: "hook", Payload: payload},
		{ID: domain.DeterministicUUID("hook-event", runID, "run_completed"), MachineID: machineID, SessionID: sessionID, RunID: runID, Sequence: 3, OccurredAt: item.EndedAt, Kind: domain.EventRunCompleted, Source: "hook", Payload: json.RawMessage(`{}`)},
	}
}

func importWakapi(ctx context.Context, cfg config.Config, store *localdb.Store, args []string) {
	flags := flag.NewFlagSet("import wakapi", flag.ExitOnError)
	home, _ := os.UserHomeDir()
	wakatimeConfig := flags.String("wakatime-config", filepath.Join(home, ".wakatime.cfg"), "WakaTime configuration file")
	from := flags.String("from", "", "first date to import (YYYY-MM-DD)")
	to := flags.String("to", "", "last date to import (YYYY-MM-DD)")
	timeout := flags.Duration("timeout", 10*time.Minute, "Wakapi heartbeat timeout")
	_ = flags.Parse(args)
	if cfg.ServerURL == "" || cfg.MachineToken == "" {
		check(fmt.Errorf("configure the agent before importing Wakapi"))
	}
	credentials, err := wakapi.LoadCredentials(*wakatimeConfig)
	check(err)
	exportCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	exported, err := (wakapi.Client{Credentials: credentials}).Export(exportCtx, *from, *to, *timeout)
	check(err)
	if cutover, ok, metadataErr := store.TimeMetadata(exportCtx, "hook_cutover_at"); metadataErr != nil {
		check(metadataErr)
	} else if ok {
		exported.Intervals = intervalsBefore(exported.Intervals, cutover)
		exported.Duration = 0
		for _, item := range exported.Intervals {
			exported.Duration += item.EndedAt.Sub(item.StartedAt)
		}
	}
	machineID, err := store.MachineID(exportCtx)
	check(err)
	events := wakapiEvents(machineID, exported.Intervals)
	check(store.Enqueue(exportCtx, events))
	client := &ledgersync.Client{Store: store, ServerURL: cfg.ServerURL, Token: cfg.MachineToken, MachineID: machineID}
	for {
		pending, err := store.Pending(exportCtx, 1)
		check(err)
		if len(pending) == 0 {
			break
		}
		check(client.SyncOnce(exportCtx))
	}
	printJSON(map[string]any{"source": "wakapi", "from": exported.From, "to": exported.To, "heartbeats": exported.Heartbeats, "intervals": len(exported.Intervals), "seconds": exported.Duration.Seconds(), "events": len(events)})
}

func intervalsBefore(intervals []wakapi.Interval, cutover time.Time) []wakapi.Interval {
	result := make([]wakapi.Interval, 0, len(intervals))
	for _, item := range intervals {
		if !item.StartedAt.Before(cutover) {
			continue
		}
		if item.EndedAt.After(cutover) {
			item.EndedAt = cutover
		}
		if item.EndedAt.After(item.StartedAt) {
			result = append(result, item)
		}
	}
	return result
}

func wakapiEvents(machineID string, intervals []wakapi.Interval) []domain.Event {
	events := make([]domain.Event, 0, len(intervals)*3)
	for _, item := range intervals {
		projectID := domain.DeterministicUUID("wakapi-project", item.Project)
		runID := domain.DeterministicUUID("wakapi-run", item.Project, item.StartedAt.Format(time.RFC3339Nano))
		sessionID := "wakapi:" + runID
		payload, _ := json.Marshal(map[string]string{"project_id": projectID, "project_name": item.Project, "project_fingerprint": "wakapi:" + strings.ToLower(item.Project)})
		events = append(events,
			domain.Event{ID: domain.DeterministicUUID("wakapi-event", runID, "session_started"), MachineID: machineID, SessionID: sessionID, Sequence: 1, OccurredAt: item.StartedAt, Kind: domain.EventSessionStarted, Source: "wakapi", Payload: payload},
			domain.Event{ID: domain.DeterministicUUID("wakapi-event", runID, "run_started"), MachineID: machineID, SessionID: sessionID, RunID: runID, Sequence: 2, OccurredAt: item.StartedAt, Kind: domain.EventRunStarted, Source: "wakapi", Payload: payload},
			domain.Event{ID: domain.DeterministicUUID("wakapi-event", runID, "run_completed"), MachineID: machineID, SessionID: sessionID, RunID: runID, Sequence: 3, OccurredAt: item.EndedAt, Kind: domain.EventRunCompleted, Source: "wakapi", Payload: json.RawMessage(`{}`)},
		)
	}
	return events
}

func rangeFor(period string, now time.Time) (time.Time, time.Time) {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if period == "week" {
		weekday := (int(today.Weekday()) + 6) % 7
		from := today.AddDate(0, 0, -weekday)
		return from, from.AddDate(0, 0, 7)
	}
	return today, today.AddDate(0, 0, 1)
}
func printJSON(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	check(err)
	fmt.Println(string(data))
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: codex-tempo [report today|week|timeline|sessions|projects|doctor|reparse|import wakapi|hook]")
	os.Exit(2)
}
