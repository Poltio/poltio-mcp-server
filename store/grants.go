package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// GrantState enumerates valid lifecycle states for an OAuthGrant.
type GrantState string

const (
	GrantStatePending        GrantState = "pending"
	GrantStateActive         GrantState = "active"
	GrantStateNeedsReconnect GrantState = "needs_reconnect"
	GrantStateRevoked        GrantState = "revoked"
)

// OAuthGrant holds the OAuth grant record.
type OAuthGrant struct {
	GrantID          string
	AccessTokenHash  string
	RefreshTokenHash string
	PoltioTokenEnc   []byte     // nil when NULL in DB
	PoltioOrgID      string
	PoltioAccountID  string
	OrgOverride      string     // empty string means "not set"
	GrantState       GrantState
	CreatedAt        time.Time
	LastUsedAt       *time.Time
}

// AuthCode is a short-lived auth code used in the authorization code flow.
type AuthCode struct {
	CodeHash      string
	ClientID      string
	PKCEChallenge string
	State         string
	RedirectURI   string
	GrantID       string
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// PendingSession holds state for a consent session that hasn't completed yet.
type PendingSession struct {
	SessionID     string
	ClientID      string
	RedirectURI   string
	CodeChallenge string
	State         string
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// ErrAuthCodeNotFound is returned when ConsumeAuthCode finds no matching (or expired) code.
var ErrAuthCodeNotFound = errors.New("store: auth code not found or expired")

// CreateGrant inserts a new OAuthGrant.
func (s *Store) CreateGrant(g *OAuthGrant) error {
	var lastUsedAt *string
	if g.LastUsedAt != nil {
		v := g.LastUsedAt.UTC().Format(time.RFC3339)
		lastUsedAt = &v
	}

	var orgOverride *string
	if g.OrgOverride != "" {
		orgOverride = &g.OrgOverride
	}

	var accessTokenHash *string
	if g.AccessTokenHash != "" {
		accessTokenHash = &g.AccessTokenHash
	}

	var refreshTokenHash *string
	if g.RefreshTokenHash != "" {
		refreshTokenHash = &g.RefreshTokenHash
	}

	_, err := s.writeDB.Exec(
		`INSERT INTO oauth_grants
		   (grant_id, access_token_hash, refresh_token_hash, poltio_token_enc,
		    poltio_org_id, poltio_account_id, org_override, grant_state,
		    created_at, last_used_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		g.GrantID,
		accessTokenHash,
		refreshTokenHash,
		g.PoltioTokenEnc,
		g.PoltioOrgID,
		g.PoltioAccountID,
		orgOverride,
		string(g.GrantState),
		g.CreatedAt.UTC().Format(time.RFC3339),
		lastUsedAt,
	)
	if err != nil {
		return fmt.Errorf("store: create grant: %w", err)
	}
	return nil
}

// scanGrant scans a row into an OAuthGrant.
func scanGrant(row *sql.Row) (*OAuthGrant, error) {
	var (
		grantID          string
		accessTokenHash  sql.NullString
		refreshTokenHash sql.NullString
		poltioTokenEnc   []byte
		poltioOrgID      string
		poltioAccountID  string
		orgOverride      sql.NullString
		grantState       string
		createdAt        string
		lastUsedAt       sql.NullString
	)

	if err := row.Scan(
		&grantID,
		&accessTokenHash,
		&refreshTokenHash,
		&poltioTokenEnc,
		&poltioOrgID,
		&poltioAccountID,
		&orgOverride,
		&grantState,
		&createdAt,
		&lastUsedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan grant: %w", err)
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	var lastUsed *time.Time
	if lastUsedAt.Valid {
		t, err := time.Parse(time.RFC3339, lastUsedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse last_used_at: %w", err)
		}
		lu := t.UTC()
		lastUsed = &lu
	}

	return &OAuthGrant{
		GrantID:          grantID,
		AccessTokenHash:  accessTokenHash.String,
		RefreshTokenHash: refreshTokenHash.String,
		PoltioTokenEnc:   poltioTokenEnc,
		PoltioOrgID:      poltioOrgID,
		PoltioAccountID:  poltioAccountID,
		OrgOverride:      orgOverride.String,
		GrantState:       GrantState(grantState),
		CreatedAt:        created.UTC(),
		LastUsedAt:       lastUsed,
	}, nil
}

// GetGrant returns the OAuthGrant with the given grantID, or nil, nil if not found.
func (s *Store) GetGrant(grantID string) (*OAuthGrant, error) {
	row := s.readDB.QueryRow(
		`SELECT grant_id, access_token_hash, refresh_token_hash, poltio_token_enc,
		        poltio_org_id, poltio_account_id, org_override, grant_state,
		        created_at, last_used_at
		   FROM oauth_grants
		  WHERE grant_id = ?`,
		grantID,
	)
	g, err := scanGrant(row)
	if err != nil {
		return nil, fmt.Errorf("store: get grant: %w", err)
	}
	return g, nil
}

// ActivateGrant transitions a grant from pending → active and sets token hashes.
func (s *Store) ActivateGrant(grantID, accessTokenHash, refreshTokenHash string) error {
	_, err := s.writeDB.Exec(
		`UPDATE oauth_grants
		    SET grant_state = 'active',
		        access_token_hash = ?,
		        refresh_token_hash = ?
		  WHERE grant_id = ? AND grant_state = 'pending'`,
		accessTokenHash,
		refreshTokenHash,
		grantID,
	)
	if err != nil {
		return fmt.Errorf("store: activate grant: %w", err)
	}
	return nil
}

// RotateTokens replaces access and refresh token hashes atomically, but only
// if the grant is still active AND the current refresh_token_hash matches oldRefreshHash.
// This prevents double-spend: concurrent callers with the same token see exactly one
// rows-affected=1; the loser sees 0 and should respond with invalid_grant.
// Returns (false, nil) if 0 rows were updated (race lost or stale token).
func (s *Store) RotateTokens(grantID, oldRefreshHash, newAccessHash, newRefreshHash string) (bool, error) {
	res, err := s.writeDB.Exec(
		`UPDATE oauth_grants
		    SET access_token_hash = ?,
		        refresh_token_hash = ?
		  WHERE grant_id = ? AND refresh_token_hash = ? AND grant_state = 'active'`,
		newAccessHash,
		newRefreshHash,
		grantID,
		oldRefreshHash,
	)
	if err != nil {
		return false, fmt.Errorf("store: rotate tokens: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: rotate tokens: rows affected: %w", err)
	}
	return n == 1, nil
}

// MarkNeedsReconnect transitions active → needs_reconnect and NULLs poltio_token_enc.
// Idempotent: if already needs_reconnect or revoked, no rows are updated — no error.
func (s *Store) MarkNeedsReconnect(grantID string) error {
	_, err := s.writeDB.Exec(
		`UPDATE oauth_grants
		    SET grant_state = 'needs_reconnect',
		        poltio_token_enc = NULL
		  WHERE grant_id = ? AND grant_state = 'active'`,
		grantID,
	)
	if err != nil {
		return fmt.Errorf("store: mark needs reconnect: %w", err)
	}
	return nil
}

// RevokeGrant sets grant_state = revoked and NULLs poltio_token_enc.
func (s *Store) RevokeGrant(grantID string) error {
	_, err := s.writeDB.Exec(
		`UPDATE oauth_grants
		    SET grant_state = 'revoked',
		        poltio_token_enc = NULL
		  WHERE grant_id = ?`,
		grantID,
	)
	if err != nil {
		return fmt.Errorf("store: revoke grant: %w", err)
	}
	return nil
}

// SetOrgOverride writes a new org_id to org_override for session-scoped org switching.
func (s *Store) SetOrgOverride(grantID, orgID string) error {
	_, err := s.writeDB.Exec(
		`UPDATE oauth_grants SET org_override = ? WHERE grant_id = ?`,
		orgID,
		grantID,
	)
	if err != nil {
		return fmt.Errorf("store: set org override: %w", err)
	}
	return nil
}

// SweepGrants deletes expired pending grants, expired needs_reconnect grants, and expired clients.
// consentTTL controls how long pending grants survive before sweep.
// reconnectRetention controls how long needs_reconnect grants survive.
// Returns the total number of rows deleted.
func (s *Store) SweepGrants(consentTTL, reconnectRetention time.Duration) (deleted int64, err error) {
	now := time.Now().UTC()
	pendingCutoff := now.Add(-consentTTL).Format(time.RFC3339)
	reconnectCutoff := now.Add(-reconnectRetention).Format(time.RFC3339)

	res1, err := s.writeDB.Exec(
		`DELETE FROM oauth_grants
		  WHERE grant_state = 'pending' AND created_at < ?`,
		pendingCutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("store: sweep pending grants: %w", err)
	}
	n1, _ := res1.RowsAffected()

	res2, err := s.writeDB.Exec(
		`DELETE FROM oauth_grants
		  WHERE grant_state = 'needs_reconnect' AND created_at < ?`,
		reconnectCutoff,
	)
	if err != nil {
		return n1, fmt.Errorf("store: sweep needs_reconnect grants: %w", err)
	}
	n2, _ := res2.RowsAffected()

	clientsDeleted, err := s.DeleteExpiredClients()
	if err != nil {
		return n1 + n2, fmt.Errorf("store: sweep expired clients: %w", err)
	}

	return n1 + n2 + clientsDeleted, nil
}

// CreateAuthCode inserts an auth code record.
func (s *Store) CreateAuthCode(code *AuthCode) error {
	_, err := s.writeDB.Exec(
		`INSERT INTO auth_codes
		   (code_hash, client_id, pkce_challenge, state, redirect_uri, grant_id, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		code.CodeHash,
		code.ClientID,
		code.PKCEChallenge,
		code.State,
		code.RedirectURI,
		code.GrantID,
		code.CreatedAt.UTC().Format(time.RFC3339),
		code.ExpiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("store: create auth code: %w", err)
	}
	return nil
}

// ConsumeAuthCode atomically SELECTs, verifies not expired, and DELETEs the auth code.
// Returns ErrAuthCodeNotFound if absent or expired.
// Because writeDB has MaxOpenConns=1, the transaction serializes concurrent callers.
func (s *Store) ConsumeAuthCode(codeHash string) (*AuthCode, error) {
	ctx := context.Background()
	tx, err := s.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: consume auth code: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var (
		cHash         string
		clientID      string
		pkceChallenge string
		state         string
		redirectURI   string
		grantID       string
		createdAt     string
		expiresAt     string
	)

	err = tx.QueryRowContext(ctx,
		`SELECT code_hash, client_id, pkce_challenge, state, redirect_uri, grant_id, created_at, expires_at
		   FROM auth_codes
		  WHERE code_hash = ?`,
		codeHash,
	).Scan(&cHash, &clientID, &pkceChallenge, &state, &redirectURI, &grantID, &createdAt, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAuthCodeNotFound
		}
		return nil, fmt.Errorf("store: consume auth code: select: %w", err)
	}

	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("store: consume auth code: parse expires_at: %w", err)
	}
	if time.Now().UTC().After(expires.UTC()) {
		// Delete expired code and return ErrAuthCodeNotFound.
		_, _ = tx.ExecContext(ctx, `DELETE FROM auth_codes WHERE code_hash = ?`, codeHash)
		_ = tx.Commit()
		return nil, ErrAuthCodeNotFound
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM auth_codes WHERE code_hash = ?`, codeHash); err != nil {
		return nil, fmt.Errorf("store: consume auth code: delete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: consume auth code: commit: %w", err)
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("store: consume auth code: parse created_at: %w", err)
	}

	return &AuthCode{
		CodeHash:      cHash,
		ClientID:      clientID,
		PKCEChallenge: pkceChallenge,
		State:         state,
		RedirectURI:   redirectURI,
		GrantID:       grantID,
		CreatedAt:     created.UTC(),
		ExpiresAt:     expires.UTC(),
	}, nil
}

// CreatePendingSession inserts a pending consent session.
func (s *Store) CreatePendingSession(sess *PendingSession) error {
	_, err := s.writeDB.Exec(
		`INSERT INTO pending_consent_sessions
		   (session_id, client_id, redirect_uri, code_challenge, state, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.SessionID,
		sess.ClientID,
		sess.RedirectURI,
		sess.CodeChallenge,
		sess.State,
		sess.CreatedAt.UTC().Format(time.RFC3339),
		sess.ExpiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("store: create pending session: %w", err)
	}
	return nil
}

// GetPendingSession returns nil, nil if not found or expired.
func (s *Store) GetPendingSession(sessionID string) (*PendingSession, error) {
	var (
		sessID        string
		clientID      string
		redirectURI   string
		codeChallenge string
		state         string
		createdAt     string
		expiresAt     string
	)

	err := s.readDB.QueryRow(
		`SELECT session_id, client_id, redirect_uri, code_challenge, state, created_at, expires_at
		   FROM pending_consent_sessions
		  WHERE session_id = ?`,
		sessionID,
	).Scan(&sessID, &clientID, &redirectURI, &codeChallenge, &state, &createdAt, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get pending session: scan: %w", err)
	}

	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("store: get pending session: parse expires_at: %w", err)
	}
	if time.Now().UTC().After(expires.UTC()) {
		return nil, nil
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("store: get pending session: parse created_at: %w", err)
	}

	return &PendingSession{
		SessionID:     sessID,
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		CodeChallenge: codeChallenge,
		State:         state,
		CreatedAt:     created.UTC(),
		ExpiresAt:     expires.UTC(),
	}, nil
}

// DeletePendingSession removes the session (used after consent completes or on error).
func (s *Store) DeletePendingSession(sessionID string) error {
	_, err := s.writeDB.Exec(
		`DELETE FROM pending_consent_sessions WHERE session_id = ?`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("store: delete pending session: %w", err)
	}
	return nil
}
