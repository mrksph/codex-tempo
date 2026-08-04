# Privacy

The default parser persists timestamps, identifiers, project fingerprints, model metadata, token counts, and lifecycle event kinds. It does not persist prompts, responses, reasoning, source code, diffs, tool arguments, tool outputs, environment variables, or Git remotes in clear text.

Project paths are disabled by default. Remote identifiers are normalized and SHA-256 hashed before storage.
