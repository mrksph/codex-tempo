# Codex Tempo

[![CI](https://github.com/mrksph/codex-tempo/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/mrksph/codex-tempo/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

Codex Tempo is self-hosted time tracking for parallel Codex sessions. It records agent time, project span, wall-clock time, token usage, and concurrency without flattening independent sessions into a single heartbeat stream.

## Why It Exists

Traditional activity trackers assume one linear stream of work. Codex Tempo is built for the opposite case: multiple agent sessions, multiple worktrees, and overlapping intervals on the same machine or across several machines.

It answers questions like:

- How much agent time did each project consume?
- What was the real wall-clock span after removing overlap?
- Which sessions were active at the same time?
- How much did the agent actually do, versus how long the calendar stayed busy?

## Example

Two sessions can overlap without losing attribution:

```text
Project A: 10:00-10:20
Project B: 10:05-10:25

Agent time      40 min
Wall-clock time  25 min
Peak parallelism 2
```

## What It Includes

- A Go agent with incremental Codex JSONL parsing and a local SQLite queue.
- A Go API with PostgreSQL, idempotent ingestion, and deterministic projection.
- A Next.js dashboard with project charts, timelines, filtering, and live polling.
- Hook diagnostics with daily rotation and 14-day retention.
- Worktree-aware project resolution for repositories with linked worktrees.

## Architecture

The system is split into three layers:

- `codex-tempo-agent` runs locally, captures Codex activity, and syncs batches later.
- `codex-tempo-server` accepts immutable events and rebuilds projections from them.
- `codex-tempo` provides offline reports and diagnostics against the local database.

The dashboard is a BFF-style frontend. Time calculations stay in Go, not in the browser.

## Requirements

For self-hosted deployment:

- Docker Engine with Compose
- A recent browser for the dashboard
- One or more machines with Codex hooks or the local agent installed

For local builds from source:

- Go 1.24 or newer
- Node.js 24 or newer
- pnpm 9 or newer

## Quick Start

1. Copy the example environment file and fill the secrets.

```bash
cp .env.example .env
```

2. Start the full stack.

```bash
docker compose --env-file .env -f deploy/compose/docker-compose.yml up --build
```

3. Open the dashboard.

```text
http://localhost:3100
```

4. Sign in with `WEB_PASSWORD`.

5. In the dashboard settings page, create or copy the agent setup key and configure each machine.

```bash
codex-tempo-agent configure \
  --server-url http://localhost:8080 \
  --api-key "$AGENT_SETUP_KEY"
```

The setup command registers the machine, stores its dedicated token locally, imports existing sessions, and performs the first sync. The setup key itself is not kept.

## Local Agent

Build the local tools from source:

```bash
go build -o bin/codex-tempo-agent ./apps/agent
go build -o bin/codex-tempo ./apps/cli
```

Useful commands:

```bash
bin/codex-tempo-agent --once
bin/codex-tempo report today
bin/codex-tempo doctor
```

The default local config lives at `~/.config/codex-tempo/config.toml`. The default database lives at `~/.local/share/codex-tempo/tempo.db`.

## Configuration

The example configuration is in [`deploy/examples/config.toml`](./deploy/examples/config.toml).

Relevant runtime values include:

- `hook_sync_interval`
- `hook_activity_timeout`
- `log_retention`

The production Compose stack also expects secrets such as:

- `POSTGRES_PASSWORD`
- `AGENT_SETUP_KEY`
- `INTERNAL_API_TOKEN`
- `WEB_PASSWORD`
- `AUTH_SECRET`
- `PUBLIC_API_URL`
- `TZ`
- `WEB_PORT`
- `API_PORT`

## Privacy

- Prompts, responses, reasoning, tool output, and file content are not stored by default.
- Worktree names and linked-worktree metadata are stored.
- Paths are optional and only stored when `privacy.store_paths` is enabled.
- Hook logs stay local and are written with restrictive permissions.

More detail lives in [`docs/privacy.md`](./docs/privacy.md).

## Verification

Current Go statement coverage: `38.3%`.

The CI workflow runs:

- `go vet ./...`
- `go test -race ./...`
- `go build ./apps/agent ./apps/cli ./apps/server`
- `pnpm lint`
- `pnpm typecheck`
- `pnpm build`

The repository also keeps focused unit tests around the parser, resolver, local database, sync, and hook logging paths.

## Documentation

- [`CONTRIBUTING.md`](./CONTRIBUTING.md)
- [`docs/architecture.md`](./docs/architecture.md)
- [`docs/deployment-dokploy.md`](./docs/deployment-dokploy.md)
- [`docs/event-model.md`](./docs/event-model.md)
- [`docs/privacy.md`](./docs/privacy.md)

## Contributing

Contributions are welcome, including bug reports, documentation improvements, feature suggestions, and code changes.

Before working on a significant change, please open an issue to discuss the proposal. That helps avoid duplicated work and keeps the change in scope.

For a code contribution:

1. Fork the repository.
2. Create a branch for your change.
3. Keep the change focused and avoid unrelated refactoring.
4. Format and check the code:

   ```bash
   go fmt ./...
   go vet ./...
   go test ./...
   ```

5. Open a pull request that explains what the change does, why it is needed, and how it was tested.

Small pull requests are easier to review and merge.

## Feedback

Use GitHub Issues for bug reports, documentation gaps, and feature requests. Include the relevant command, environment, logs, and reproduction steps when possible. If the project later enables Discussions, use them for broader design questions.

## Notes

- The public default branch is `master`.
- The project is intended to be self-hosted first.
