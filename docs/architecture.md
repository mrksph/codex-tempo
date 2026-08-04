# Architecture

Codex Tempo separates collection, calculation, and presentation:

- `codex-tempo-agent` incrementally reads Codex JSONL transcripts and first commits canonical metadata-only events to SQLite.
- `codex-tempo-server` accepts idempotent event batches, stores immutable events in PostgreSQL, and deterministically rebuilds runs.
- `codex-tempo` reports from the local SQLite database while offline.
- Next.js calls the Go API through a server-side client and authenticated BFF route.
- Project identity is resolved from the normalized Git remote when one exists, with linked worktrees grouped under the same project and the worktree metadata kept for later breakdowns.
- Hook diagnostics stay local, rotate daily, and are retained for a bounded period instead of being treated as application data.

Intervals use `[start, end)`. Events have per-session order; there is no global ordering assumption. Original events are never mutated by projection.
