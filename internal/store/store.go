// Package store is the persistence layer for localchat's Conversations API:
// conversations, their messages, the delivery channels bound to them, and
// the API keys that authenticate external callers. Backed by SQLite via a
// pure-Go driver (no cgo) so the existing "fully local, no external
// dependencies" build stays true — a single file on disk, no database
// server to run.
package store

import (
    "database/sql"
    "fmt"

    _ "modernc.org/sqlite"
)

type Store struct {
    db *sql.DB
}

// Open creates (if needed) and connects to the SQLite database at path.
// WAL mode + a single-connection pool avoids SQLite's classic "database is
// locked" errors under concurrent access without needing a real connection
// pool — this process is the only writer.
func Open(path string) (*Store, error) {
    db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
    if err != nil {
        return nil, fmt.Errorf("open sqlite: %w", err)
    }
    db.SetMaxOpenConns(1)

    s := &Store{db: db}
    if err := s.migrate(); err != nil {
        db.Close()
        return nil, fmt.Errorf("migrate: %w", err)
    }
    return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
    _, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS conversations (
            id            TEXT PRIMARY KEY,
            title         TEXT NOT NULL DEFAULT '',
            metadata_json TEXT NOT NULL DEFAULT '{}',
            created_at    TEXT NOT NULL,
            updated_at    TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS messages (
            id              TEXT PRIMARY KEY,
            conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
            role            TEXT NOT NULL,
            author          TEXT NOT NULL DEFAULT '',
            content         TEXT NOT NULL,
            created_at      TEXT NOT NULL
        );
        CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at);

        -- conversation_id is nullable: a NULL-scoped channel is "global"
        -- and receives every message on every conversation. A non-null
        -- channel only fires for that one conversation.
        CREATE TABLE IF NOT EXISTS channels (
            id              TEXT PRIMARY KEY,
            conversation_id TEXT REFERENCES conversations(id) ON DELETE CASCADE,
            type            TEXT NOT NULL,
            config_json     TEXT NOT NULL DEFAULT '{}',
            created_at      TEXT NOT NULL
        );
        CREATE INDEX IF NOT EXISTS idx_channels_conversation ON channels(conversation_id);

        CREATE TABLE IF NOT EXISTS api_keys (
            id           TEXT PRIMARY KEY,
            name         TEXT NOT NULL,
            key_hash     TEXT NOT NULL UNIQUE,
            created_at   TEXT NOT NULL,
            last_used_at TEXT
        );
    `)
    return err
}
