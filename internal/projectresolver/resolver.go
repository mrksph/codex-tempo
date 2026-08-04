package projectresolver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mrksph/codex-tempo/internal/domain"
)

type Result struct {
	ID, Fingerprint, Name, RootPath, RemoteHash string
	WorktreeName, WorktreePath                  string
	IsWorktree                                  bool
}

func Resolve(ctx context.Context, cwd, machineID string, storePaths bool) Result {
	real, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		real = filepath.Clean(cwd)
	}
	root := git(ctx, real, "rev-parse", "--show-toplevel")
	if root == "" {
		root = real
	}
	remote := normalizeRemote(git(ctx, root, "config", "--get", "remote.origin.url"))
	key := remote
	if key == "" {
		key = machineID + "\x00" + root
	}
	fingerprint := hash(key)
	projectName := filepath.Base(root)
	if remote != "" {
		projectName = path.Base(remote)
	}
	result := Result{
		ID: domain.DeterministicUUID("project", fingerprint), Fingerprint: fingerprint,
		Name: projectName, RemoteHash: hash(remote), WorktreeName: filepath.Base(root),
		IsWorktree: isLinkedWorktree(ctx, root),
	}
	if storePaths {
		result.RootPath, result.WorktreePath = root, root
	}
	return result
}

func isLinkedWorktree(ctx context.Context, root string) bool {
	gitDir := absoluteGitPath(root, git(ctx, root, "rev-parse", "--git-dir"))
	commonDir := absoluteGitPath(root, git(ctx, root, "rev-parse", "--git-common-dir"))
	return gitDir != "" && commonDir != "" && gitDir != commonDir
}

func absoluteGitPath(root, value string) string {
	if value == "" {
		return ""
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(root, value)
	}
	return filepath.Clean(value)
}

func git(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func normalizeRemote(remote string) string {
	remote = strings.TrimSpace(strings.TrimSuffix(remote, ".git"))
	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remote, "git@"), ":", 2)
		if len(parts) == 2 {
			return strings.ToLower(parts[0] + "/" + strings.Trim(parts[1], "/"))
		}
	}
	if parsed, err := url.Parse(remote); err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Host + "/" + strings.Trim(parsed.Path, "/"))
	}
	return strings.ToLower(remote)
}

func hash(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
