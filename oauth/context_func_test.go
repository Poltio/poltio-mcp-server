package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/store"
)

// testKey returns a 32-byte AES key for tests.
func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

// openTestStore opens a throwaway SQLite store in a temp dir.
func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// insertBridgeGrant creates a grant with an encrypted poltio token and returns the raw access token.
func insertBridgeGrant(t *testing.T, db *store.Store, key []byte, grantID, rawToken, orgID string) string {
	t.Helper()
	enc, err := store.Encrypt([]byte(rawToken), []byte(grantID), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	rawAccess := "access-" + grantID
	h := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(h[:])

	g := &store.OAuthGrant{
		GrantID:         grantID,
		AccessTokenHash: accessHash,
		PoltioTokenEnc:  enc,
		PoltioOrgID:     orgID,
		GrantState:      store.GrantStateActive,
		CreatedAt:       time.Now().UTC(),
	}
	if err := db.CreateGrant(g); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	return rawAccess
}

// 1. Active grant → correct client in context
func TestBridgeContextFunc_ActiveGrant(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	rawAccess := insertBridgeGrant(t, db, key, "grant-1", "poltio-secret-token", "42")

	fn := BridgeContextFunc(db, key, "")
	r := httptest.NewRequest("GET", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+rawAccess)

	ctx := fn(context.Background(), r)

	pc, err := client.FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext: %v (ctx has no client)", err)
	}
	if pc == nil {
		t.Fatal("expected non-nil PoltioClient in context")
	}
	if NeedsAuth(ctx) {
		t.Fatal("expected NeedsAuth to be false for active grant")
	}
	if NeedsReconnect(ctx) {
		t.Fatal("expected NeedsReconnect to be false for active grant")
	}
	grantID := client.GrantIDFromContext(ctx)
	if grantID != "grant-1" {
		t.Fatalf("expected grant ID 'grant-1', got %q", grantID)
	}
}

// 2. Unknown token → NeedsAuth sentinel
func TestBridgeContextFunc_UnknownToken(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	fn := BridgeContextFunc(db, key, "")
	r := httptest.NewRequest("GET", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer no-such-token")

	ctx := fn(context.Background(), r)

	if !NeedsAuth(ctx) {
		t.Fatal("expected NeedsAuth for unknown token")
	}
	_, err := client.FromContext(ctx)
	if err == nil {
		t.Fatal("expected no client in context for unknown token")
	}
}

// 3. Missing Authorization header → NeedsAuth sentinel
func TestBridgeContextFunc_MissingAuth(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	fn := BridgeContextFunc(db, key, "")
	r := httptest.NewRequest("GET", "/mcp", nil)
	// No Authorization header

	ctx := fn(context.Background(), r)

	if !NeedsAuth(ctx) {
		t.Fatal("expected NeedsAuth for missing Authorization header")
	}
}

// 4. Revoked grant → NeedsReconnect sentinel
func TestBridgeContextFunc_RevokedGrant(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	rawAccess := insertBridgeGrant(t, db, key, "grant-revoked", "poltio-token", "1")
	// Revoke it
	h := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(h[:])
	// Find the grant and revoke it
	g, _ := db.GetGrantByAccessToken(accessHash)
	if err := db.RevokeGrant(g.GrantID); err != nil {
		t.Fatalf("RevokeGrant: %v", err)
	}

	fn := BridgeContextFunc(db, key, "")
	r := httptest.NewRequest("GET", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+rawAccess)

	ctx := fn(context.Background(), r)

	if !NeedsReconnect(ctx) {
		t.Fatal("expected NeedsReconnect for revoked grant")
	}
	if NeedsAuth(ctx) {
		t.Fatal("expected NeedsAuth to be false for revoked grant (should be reconnect)")
	}
}

// 5. needs_reconnect grant → NeedsReconnect sentinel
func TestBridgeContextFunc_NeedsReconnectGrant(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	rawAccess := insertBridgeGrant(t, db, key, "grant-nr", "poltio-token", "1")
	h := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(h[:])
	g, _ := db.GetGrantByAccessToken(accessHash)
	if err := db.MarkNeedsReconnect(g.GrantID); err != nil {
		t.Fatalf("MarkNeedsReconnect: %v", err)
	}

	fn := BridgeContextFunc(db, key, "")
	r := httptest.NewRequest("GET", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+rawAccess)

	ctx := fn(context.Background(), r)

	if !NeedsReconnect(ctx) {
		t.Fatal("expected NeedsReconnect for needs_reconnect grant")
	}
}

// 6. OrgOverride is respected
func TestBridgeContextFunc_OrgOverride(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	rawAccess := insertBridgeGrant(t, db, key, "grant-org", "poltio-token", "original-org")
	// Set org_override
	h := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(h[:])
	g, _ := db.GetGrantByAccessToken(accessHash)
	if err := db.SetOrgOverride(g.GrantID, "overridden-org"); err != nil {
		t.Fatalf("SetOrgOverride: %v", err)
	}

	fn := BridgeContextFunc(db, key, "")
	r := httptest.NewRequest("GET", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+rawAccess)

	ctx := fn(context.Background(), r)

	pc, err := client.FromContext(ctx)
	if err != nil || pc == nil {
		t.Fatalf("expected PoltioClient in context, got err: %v", err)
	}
	// NeedsAuth/NeedsReconnect should both be false
	if NeedsAuth(ctx) || NeedsReconnect(ctx) {
		t.Fatal("expected no sentinel for active grant with org override")
	}
}

// 7. Two concurrent requests with different tokens — no cross-contamination
func TestBridgeContextFunc_ConcurrentNoCrossContamination(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	token1 := insertBridgeGrant(t, db, key, "grant-c1", "secret-1", "org-1")
	token2 := insertBridgeGrant(t, db, key, "grant-c2", "secret-2", "org-2")

	fn := BridgeContextFunc(db, key, "")

	const iterations = 50
	var wg sync.WaitGroup
	errs := make(chan string, iterations*2)

	for range iterations {
		wg.Add(2)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/mcp", nil)
			r.Header.Set("Authorization", "Bearer "+token1)
			ctx := fn(context.Background(), r)
			grantID := client.GrantIDFromContext(ctx)
			if grantID != "grant-c1" {
				errs <- "token1 got grant ID " + grantID
			}
		}()
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/mcp", nil)
			r.Header.Set("Authorization", "Bearer "+token2)
			ctx := fn(context.Background(), r)
			grantID := client.GrantIDFromContext(ctx)
			if grantID != "grant-c2" {
				errs <- "token2 got grant ID " + grantID
			}
		}()
	}
	wg.Wait()
	close(errs)

	for e := range errs {
		t.Error(e)
	}
}
