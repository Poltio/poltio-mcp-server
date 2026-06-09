package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// OAuthClient represents a registered OAuth client.
type OAuthClient struct {
	ClientID     string
	RedirectURIs []string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// CreateClient inserts an OAuthClient record.
func (s *Store) CreateClient(c *OAuthClient) error {
	urisJSON, err := json.Marshal(c.RedirectURIs)
	if err != nil {
		return fmt.Errorf("store: create client: marshal redirect_uris: %w", err)
	}
	_, err = s.writeDB.Exec(
		`INSERT INTO oauth_clients (client_id, redirect_uris_json, created_at, expires_at)
		 VALUES (?, ?, ?, ?)`,
		c.ClientID,
		string(urisJSON),
		c.CreatedAt.UTC().Format(time.RFC3339),
		c.ExpiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("store: create client: %w", err)
	}
	return nil
}

// GetClient returns the OAuthClient with the given clientID, or nil, nil if not found.
func (s *Store) GetClient(clientID string) (*OAuthClient, error) {
	row := s.readDB.QueryRow(
		`SELECT client_id, redirect_uris_json, created_at, expires_at
		   FROM oauth_clients
		  WHERE client_id = ?`,
		clientID,
	)

	var (
		id        string
		urisJSON  string
		createdAt string
		expiresAt string
	)
	if err := row.Scan(&id, &urisJSON, &createdAt, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get client: scan: %w", err)
	}

	var uris []string
	if err := json.Unmarshal([]byte(urisJSON), &uris); err != nil {
		return nil, fmt.Errorf("store: get client: unmarshal redirect_uris: %w", err)
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("store: get client: parse created_at: %w", err)
	}
	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("store: get client: parse expires_at: %w", err)
	}

	return &OAuthClient{
		ClientID:     id,
		RedirectURIs: uris,
		CreatedAt:    created.UTC(),
		ExpiresAt:    expires.UTC(),
	}, nil
}

// DeleteExpiredClients removes clients whose expires_at is in the past.
// Returns the number of rows deleted.
func (s *Store) DeleteExpiredClients() (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.writeDB.Exec(
		`DELETE FROM oauth_clients WHERE expires_at < ?`,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("store: delete expired clients: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: delete expired clients: rows affected: %w", err)
	}
	return n, nil
}
