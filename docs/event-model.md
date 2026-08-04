# Event model

The Codex parser currently recognizes these rollout records:

| Codex record | Canonical event |
| --- | --- |
| `session_meta` | `session_started` |
| `event_msg.task_started` | `run_started` |
| `turn_context` | `lease` with model metadata |
| `event_msg.token_count` | `token_usage` |
| `event_msg.task_complete` | `run_completed` |

Native `turn_id` values produce deterministic run UUIDs. Event UUIDs include machine, session, event kind, run, and session sequence. Message, prompt, reasoning, tool output, and file content fields are ignored.
