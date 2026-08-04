# Privacy

The default parser persists timestamps, identifiers, project fingerprints, model metadata, token counts, and lifecycle event kinds. It does not persist prompts, responses, reasoning, source code, diffs, tool arguments, tool outputs, environment variables, or Git remotes in clear text.

Project and worktree paths are disabled by default. Worktree names and linked-worktree flags are stored; paths are included only when `privacy.store_paths` is enabled. Remote identifiers are normalized and SHA-256 hashed before storage.

Local hook logs are written with mode `0600` below the configured data directory. They include session, turn, project/worktree identifiers, execution stages, durations, synchronization outcomes, and errors, but not prompt, response, or tool content. Daily files are retained for 14 days by default.
