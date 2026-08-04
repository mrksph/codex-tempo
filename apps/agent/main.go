package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mrksph/codex-tempo/internal/agent"
	"github.com/mrksph/codex-tempo/internal/codex"
	"github.com/mrksph/codex-tempo/internal/config"
	"github.com/mrksph/codex-tempo/internal/localdb"
	"github.com/mrksph/codex-tempo/internal/registration"
	ledgersync "github.com/mrksph/codex-tempo/internal/sync"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "configure" {
		configure(os.Args[2:])
		return
	}
	flags := flag.NewFlagSet("codex-tempo-agent", flag.ExitOnError)
	configPath := flags.String("config", "", "configuration file")
	once := flags.Bool("once", false, "scan once and exit")
	_ = flags.Parse(os.Args[1:])
	run(*configPath, *once)
}

func run(configPath string, once bool) {
	cfg, err := config.Load(configPath)
	if err != nil {
		fatal(err)
	}
	store, err := localdb.Open(filepath.Join(cfg.DataDir, "tempo.db"))
	if err != nil {
		fatal(err)
	}
	defer store.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	machineID, err := store.MachineID(ctx)
	if err != nil {
		fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	service := &agent.Service{Store: store, Parser: codex.Parser{MachineID: machineID, StorePaths: cfg.Privacy.StorePaths}, SessionsDir: filepath.Join(cfg.CodexHome, "sessions"), ScanInterval: cfg.ScanInterval, Logger: logger}
	if once {
		if err = service.ScanOnce(ctx); err != nil {
			fatal(err)
		}
		return
	}
	if cfg.ServerURL != "" {
		client := &ledgersync.Client{Store: store, ServerURL: cfg.ServerURL, Token: cfg.MachineToken, MachineID: machineID}
		go func() {
			if err := client.Run(ctx, cfg.SyncInterval); err != nil && ctx.Err() == nil {
				logger.Error("sync stopped", "error", err)
			}
		}()
	}
	if err = service.Run(ctx); err != nil && ctx.Err() == nil {
		fatal(err)
	}
}

func configure(args []string) {
	flags := flag.NewFlagSet("configure", flag.ExitOnError)
	configPath := flags.String("config", "", "configuration file")
	serverURL := flags.String("server-url", "", "Codex Tempo server URL")
	setupKey := flags.String("api-key", os.Getenv("CODEX_TEMPO_SETUP_KEY"), "agent setup key")
	machineName := flags.String("name", "", "machine name")
	_ = flags.Parse(args)
	cfg, err := config.Load(*configPath)
	if err != nil {
		fatal(err)
	}
	if *serverURL == "" {
		*serverURL = cfg.ServerURL
	}
	if *serverURL == "" {
		fmt.Fprint(os.Stderr, "Server URL: ")
		*serverURL = readLine()
	}
	if *setupKey == "" {
		fmt.Fprint(os.Stderr, "Agent setup key: ")
		*setupKey = readLine()
	}
	if *serverURL == "" || *setupKey == "" {
		fatal(fmt.Errorf("server URL and agent setup key are required"))
	}
	if *machineName == "" {
		*machineName, _ = os.Hostname()
	}
	if *machineName == "" {
		*machineName = "codex-machine"
	}
	store, err := localdb.Open(filepath.Join(cfg.DataDir, "tempo.db"))
	if err != nil {
		fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	machineID, err := store.MachineID(ctx)
	if err != nil {
		fatal(err)
	}
	registered, err := (registration.Client{ServerURL: *serverURL, SetupKey: *setupKey}).Register(ctx, machineID, *machineName)
	if err != nil {
		fatal(err)
	}
	cfg.ServerURL = strings.TrimRight(*serverURL, "/")
	cfg.MachineToken = registered.Token
	cfg.MachineName = *machineName
	if err = config.Save(*configPath, cfg); err != nil {
		fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	service := &agent.Service{Store: store, Parser: codex.Parser{MachineID: machineID, StorePaths: cfg.Privacy.StorePaths}, SessionsDir: filepath.Join(cfg.CodexHome, "sessions"), ScanInterval: cfg.ScanInterval, Logger: logger}
	if err = service.ScanOnce(ctx); err != nil {
		fatal(err)
	}
	client := &ledgersync.Client{Store: store, ServerURL: cfg.ServerURL, Token: cfg.MachineToken, MachineID: machineID}
	if err = client.SyncOnce(ctx); err != nil {
		fatal(err)
	}
	fmt.Printf("Machine %s configured. Start or restart codex-tempo-agent.\n", machineID)
}

func readLine() string {
	value, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(value)
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
