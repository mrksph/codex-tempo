package postgres

import (
	"encoding/json"
	"testing"

	"github.com/mrksph/codex-tempo/internal/domain"
)

func TestProjectFromEventsIncludesWorktreeMetadata(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"project_id": "project", "project_name": "tempo", "project_fingerprint": "fingerprint",
		"remote_hash": "remote", "worktree_name": "feature", "worktree_path": "/workspace/feature",
		"is_worktree": true, "codex_version": "1.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := projectFromEvents([]domain.Event{{Payload: payload}})
	if got.ProjectID != "project" || got.ProjectName != "tempo" || got.WorktreeName != "feature" ||
		got.WorktreePath != "/workspace/feature" || !got.IsWorktree || got.CodexVersion != "1.0" {
		t.Fatalf("metadata = %#v", got)
	}
}
