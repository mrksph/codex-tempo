-- +goose Up
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS worktree_name text;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS worktree_path text;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS is_worktree boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE sessions DROP COLUMN IF EXISTS is_worktree;
ALTER TABLE sessions DROP COLUMN IF EXISTS worktree_path;
ALTER TABLE sessions DROP COLUMN IF EXISTS worktree_name;
