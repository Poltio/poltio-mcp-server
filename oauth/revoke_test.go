package oauth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Poltio/poltio-mcp-server/store"
)

// revokeRequest builds a POST /revoke request for the given token.
func revokeRequest(token string) *http.Request {
	form := url.Values{"token": {token}}
	req := httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// --- Test 1: Revoke by access token ---

func TestRevokeByAccessToken(t *testing.T) {
	db := openTestDB(t)

	rawAccess := "test-access-token-revoke"
	ha := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(ha[:])

	rawRefresh := "test-refresh-token-revoke"
	hr := sha256.Sum256([]byte(rawRefresh))
	refreshHash := hex.EncodeToString(hr[:])

	insertActiveGrant(t, db, "grant-revoke-by-at", accessHash, refreshHash)

	h := RevokeHandler(db)
	w := httptest.NewRecorder()
	req := revokeRequest(rawAccess)
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("revoke by access token: got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	// Grant should now be revoked.
	grant, err := db.GetGrant("grant-revoke-by-at")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if grant.GrantState != store.GrantStateRevoked {
		t.Errorf("expected revoked, got %q", grant.GrantState)
	}
}

// --- Test 2: Revoke by refresh token ---

func TestRevokeByRefreshToken(t *testing.T) {
	db := openTestDB(t)

	rawAccess := "test-access-token-revoke-rt"
	ha := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(ha[:])

	rawRefresh := "test-refresh-token-revoke-rt"
	hr := sha256.Sum256([]byte(rawRefresh))
	refreshHash := hex.EncodeToString(hr[:])

	insertActiveGrant(t, db, "grant-revoke-by-rt", accessHash, refreshHash)

	h := RevokeHandler(db)
	w := httptest.NewRecorder()
	req := revokeRequest(rawRefresh)
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("revoke by refresh token: got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	// Grant should now be revoked.
	grant, err := db.GetGrant("grant-revoke-by-rt")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if grant.GrantState != store.GrantStateRevoked {
		t.Errorf("expected revoked, got %q", grant.GrantState)
	}
}

// --- Test 3: Revoke unknown token → 200 (RFC 7009 §2.2) ---

func TestRevokeUnknownToken(t *testing.T) {
	db := openTestDB(t)
	h := RevokeHandler(db)

	w := httptest.NewRecorder()
	req := revokeRequest("completely-unknown-token-value")
	h(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("unknown token: got %d, want 200 (RFC 7009 §2.2)", w.Code)
	}
}
