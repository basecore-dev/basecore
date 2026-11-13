// Package store manages the application's database connection and schema.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// DB is the active database connection, or nil when no connection is open.
var DB *sql.DB

// IsUniqueError reports whether err is a SQLite unique constraint violation.
func IsUniqueError(err error) bool {
	var sqliteErr *sqlite.Error

	if !errors.As(err, &sqliteErr) {
		return false
	}

	return sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

// ConnectDB opens a SQLite database at dbPath and configures its connection.
func ConnectDB(dbPath string) error {
	if !strings.HasPrefix(dbPath, ":memory:") && !strings.HasPrefix(dbPath, "file:") {
		file, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0600)

		if err != nil {
			return fmt.Errorf("create database file: %w", err)
		}

		file.Close()

		if err := os.Chmod(dbPath, 0600); err != nil {
			return fmt.Errorf("set database permissions: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)

	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode and foreign keys for better concurrency and integrity.
	// secure_delete overwrites deleted content so stale ciphertext does not
	// linger in free pages.
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA secure_delete = ON;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return fmt.Errorf("exec pragma %q: %w", pragma, err)
		}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("ping database: %w", err)
	}

	DB = db

	return nil
}

// CloseDB closes the active database connection, if one exists.
func CloseDB() error {
	if DB == nil {
		return nil
	}

	if err := DB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	DB = nil

	return nil
}

// CreateTables creates the application tables if they do not already exist.
func CreateTables() error {
	if DB == nil {
		return errors.New("database not connected")
	}

	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS organizations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	description TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS organizations_members (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role TEXT NOT NULL,
	joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,

	UNIQUE(user_id, organization_id)
);

CREATE TABLE IF NOT EXISTS projects (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

	UNIQUE(title, organization_id)
);

CREATE TABLE IF NOT EXISTS api_keys (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	title TEXT NOT NULL,
	key_hash TEXT NOT NULL,
	permissions TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

	UNIQUE (title, project_id)
)
`

	if _, err := DB.Exec(schema); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	return nil
}
