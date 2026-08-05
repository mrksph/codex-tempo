package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DataDir             string        `toml:"data_dir"`
	CodexHome           string        `toml:"codex_home"`
	ServerURL           string        `toml:"server_url"`
	MachineToken        string        `toml:"machine_token"`
	ScanInterval        time.Duration `toml:"scan_interval"`
	SyncInterval        time.Duration `toml:"sync_interval"`
	HookSyncInterval    time.Duration `toml:"hook_sync_interval"`
	HookActivityTimeout time.Duration `toml:"hook_activity_timeout"`
	LogRetention        time.Duration `toml:"log_retention"`
	MachineName         string        `toml:"machine_name"`
	Privacy             Privacy       `toml:"privacy"`
}

type Privacy struct {
	StorePaths          bool `toml:"store_paths"`
	StoreToolNames      bool `toml:"store_tool_names"`
	StorePromptMetadata bool `toml:"store_prompt_metadata"`
	StoreContent        bool `toml:"store_content"`
}

type diskConfig struct {
	DataDir             string  `toml:"data_dir"`
	CodexHome           string  `toml:"codex_home"`
	ServerURL           string  `toml:"server_url"`
	MachineToken        string  `toml:"machine_token"`
	MachineName         string  `toml:"machine_name"`
	ScanInterval        string  `toml:"scan_interval"`
	SyncInterval        string  `toml:"sync_interval"`
	HookSyncInterval    string  `toml:"hook_sync_interval"`
	HookActivityTimeout string  `toml:"hook_activity_timeout"`
	LogRetention        string  `toml:"log_retention"`
	Privacy             Privacy `toml:"privacy"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DataDir:             filepath.Join(home, ".local", "share", "codex-tempo"),
		CodexHome:           filepath.Join(home, ".codex"),
		ScanInterval:        15 * time.Second,
		SyncInterval:        30 * time.Second,
		HookSyncInterval:    30 * time.Second,
		HookActivityTimeout: 90 * time.Second,
		LogRetention:        14 * 24 * time.Hour,
		Privacy:             Privacy{StoreToolNames: true},
	}
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "codex-tempo", "config.toml")
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = DefaultPath()
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, err
	}
	cfg.DataDir = expandHome(cfg.DataDir)
	cfg.CodexHome = expandHome(cfg.CodexHome)
	applyEnvOverrides(&cfg)
	if cfg.HookSyncInterval <= 0 {
		cfg.HookSyncInterval = 30 * time.Second
	}
	if cfg.HookActivityTimeout <= 0 {
		cfg.HookActivityTimeout = 90 * time.Second
	}
	if cfg.LogRetention <= 0 {
		cfg.LogRetention = 14 * 24 * time.Hour
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if value := strings.TrimSpace(os.Getenv("CODEX_TEMPO_SERVER_URL")); value != "" {
		cfg.ServerURL = strings.TrimRight(value, "/")
	}
	if value := strings.TrimSpace(os.Getenv("CODEX_TEMPO_MACHINE_TOKEN")); value != "" {
		cfg.MachineToken = value
	}
	if value := strings.TrimSpace(os.Getenv("CODEX_TEMPO_MACHINE_NAME")); value != "" {
		cfg.MachineName = value
	}
}

func Save(path string, cfg Config) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".config-*.toml")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err = temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	disk := diskConfig{
		DataDir: cfg.DataDir, CodexHome: cfg.CodexHome, ServerURL: cfg.ServerURL,
		MachineToken: cfg.MachineToken, MachineName: cfg.MachineName,
		ScanInterval: cfg.ScanInterval.String(), SyncInterval: cfg.SyncInterval.String(),
		HookSyncInterval: cfg.HookSyncInterval.String(), HookActivityTimeout: cfg.HookActivityTimeout.String(),
		LogRetention: cfg.LogRetention.String(),
		Privacy:      cfg.Privacy,
	}
	if err = toml.NewEncoder(temporary).Encode(disk); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

func expandHome(path string) string {
	if path == "~" || len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path)
}
