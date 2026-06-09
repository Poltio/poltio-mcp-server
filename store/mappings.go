package store

import (
	"fmt"
)

// GetGrantByAccessToken looks up a grant by access_token_hash.
func (s *Store) GetGrantByAccessToken(tokenHash string) (*OAuthGrant, error) {
	row := s.readDB.QueryRow(
		`SELECT grant_id, access_token_hash, refresh_token_hash, poltio_token_enc,
		        poltio_org_id, poltio_account_id, org_override, grant_state,
		        created_at, last_used_at
		   FROM oauth_grants
		  WHERE access_token_hash = ?`,
		tokenHash,
	)
	g, err := scanGrant(row)
	if err != nil {
		return nil, fmt.Errorf("store: get grant by access token: %w", err)
	}
	return g, nil
}

// GetGrantByRefreshToken looks up a grant by refresh_token_hash.
func (s *Store) GetGrantByRefreshToken(tokenHash string) (*OAuthGrant, error) {
	row := s.readDB.QueryRow(
		`SELECT grant_id, access_token_hash, refresh_token_hash, poltio_token_enc,
		        poltio_org_id, poltio_account_id, org_override, grant_state,
		        created_at, last_used_at
		   FROM oauth_grants
		  WHERE refresh_token_hash = ?`,
		tokenHash,
	)
	g, err := scanGrant(row)
	if err != nil {
		return nil, fmt.Errorf("store: get grant by refresh token: %w", err)
	}
	return g, nil
}
