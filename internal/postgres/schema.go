package postgres

const schema = `
CREATE TABLE IF NOT EXISTS machines (id uuid PRIMARY KEY,name text NOT NULL,token_hash bytea NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),last_seen_at timestamptz);
CREATE TABLE IF NOT EXISTS projects (id uuid PRIMARY KEY,fingerprint text NOT NULL UNIQUE,name text NOT NULL,remote_hash text,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS sessions (id text PRIMARY KEY,machine_id uuid NOT NULL REFERENCES machines(id),project_id uuid NOT NULL REFERENCES projects(id),cwd text,source text NOT NULL,codex_version text,started_at timestamptz NOT NULL,ended_at timestamptz);
CREATE TABLE IF NOT EXISTS events (id uuid PRIMARY KEY,machine_id uuid NOT NULL REFERENCES machines(id),session_id text NOT NULL,run_id uuid,sequence bigint NOT NULL,occurred_at timestamptz NOT NULL,kind text NOT NULL,source text NOT NULL,payload jsonb NOT NULL DEFAULT '{}',received_at timestamptz NOT NULL DEFAULT now(),UNIQUE(machine_id,session_id,sequence));
CREATE INDEX IF NOT EXISTS events_session_order_idx ON events(session_id,occurred_at,sequence);
CREATE INDEX IF NOT EXISTS events_time_idx ON events(occurred_at);
CREATE TABLE IF NOT EXISTS runs (id uuid PRIMARY KEY,session_id text NOT NULL REFERENCES sessions(id),project_id uuid NOT NULL REFERENCES projects(id),started_at timestamptz NOT NULL,ended_at timestamptz,last_activity_at timestamptz NOT NULL,status text NOT NULL,model text,reasoning_effort text,input_tokens bigint NOT NULL DEFAULT 0,cached_input_tokens bigint NOT NULL DEFAULT 0,output_tokens bigint NOT NULL DEFAULT 0,reasoning_tokens bigint NOT NULL DEFAULT 0,close_reason text,projection_version integer NOT NULL DEFAULT 1,updated_at timestamptz NOT NULL DEFAULT now(),CHECK(ended_at IS NULL OR ended_at>=started_at));
CREATE INDEX IF NOT EXISTS runs_project_time_idx ON runs(project_id,started_at,ended_at);
CREATE UNIQUE INDEX IF NOT EXISTS runs_active_idx ON runs(session_id) WHERE ended_at IS NULL;
CREATE TABLE IF NOT EXISTS project_aliases(project_id uuid NOT NULL REFERENCES projects(id),alias text NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(project_id,alias));
CREATE TABLE IF NOT EXISTS projection_checkpoints(projector text PRIMARY KEY,last_event_received_at timestamptz,projection_version integer NOT NULL);
`
