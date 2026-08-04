# Codex Tempo

Self-hosted time tracking for parallel Codex sessions. It reports agent time, union wall-clock time, project span, token usage, and concurrency without collapsing independent sessions into one heartbeat stream.

## Components

- Go agent with incremental Codex JSONL parsing and SQLite WAL queue.
- Offline Go CLI for reports, timelines, diagnostics, and reparsing.
- Go API with PostgreSQL, idempotent batch ingestion, deterministic projection, and reports.
- Authenticated Next.js dashboard with project charts, interactive timeline, and active-run polling.

## Local agent

Go 1.24 or newer is required.

```bash
go build -o bin/codex-tempo-agent ./apps/agent
go build -o bin/codex-tempo ./apps/cli
bin/codex-tempo-agent --once
bin/codex-tempo report today
bin/codex-tempo doctor
```

Configuration defaults to `~/.config/codex-tempo/config.toml`; see `deploy/examples/config.toml`. The database defaults to `~/.local/share/codex-tempo/tempo.db`.

## Server and dashboard

```bash
cp .env.example .env
docker compose --env-file .env -f deploy/compose/docker-compose.yml up --build
```

Open `http://localhost:3100` and sign in with `WEB_PASSWORD`. Set `WEB_PORT` to publish another host port.

Configure an agent machine with the setup key shown in Dashboard → Ajustes:

```bash
codex-tempo-agent configure \
  --server-url http://localhost:8080 \
  --api-key "$AGENT_SETUP_KEY"
```

The command registers the local machine, stores its dedicated token in `~/.config/codex-tempo/config.toml`, imports existing sessions, and performs the first synchronization. The setup key is not retained.

### Plugin-driven capture

Codex Tempo can run without a permanent background process. Its plugin uses the same three lifecycle events as `codex-cli-wakatime`: `SessionStart`, `UserPromptSubmit`, and `PostToolUse`. Time is recorded only between consecutive heartbeats from the same Codex session when they are within `hook_activity_timeout`; longer idle gaps record no time. Each bounded interval is queued in SQLite. Upload attempts are rate-limited by `hook_sync_interval`, and failed uploads remain queued until a later hook.

Prompt, response, and tool contents are never stored. The transcript is read incrementally only to extract model metadata and token counters, which are recorded in zero-duration `metrics` runs and therefore cannot inflate tracked time. The first plugin heartbeat establishes a source cutover: later Wakapi imports stop at that timestamp so they cannot overlap future hook intervals.

Git checkouts and linked worktrees with the same normalized `remote.origin.url` share one project ID. Sessions retain the worktree name and linked-worktree flag; when `privacy.store_paths` is enabled they also retain the worktree root path. Repositories without an origin use their physical root path as identity, so separate worktrees remain separate projects in that case.

Every hook writes structured diagnostics to `data_dir/logs/hook-YYYY-MM-DD.jsonl`. Files rotate daily and logs older than `log_retention` are removed. Logs contain lifecycle identifiers, project/worktree metadata, stage timings, queue activity, and errors, but never prompt or tool content.

The systemd agent remains available as an optional transcript-import fallback, but it is not required when the plugin is enabled and should not run alongside hook capture for normal usage. A direct hooks file remains available at `deploy/codex/hooks.json` for installations that do not use the plugin.

Relevant configuration:

```toml
hook_sync_interval = "30s"
hook_activity_timeout = "90s"
log_retention = "336h"
```

Import historical coding time from the Wakapi server configured in `~/.wakatime.cfg`:

```bash
codex-tempo import wakapi
```

The importer reads heartbeat timestamps and project names only. It uses Wakapi's 10-minute heartbeat timeout by default, creates deterministic `wakapi` sessions and runs, and can be executed repeatedly without duplicating data. Use `--from`, `--to`, or `--timeout` to override the detected range and timeout.

## Verification

```bash
make test
make web-generate
make web-check
```

The required parallel-session example is covered by unit tests and `testdata/parallel-events.jsonl`.
