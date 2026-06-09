package store

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// -------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// -------------------------------------------------------------------
// 1. Encrypt/Decrypt round-trip
// -------------------------------------------------------------------

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("hello, world!")
	recordID := []byte("record-001")

	ct, err := Encrypt(plaintext, recordID, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := Decrypt(ct, recordID, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, plaintext)
	}
}

// -------------------------------------------------------------------
// 2. Wrong key → error
// -------------------------------------------------------------------

func TestDecryptWrongKey(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("secret value")
	recordID := []byte("rec-1")

	ct, err := Encrypt(plaintext, recordID, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	wrongKey := make([]byte, 32)
	// all zeros — different from testKey
	if _, err := Decrypt(ct, recordID, wrongKey); err == nil {
		t.Fatal("expected error decrypting with wrong key, got nil")
	}
}

// -------------------------------------------------------------------
// 3. Wrong AAD → error
// -------------------------------------------------------------------

func TestDecryptWrongAAD(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("secret value")
	recordID := []byte("rec-correct")

	ct, err := Encrypt(plaintext, recordID, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := Decrypt(ct, []byte("rec-wrong"), key); err == nil {
		t.Fatal("expected error decrypting with wrong recordID, got nil")
	}
}

// -------------------------------------------------------------------
// 4. Empty plaintext
// -------------------------------------------------------------------

func TestEncryptDecryptEmptyPlaintext(t *testing.T) {
	key := testKey(t)
	plaintext := []byte{}
	recordID := []byte("rec-empty")

	ct, err := Encrypt(plaintext, recordID, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := Decrypt(ct, recordID, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("expected empty plaintext, got %v", got)
	}
}

// -------------------------------------------------------------------
// 5. KeyFromEnv absent
// -------------------------------------------------------------------

func TestKeyFromEnvAbsent(t *testing.T) {
	os.Unsetenv("BRIDGE_ENCRYPTION_KEY")
	_, err := KeyFromEnv()
	if err == nil {
		t.Fatal("expected error when BRIDGE_ENCRYPTION_KEY is absent, got nil")
	}
}

// -------------------------------------------------------------------
// 6. KeyFromEnv wrong length
// -------------------------------------------------------------------

func TestKeyFromEnvWrongLength(t *testing.T) {
	// 16 bytes = 32 hex chars — valid hex but wrong key length
	shortKey := make([]byte, 16)
	t.Setenv("BRIDGE_ENCRYPTION_KEY", hex.EncodeToString(shortKey))
	_, err := KeyFromEnv()
	if err == nil {
		t.Fatal("expected error for wrong length key, got nil")
	}
}

// -------------------------------------------------------------------
// 7. Schema idempotency: Open same file twice → no error
// -------------------------------------------------------------------

func TestSchemaIdempotency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "idempotent.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	s2.Close()
}

// -------------------------------------------------------------------
// 8. All 4 tables created
// -------------------------------------------------------------------

func TestAllTablesCreated(t *testing.T) {
	s := openTestStore(t)

	want := map[string]bool{
		"oauth_clients":            false,
		"auth_codes":               false,
		"oauth_grants":             false,
		"pending_consent_sessions": false,
	}

	rows, err := s.readDB.Query(
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`,
	)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		want[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	for table, found := range want {
		if !found {
			t.Errorf("expected table %q to exist, but it was not found", table)
		}
	}
}

// -------------------------------------------------------------------
// 9. CreateClient + GetClient
// -------------------------------------------------------------------

func TestCreateAndGetClient(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	c := &OAuthClient{
		ClientID:     "client-abc",
		RedirectURIs: []string{"https://example.com/callback", "https://other.example.com/cb"},
		CreatedAt:    now,
		ExpiresAt:    now.Add(24 * time.Hour),
	}

	if err := s.CreateClient(c); err != nil {
		t.Fatalf("CreateClient: %v", err)
	}

	got, err := s.GetClient("client-abc")
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	if got == nil {
		t.Fatal("expected client, got nil")
	}
	if got.ClientID != c.ClientID {
		t.Errorf("ClientID: got %q want %q", got.ClientID, c.ClientID)
	}
	if len(got.RedirectURIs) != 2 {
		t.Errorf("RedirectURIs length: got %d want 2", len(got.RedirectURIs))
	}
	if got.RedirectURIs[0] != "https://example.com/callback" {
		t.Errorf("RedirectURIs[0]: got %q", got.RedirectURIs[0])
	}
	if !got.CreatedAt.Equal(c.CreatedAt) {
		t.Errorf("CreatedAt: got %v want %v", got.CreatedAt, c.CreatedAt)
	}
	if !got.ExpiresAt.Equal(c.ExpiresAt) {
		t.Errorf("ExpiresAt: got %v want %v", got.ExpiresAt, c.ExpiresAt)
	}

	// Not found → nil, nil
	missing, err := s.GetClient("nonexistent")
	if err != nil {
		t.Fatalf("GetClient nonexistent: %v", err)
	}
	if missing != nil {
		t.Fatal("expected nil for missing client")
	}
}

// -------------------------------------------------------------------
// 10. CreateGrant + GetGrant
// -------------------------------------------------------------------

func TestCreateAndGetGrant(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)
	lastUsed := now.Add(-time.Minute)
	enc := []byte("encrypted-token-data")

	g := &OAuthGrant{
		GrantID:          "grant-001",
		AccessTokenHash:  "ath-001",
		RefreshTokenHash: "rth-001",
		PoltioTokenEnc:   enc,
		PoltioOrgID:      "org-1",
		PoltioAccountID:  "acc-1",
		OrgOverride:      "override-org",
		GrantState:       GrantStatePending,
		CreatedAt:        now,
		LastUsedAt:       &lastUsed,
	}

	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	got, err := s.GetGrant("grant-001")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got == nil {
		t.Fatal("expected grant, got nil")
	}
	if got.GrantID != g.GrantID {
		t.Errorf("GrantID: got %q want %q", got.GrantID, g.GrantID)
	}
	if got.AccessTokenHash != g.AccessTokenHash {
		t.Errorf("AccessTokenHash: got %q want %q", got.AccessTokenHash, g.AccessTokenHash)
	}
	if got.RefreshTokenHash != g.RefreshTokenHash {
		t.Errorf("RefreshTokenHash: got %q want %q", got.RefreshTokenHash, g.RefreshTokenHash)
	}
	if !bytes.Equal(got.PoltioTokenEnc, enc) {
		t.Errorf("PoltioTokenEnc: got %v want %v", got.PoltioTokenEnc, enc)
	}
	if got.PoltioOrgID != "org-1" {
		t.Errorf("PoltioOrgID: got %q", got.PoltioOrgID)
	}
	if got.OrgOverride != "override-org" {
		t.Errorf("OrgOverride: got %q want %q", got.OrgOverride, "override-org")
	}
	if got.GrantState != GrantStatePending {
		t.Errorf("GrantState: got %q want %q", got.GrantState, GrantStatePending)
	}
	if !got.CreatedAt.Equal(g.CreatedAt) {
		t.Errorf("CreatedAt: got %v want %v", got.CreatedAt, g.CreatedAt)
	}
	if got.LastUsedAt == nil {
		t.Error("LastUsedAt: expected non-nil")
	} else if !got.LastUsedAt.Equal(lastUsed) {
		t.Errorf("LastUsedAt: got %v want %v", *got.LastUsedAt, lastUsed)
	}

	// Not found
	missing, err := s.GetGrant("nonexistent")
	if err != nil {
		t.Fatalf("GetGrant nonexistent: %v", err)
	}
	if missing != nil {
		t.Fatal("expected nil for missing grant")
	}
}

// -------------------------------------------------------------------
// 11. ActivateGrant
// -------------------------------------------------------------------

func TestActivateGrant(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         "grant-activate",
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStatePending,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	if err := s.ActivateGrant("grant-activate", "new-access-hash", "new-refresh-hash"); err != nil {
		t.Fatalf("ActivateGrant: %v", err)
	}

	got, err := s.GetGrant("grant-activate")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got.GrantState != GrantStateActive {
		t.Errorf("expected state=active, got %q", got.GrantState)
	}
	if got.AccessTokenHash != "new-access-hash" {
		t.Errorf("AccessTokenHash: got %q", got.AccessTokenHash)
	}
	if got.RefreshTokenHash != "new-refresh-hash" {
		t.Errorf("RefreshTokenHash: got %q", got.RefreshTokenHash)
	}
}

// -------------------------------------------------------------------
// 12. MarkNeedsReconnect
// -------------------------------------------------------------------

func TestMarkNeedsReconnect(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)
	enc := []byte("some-encrypted-token")

	g := &OAuthGrant{
		GrantID:         "grant-reconnect",
		PoltioTokenEnc:  enc,
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStateActive,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	if err := s.MarkNeedsReconnect("grant-reconnect"); err != nil {
		t.Fatalf("MarkNeedsReconnect: %v", err)
	}

	got, err := s.GetGrant("grant-reconnect")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got.GrantState != GrantStateNeedsReconnect {
		t.Errorf("expected needs_reconnect, got %q", got.GrantState)
	}
	if got.PoltioTokenEnc != nil {
		t.Errorf("PoltioTokenEnc should be nil, got %v", got.PoltioTokenEnc)
	}
}

// -------------------------------------------------------------------
// 13. MarkNeedsReconnect idempotent
// -------------------------------------------------------------------

func TestMarkNeedsReconnectIdempotent(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         "grant-reconnect-idem",
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStateNeedsReconnect,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	// Should not error even though already in needs_reconnect state.
	if err := s.MarkNeedsReconnect("grant-reconnect-idem"); err != nil {
		t.Fatalf("MarkNeedsReconnect idempotent: %v", err)
	}

	got, err := s.GetGrant("grant-reconnect-idem")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got.GrantState != GrantStateNeedsReconnect {
		t.Errorf("expected needs_reconnect, got %q", got.GrantState)
	}
}

// -------------------------------------------------------------------
// 14. MarkNeedsReconnect revoked wins
// -------------------------------------------------------------------

func TestMarkNeedsReconnectRevokedWins(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         "grant-revoked-wins",
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStateRevoked,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	// MarkNeedsReconnect on revoked grant — WHERE clause on active means no rows updated.
	if err := s.MarkNeedsReconnect("grant-revoked-wins"); err != nil {
		t.Fatalf("MarkNeedsReconnect on revoked: %v", err)
	}

	got, err := s.GetGrant("grant-revoked-wins")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got.GrantState != GrantStateRevoked {
		t.Errorf("expected revoked to stay revoked, got %q", got.GrantState)
	}
}

// -------------------------------------------------------------------
// 15. RevokeGrant
// -------------------------------------------------------------------

func TestRevokeGrant(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         "grant-revoke",
		PoltioTokenEnc:  []byte("some-token"),
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStateActive,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	if err := s.RevokeGrant("grant-revoke"); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}

	got, err := s.GetGrant("grant-revoke")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got.GrantState != GrantStateRevoked {
		t.Errorf("expected revoked, got %q", got.GrantState)
	}
	if got.PoltioTokenEnc != nil {
		t.Errorf("PoltioTokenEnc should be nil after revoke, got %v", got.PoltioTokenEnc)
	}
}

// -------------------------------------------------------------------
// 16. SetOrgOverride
// -------------------------------------------------------------------

func TestSetOrgOverride(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         "grant-org-override",
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStateActive,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	if err := s.SetOrgOverride("grant-org-override", "overridden-org-99"); err != nil {
		t.Fatalf("SetOrgOverride: %v", err)
	}

	got, err := s.GetGrant("grant-org-override")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got.OrgOverride != "overridden-org-99" {
		t.Errorf("OrgOverride: got %q want %q", got.OrgOverride, "overridden-org-99")
	}
}

// -------------------------------------------------------------------
// 17. CreateAuthCode + ConsumeAuthCode round-trip; second consume → error
// -------------------------------------------------------------------

func TestCreateAndConsumeAuthCode(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	code := &AuthCode{
		CodeHash:      "codehash-001",
		ClientID:      "client-1",
		PKCEChallenge: "challenge-xyz",
		State:         "state-abc",
		RedirectURI:   "https://example.com/callback",
		GrantID:       "grant-1",
		CreatedAt:     now,
		ExpiresAt:     now.Add(10 * time.Minute),
	}

	if err := s.CreateAuthCode(code); err != nil {
		t.Fatalf("CreateAuthCode: %v", err)
	}

	got, err := s.ConsumeAuthCode("codehash-001")
	if err != nil {
		t.Fatalf("ConsumeAuthCode first: %v", err)
	}
	if got == nil {
		t.Fatal("expected auth code, got nil")
	}
	if got.CodeHash != code.CodeHash {
		t.Errorf("CodeHash: got %q want %q", got.CodeHash, code.CodeHash)
	}
	if got.PKCEChallenge != "challenge-xyz" {
		t.Errorf("PKCEChallenge: got %q", got.PKCEChallenge)
	}

	// Second consume → ErrAuthCodeNotFound
	_, err = s.ConsumeAuthCode("codehash-001")
	if err != ErrAuthCodeNotFound {
		t.Errorf("second ConsumeAuthCode: expected ErrAuthCodeNotFound, got %v", err)
	}
}

// -------------------------------------------------------------------
// 18. ConsumeAuthCode expired → ErrAuthCodeNotFound
// -------------------------------------------------------------------

func TestConsumeExpiredAuthCode(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	code := &AuthCode{
		CodeHash:      "expired-code-hash",
		ClientID:      "client-1",
		PKCEChallenge: "ch",
		State:         "st",
		RedirectURI:   "https://example.com/callback",
		GrantID:       "grant-x",
		CreatedAt:     now.Add(-20 * time.Minute),
		ExpiresAt:     now.Add(-10 * time.Minute), // already expired
	}

	if err := s.CreateAuthCode(code); err != nil {
		t.Fatalf("CreateAuthCode: %v", err)
	}

	_, err := s.ConsumeAuthCode("expired-code-hash")
	if err != ErrAuthCodeNotFound {
		t.Errorf("expected ErrAuthCodeNotFound for expired code, got %v", err)
	}
}

// -------------------------------------------------------------------
// 19. CreatePendingSession + GetPendingSession + DeletePendingSession
// -------------------------------------------------------------------

func TestPendingSessionLifecycle(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	sess := &PendingSession{
		SessionID:     "session-001",
		ClientID:      "client-1",
		RedirectURI:   "https://example.com/callback",
		CodeChallenge: "challenge-abc",
		State:         "state-xyz",
		CreatedAt:     now,
		ExpiresAt:     now.Add(5 * time.Minute),
	}

	if err := s.CreatePendingSession(sess); err != nil {
		t.Fatalf("CreatePendingSession: %v", err)
	}

	got, err := s.GetPendingSession("session-001")
	if err != nil {
		t.Fatalf("GetPendingSession: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.SessionID != sess.SessionID {
		t.Errorf("SessionID: got %q want %q", got.SessionID, sess.SessionID)
	}
	if got.CodeChallenge != "challenge-abc" {
		t.Errorf("CodeChallenge: got %q", got.CodeChallenge)
	}

	if err := s.DeletePendingSession("session-001"); err != nil {
		t.Fatalf("DeletePendingSession: %v", err)
	}

	afterDelete, err := s.GetPendingSession("session-001")
	if err != nil {
		t.Fatalf("GetPendingSession after delete: %v", err)
	}
	if afterDelete != nil {
		t.Fatal("expected nil after delete")
	}
}

// -------------------------------------------------------------------
// 20. GetGrantByAccessToken
// -------------------------------------------------------------------

func TestGetGrantByAccessToken(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         "grant-by-at",
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStatePending,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if err := s.ActivateGrant("grant-by-at", "access-hash-xyz", "refresh-hash-xyz"); err != nil {
		t.Fatalf("ActivateGrant: %v", err)
	}

	got, err := s.GetGrantByAccessToken("access-hash-xyz")
	if err != nil {
		t.Fatalf("GetGrantByAccessToken: %v", err)
	}
	if got == nil {
		t.Fatal("expected grant, got nil")
	}
	if got.GrantID != "grant-by-at" {
		t.Errorf("GrantID: got %q want %q", got.GrantID, "grant-by-at")
	}
}

// -------------------------------------------------------------------
// 21. GetGrantByRefreshToken
// -------------------------------------------------------------------

func TestGetGrantByRefreshToken(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         "grant-by-rt",
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      GrantStatePending,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if err := s.ActivateGrant("grant-by-rt", "at-hash-abc", "rt-hash-abc"); err != nil {
		t.Fatalf("ActivateGrant: %v", err)
	}

	got, err := s.GetGrantByRefreshToken("rt-hash-abc")
	if err != nil {
		t.Fatalf("GetGrantByRefreshToken: %v", err)
	}
	if got == nil {
		t.Fatal("expected grant, got nil")
	}
	if got.GrantID != "grant-by-rt" {
		t.Errorf("GrantID: got %q want %q", got.GrantID, "grant-by-rt")
	}
}

// -------------------------------------------------------------------
// 22. Integration: encrypt Poltio token, write to grant, read back, decrypt
// -------------------------------------------------------------------

func TestIntegrationEncryptedPoltioToken(t *testing.T) {
	key := testKey(t)
	originalToken := []byte("poltio-api-token-super-secret-value")
	grantID := "grant-integration-01"

	ct, err := Encrypt(originalToken, []byte(grantID), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	g := &OAuthGrant{
		GrantID:         grantID,
		PoltioTokenEnc:  ct,
		PoltioOrgID:     "org-x",
		PoltioAccountID: "acc-x",
		GrantState:      GrantStateActive,
		CreatedAt:       now,
	}
	if err := s.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	got, err := s.GetGrant(grantID)
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if got == nil {
		t.Fatal("expected grant, got nil")
	}

	decrypted, err := Decrypt(got.PoltioTokenEnc, []byte(grantID), key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, originalToken) {
		t.Errorf("decrypted token mismatch: got %q want %q", decrypted, originalToken)
	}
}

// -------------------------------------------------------------------
// Bonus: verify WAL mode is actually on (per advisor recommendation)
// -------------------------------------------------------------------

func TestWALModeApplied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var mode string
	if err := s.readDB.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", mode)
	}
}

// -------------------------------------------------------------------
// Bonus: KeyFromEnv valid
// -------------------------------------------------------------------

func TestKeyFromEnvValid(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	t.Setenv("BRIDGE_ENCRYPTION_KEY", hex.EncodeToString(key))

	got, err := KeyFromEnv()
	if err != nil {
		t.Fatalf("KeyFromEnv: %v", err)
	}
	if !bytes.Equal(got, key) {
		t.Errorf("key mismatch")
	}
}

