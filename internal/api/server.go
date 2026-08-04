package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/mrksph/codex-tempo/internal/domain"
	"github.com/mrksph/codex-tempo/internal/interval"
	"github.com/mrksph/codex-tempo/internal/postgres"
)

type Server struct {
	Store                     *postgres.Store
	AdminToken, InternalToken string
	Logger                    *slog.Logger
}
type ingestRequest struct {
	MachineID string         `json:"machine_id"`
	Events    []domain.Event `json:"events"`
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(30*time.Second))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", s.ready)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/ingest/register", s.register)
		r.Post("/ingest/events", s.ingest)
		r.Group(func(r chi.Router) {
			r.Use(s.internalAuth)
			r.Get("/projects", s.projects)
			r.Get("/reports/summary", s.summary)
			r.Get("/reports/timeline", s.timeline)
			r.Get("/runs", s.timeline)
			r.Get("/runs/active", s.activeRuns)
			r.Get("/sessions", s.sessions)
			r.Get("/machines", s.machines)
			r.Post("/admin/rebuild", s.rebuild)
		})
	})
	return r
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Pool.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if s.AdminToken != "" && !bearerMatches(r, s.AdminToken) {
		writeError(w, http.StatusUnauthorized, "invalid admin token")
		return
	}
	var input struct {
		MachineID string `json:"machine_id"`
		Name      string `json:"name"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		input.Name = "codex-machine"
	}
	id, token, err := s.Store.RegisterMachine(r.Context(), input.MachineID, input.Name)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"machine_id": id, "token": token})
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	var input ingestRequest
	if err := decodeJSON(w, r, &input); err != nil {
		s.logIngest(r, input.MachineID, 0, nil, time.Since(startedAt), err)
		return
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if err := s.Store.AuthenticateMachine(r.Context(), input.MachineID, token); err != nil {
		s.logIngest(r, input.MachineID, len(input.Events), nil, time.Since(startedAt), err)
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if len(input.Events) > 5000 {
		s.logIngest(r, input.MachineID, len(input.Events), nil, time.Since(startedAt), errors.New("batch exceeds 5000 events"))
		writeError(w, http.StatusRequestEntityTooLarge, "batch exceeds 5000 events")
		return
	}
	result, err := s.Store.Ingest(r.Context(), input.MachineID, input.Events)
	if err != nil {
		s.logIngest(r, input.MachineID, len(input.Events), nil, time.Since(startedAt), err)
		s.fail(w, r, err)
		return
	}
	s.logIngest(r, input.MachineID, len(input.Events), &result, time.Since(startedAt), nil)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) logIngest(r *http.Request, machineID string, eventCount int, result *postgres.IngestResult, duration time.Duration, err error) {
	if s.Logger == nil {
		return
	}
	attributes := []any{
		"request_id", middleware.GetReqID(r.Context()), "machine_id", machineID,
		"event_count", eventCount, "duration_ms", duration.Milliseconds(),
	}
	if result != nil {
		attributes = append(attributes, "accepted", result.Accepted, "duplicates", result.Duplicates, "rejected", len(result.Rejected))
	}
	if err != nil {
		attributes = append(attributes, "error", err)
		s.Logger.Warn("event ingest failed", attributes...)
		return
	}
	s.Logger.Info("events ingested", attributes...)
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	values, err := s.Store.Projects(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": values})
}
func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	values, err := s.Store.Sessions(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": values})
}
func (s *Server) machines(w http.ResponseWriter, r *http.Request) {
	values, err := s.Store.Machines(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"machines": values})
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	from, to, err := dateRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	projectID, err := optionalProjectID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	runs, err := s.Store.Runs(r.Context(), from, to, false, projectID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	value := interval.Summarize(runs, from, to, time.Now())
	projectSeconds := map[string]float64{}
	for id, duration := range value.ProjectSpan {
		projectSeconds[id] = duration.Seconds()
	}
	writeJSON(w, http.StatusOK, map[string]any{"from": from, "to": to, "agent_seconds": value.AgentTime.Seconds(), "wall_clock_seconds": value.WallClock.Seconds(), "project_span_seconds": projectSeconds, "parallelism_peak": value.ParallelismPeak, "parallelism_average": value.ParallelismAverage, "run_count": value.RunCount, "input_tokens": value.InputTokens, "cached_input_tokens": value.CachedInputTokens, "output_tokens": value.OutputTokens, "reasoning_tokens": value.ReasoningTokens})
}

func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	from, to, err := dateRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	projectID, err := optionalProjectID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	runs, err := s.Store.Runs(r.Context(), from, to, false, projectID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"from": from, "to": to, "runs": runs})
}
func (s *Server) activeRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.Store.Runs(r.Context(), time.Unix(0, 0), time.Now().Add(24*time.Hour), true, "")
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func optionalProjectID(r *http.Request) (string, error) {
	value := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if value == "" {
		return "", nil
	}
	if _, err := uuid.Parse(value); err != nil {
		return "", errors.New("project_id must be a valid UUID")
	}
	return value, nil
}
func (s *Server) rebuild(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.RebuildAll(r.Context()); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "rebuilt"})
}

func (s *Server) internalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.InternalToken != "" && !bearerMatches(r, s.InternalToken) {
			writeError(w, http.StatusUnauthorized, "invalid internal token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func bearerMatches(r *http.Request, token string) bool {
	return r.Header.Get("Authorization") == "Bearer "+token
}

func dateRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	var err error
	if value := r.URL.Query().Get("from"); value != "" {
		from, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return from, to, errors.New("from must be RFC3339")
		}
	}
	if value := r.URL.Query().Get("to"); value != "" {
		to, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return from, to, errors.New("to must be RFC3339")
		}
	}
	if !to.After(from) {
		return from, to, errors.New("to must be after from")
	}
	return from, to, nil
}
func decodeJSON(w http.ResponseWriter, r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return err
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if s.Logger != nil {
		s.Logger.Error("request failed", "request_id", middleware.GetReqID(r.Context()), "method", r.Method, "path", r.URL.Path, "error", err)
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}

var _ context.Context
