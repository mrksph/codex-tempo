package projectresolver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNormalizeRemote(t *testing.T) {
	cases := map[string]string{
		"git@github.com:OpenAI/codex.git":     "github.com/openai/codex",
		"https://github.com/OpenAI/codex.git": "github.com/openai/codex",
	}
	for input, want := range cases {
		if got := normalizeRemote(input); got != want {
			t.Errorf("%q => %q, want %q", input, got, want)
		}
	}
}

func TestResolveGroupsLinkedWorktreesByRemote(t *testing.T) {
	main, linked := createRepositoryWithWorktree(t, true)
	mainResult := Resolve(context.Background(), main, "machine", true)
	linkedResult := Resolve(context.Background(), linked, "machine", true)

	if mainResult.ID != linkedResult.ID || mainResult.Fingerprint != linkedResult.Fingerprint {
		t.Fatalf("project identity differs: main=%#v linked=%#v", mainResult, linkedResult)
	}
	if mainResult.Name != "tempo" || linkedResult.Name != "tempo" {
		t.Fatalf("project names = %q, %q", mainResult.Name, linkedResult.Name)
	}
	if mainResult.IsWorktree || !linkedResult.IsWorktree {
		t.Fatalf("worktree flags: main=%t linked=%t", mainResult.IsWorktree, linkedResult.IsWorktree)
	}
	if mainResult.WorktreeName == linkedResult.WorktreeName || linkedResult.WorktreePath != linked {
		t.Fatalf("worktree metadata: main=%#v linked=%#v", mainResult, linkedResult)
	}
}

func TestResolveWithoutRemoteUsesEachWorktreePath(t *testing.T) {
	main, linked := createRepositoryWithWorktree(t, false)
	mainResult := Resolve(context.Background(), main, "machine", false)
	linkedResult := Resolve(context.Background(), linked, "machine", false)

	if mainResult.ID == linkedResult.ID {
		t.Fatalf("worktrees without a remote shared project id: %s", mainResult.ID)
	}
	if mainResult.WorktreePath != "" || linkedResult.WorktreePath != "" {
		t.Fatalf("paths were retained with store_paths disabled: main=%q linked=%q", mainResult.WorktreePath, linkedResult.WorktreePath)
	}
}

func createRepositoryWithWorktree(t *testing.T, withRemote bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	main := filepath.Join(root, "main-checkout")
	linked := filepath.Join(root, "feature-checkout")
	runGit(t, root, "init", main)
	if err := os.WriteFile(filepath.Join(main, "README.md"), []byte("tempo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, main, "add", "README.md")
	runGit(t, main, "-c", "user.name=Tempo Tests", "-c", "user.email=tempo@example.com", "commit", "-m", "initial")
	if withRemote {
		runGit(t, main, "remote", "add", "origin", "git@github.com:mrksph/tempo.git")
	}
	runGit(t, main, "worktree", "add", "-b", "feature", linked)
	return main, linked
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
