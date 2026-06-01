CREATE TABLE IF NOT EXISTS projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL UNIQUE,
    repo_url        TEXT NOT NULL DEFAULT '',
    compose_path    TEXT NOT NULL DEFAULT './docker-compose.yml',
    webhook_token   TEXT NOT NULL,
    branch          TEXT NOT NULL DEFAULT 'main',
    registry_type   TEXT NOT NULL DEFAULT '',
    registry_user   TEXT NOT NULL DEFAULT '',
    registry_pass   TEXT NOT NULL DEFAULT '',
    auto_deploy     INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS deployments (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id      INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending',
    trigger         TEXT NOT NULL DEFAULT 'manual',
    started_at      TEXT NOT NULL DEFAULT (datetime('now')),
    finished_at     TEXT,
    log             TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_deployments_project ON deployments(project_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    token  TEXT PRIMARY KEY,
    data   BLOB NOT NULL,
    expiry REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expiry);
