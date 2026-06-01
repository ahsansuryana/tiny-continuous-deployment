package database

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func New(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA cache_size=-8000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, err
		}
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			name            TEXT NOT NULL UNIQUE,
			repo_url        TEXT NOT NULL DEFAULT '',
			compose_path    TEXT NOT NULL DEFAULT './docker-compose.yml',
			compose_type    TEXT NOT NULL DEFAULT 'path',
			compose_content TEXT NOT NULL DEFAULT '',
			webhook_token   TEXT NOT NULL,
			branch          TEXT NOT NULL DEFAULT 'main',
			registry_type   TEXT NOT NULL DEFAULT '',
			registry_user   TEXT NOT NULL DEFAULT '',
			registry_pass   TEXT NOT NULL DEFAULT '',
			auto_deploy     INTEGER NOT NULL DEFAULT 1,
			created_at      TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS deployments (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id      INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			status          TEXT NOT NULL DEFAULT 'pending',
			trigger         TEXT NOT NULL DEFAULT 'manual',
			started_at      TEXT NOT NULL DEFAULT (datetime('now')),
			finished_at     TEXT,
			log             TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_project ON deployments(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token  TEXT PRIMARY KEY,
			data   BLOB NOT NULL,
			expiry REAL NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expiry)`,
	}

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return db.migrateV2()
}

func (db *DB) migrateV2() error {
	columns := []struct {
		name string
		typ  string
		def  string
	}{
		{"compose_type", "TEXT", "'path'"},
		{"compose_content", "TEXT", "''"},
		{"traefik_hostname", "TEXT", "''"},
		{"traefik_port", "INTEGER", "0"},
		{"traefik_middleware", "TEXT", "''"},
	}

	for _, col := range columns {
		exists, err := db.columnExists("projects", col.name)
		if err != nil {
			return err
		}
		if !exists {
			_, err := db.Exec(fmt.Sprintf(
				"ALTER TABLE projects ADD COLUMN %s %s NOT NULL DEFAULT %s",
				col.name, col.typ, col.def))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *DB) columnExists(table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, nil
}
