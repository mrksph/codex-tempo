# Architecture

Codex Tempo separates collection, calculation, and presentation:

- `codex-tempo-agent` incrementally reads Codex JSONL transcripts and first commits canonical metadata-only events to SQLite.
- `codex-tempo-server` accepts idempotent event batches, stores immutable events in PostgreSQL, and deterministically rebuilds runs.
- `codex-tempo` reports from the local SQLite database while offline.
- Next.js calls the Go API through a server-side client and authenticated BFF route.

Intervals use `[start, end)`. Events have per-session order; there is no global ordering assumption. Original events are never mutated by projection.
