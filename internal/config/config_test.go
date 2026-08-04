package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadExpandsPathsAndDurations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "config.toml")
	data := []byte("data_dir = \"~/tempo\"\ncodex_home = \"$HOME/codex\"\nscan_interval = \"5s\"\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != filepath.Join(home, "tempo") || cfg.CodexHome != filepath.Join(home, "codex") {
		t.Fatalf("paths not expanded: %#v", cfg)
	}
	if cfg.ScanInterval != 5*time.Second {
		t.Fatalf("scan interval = %s", cfg.ScanInterval)
	}
	if cfg.LogRetention != 14*24*time.Hour {
		t.Fatalf("log retention = %s", cfg.LogRetention)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	want := Default()
	want.ServerURL = "https://tempo.example.com"
	want.MachineToken = "secret"
	want.MachineName = "workstation"
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ServerURL != want.ServerURL || got.MachineToken != want.MachineToken || got.ScanInterval != want.ScanInterval || got.LogRetention != want.LogRetention {
		t.Fatalf("round trip = %#v", got)
	}
}
