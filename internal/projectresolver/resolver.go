package projectresolver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mrksph/codex-tempo/internal/domain"
)

type Result struct{ ID, Fingerprint, Name, RootPath, RemoteHash string }

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
		key = machineID + "\x00" + real
	}
	fingerprint := hash(key)
	result := Result{ID: domain.DeterministicUUID("project", fingerprint), Fingerprint: fingerprint, Name: filepath.Base(root), RemoteHash: hash(remote)}
	if storePaths {
		result.RootPath = root
	}
	return result
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
