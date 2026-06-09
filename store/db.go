package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store wraps two database connection pools.
// writeDB: max 1 open connection — SQLite serializes writes.
// readDB: max 8 open connections — for concurrent reads.
type Store struct {
	writeDB *sql.DB
	readDB  *sql.DB
}

// schema is the set of idempotent DDL statements run at Open time.
const schema = `
CREATE TABLE IF NOT EXISTS oauth_clients (
    client_id          TEXT NOT NULL PRIMARY KEY,
    redirect_uris_json TEXT NOT NULL,
    created_at         DATETIME NOT NULL,
    expires_at         DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_codes (
    code_hash      TEXT NOT NULL PRIMARY KEY,
    client_id      TEXT NOT NULL,
    pkce_challenge TEXT NOT NULL,
    state          TEXT NOT NULL,
    redirect_uri   TEXT NOT NULL,
    grant_id       TEXT NOT NULL,
    created_at     DATETIME NOT NULL,
    expires_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS oauth_grants (
    grant_id            TEXT NOT NULL PRIMARY KEY,
    access_token_hash   TEXT,
    refresh_token_hash  TEXT,
    poltio_token_enc    BLOB,
    poltio_org_id       TEXT NOT NULL,
    poltio_account_id   TEXT NOT NULL,
    org_override        TEXT,
    grant_state         TEXT NOT NULL DEFAULT 'pending'
                        CHECK (grant_state IN ('pending','active','needs_reconnect','revoked')),
    created_at          DATETIME NOT NULL,
    last_used_at        DATETIME
);

CREATE TABLE IF NOT EXISTS pending_consent_sessions (
    session_id     TEXT NOT NULL PRIMARY KEY,
    client_id      TEXT NOT NULL,
    redirect_uri   TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    state          TEXT NOT NULL,
    created_at     DATETIME NOT NULL,
    expires_at     DATETIME NOT NULL
);
`

// Open opens the SQLite database at path and runs idempotent schema migrations.
// It returns a *Store containing separate write and read connection pools.
func Open(path string) (*Store, error) {
	// WAL mode and busy timeout via DSN query params (modernc.org/sqlite parenthesised form).
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)

	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open write db: %w", err)
	}
	writeDB.SetMaxOpenConns(1)

	if err := writeDB.Ping(); err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("store: ping write db: %w", err)
	}

	// Verify WAL mode was actually applied.
	var journalMode string
	if err := writeDB.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("store: check journal_mode: %w", err)
	}
	if journalMode != "wal" {
		writeDB.Close()
		return nil, fmt.Errorf("store: expected journal_mode=wal, got %q — check DSN pragma syntax", journalMode)
	}

	// Run migrations via the write connection.
	if _, err := writeDB.Exec(schema); err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("store: run schema migrations: %w", err)
	}

	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("store: open read db: %w", err)
	}
	readDB.SetMaxOpenConns(8)

	if err := readDB.Ping(); err != nil {
		writeDB.Close()
		readDB.Close()
		return nil, fmt.Errorf("store: ping read db: %w", err)
	}

	return &Store{
		writeDB: writeDB,
		readDB:  readDB,
	}, nil
}

// Close closes both database connection pools.
func (s *Store) Close() error {
	writeErr := s.writeDB.Close()
	readErr := s.readDB.Close()
	if writeErr != nil {
		return writeErr
	}
	return readErr
}
