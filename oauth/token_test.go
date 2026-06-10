package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Poltio/poltio-mcp-server/store"
)

// insertActiveGrant creates a grant, activates it with the given token hashes, and returns the grant.
func insertActiveGrant(t *testing.T, db *store.Store, grantID, accessHash, refreshHash string) *store.OAuthGrant {
	t.Helper()
	now := time.Now().UTC()
	g := &store.OAuthGrant{
		GrantID:         grantID,
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      store.GrantStatePending,
		CreatedAt:       now,
	}
	if err := db.CreateGrant(g); err != nil {
		t.Fatalf("insertActiveGrant CreateGrant: %v", err)
	}
	if err := db.ActivateGrant(grantID, accessHash, refreshHash); err != nil {
		t.Fatalf("insertActiveGrant ActivateGrant: %v", err)
	}
	grant, err := db.GetGrant(grantID)
	if err != nil {
		t.Fatalf("insertActiveGrant GetGrant: %v", err)
	}
	return grant
}

// insertPendingGrantWithCode creates a pending grant and a fresh auth code for it.
// Returns the raw code (hex string) and the clientID used.
func insertPendingGrantWithCode(t *testing.T, db *store.Store, grantID, clientID, redirectURI, pkceChallenge string) string {
	t.Helper()
	now := time.Now().UTC()
	g := &store.OAuthGrant{
		GrantID:         grantID,
		PoltioOrgID:     "org-1",
		PoltioAccountID: "acc-1",
		GrantState:      store.GrantStatePending,
		CreatedAt:       now,
	}
	if err := db.CreateGrant(g); err != nil {
		t.Fatalf("insertPendingGrantWithCode CreateGrant: %v", err)
	}

	// Raw code: "testcode-<grantID>" → deterministic for tests.
	rawCode := "rawcode-" + grantID
	h := sha256.Sum256([]byte(rawCode))
	codeHash := hex.EncodeToString(h[:])

	authCode := &store.AuthCode{
		CodeHash:      codeHash,
		ClientID:      clientID,
		PKCEChallenge: pkceChallenge,
		State:         "teststate",
		RedirectURI:   redirectURI,
		GrantID:       grantID,
		CreatedAt:     now,
		ExpiresAt:     now.Add(5 * time.Minute),
	}
	if err := db.CreateAuthCode(authCode); err != nil {
		t.Fatalf("insertPendingGrantWithCode CreateAuthCode: %v", err)
	}
	return rawCode
}

// s256Test computes S256 for a verifier (mirrors token.go s256).
func s256Test(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// tokenExchangeRequest builds a POST /token request for authorization_code grant.
func tokenExchangeRequest(code, clientID, redirectURI, verifier string) *http.Request {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// tokenRefreshRequest builds a POST /token request for refresh_token grant.
func tokenRefreshRequest(refreshToken string) *http.Request {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// parseTokenResponse parses the JSON token response.
func parseTokenResponse(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("parseTokenResponse: %v", err)
	}
	return m
}

// --- Test 1: Happy code exchange ---

func TestTokenHappyCodeExchange(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := s256Test(verifier)
	rawCode := insertPendingGrantWithCode(t, db, "grant-happy", clientID, testRedirectURI, challenge)

	h := TokenHandler(db)
	w := httptest.NewRecorder()
	req := tokenExchangeRequest(rawCode, clientID, testRedirectURI, verifier)
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	resp := parseTokenResponse(t, w.Body.Bytes())
	if resp["access_token"] == "" || resp["access_token"] == nil {
		t.Error("expected access_token in response")
	}
	if resp["refresh_token"] == "" || resp["refresh_token"] == nil {
		t.Error("expected refresh_token in response")
	}
	if resp["token_type"] != "bearer" {
		t.Errorf("token_type: got %v, want bearer", resp["token_type"])
	}
	if resp["expires_in"] != float64(3600) {
		t.Errorf("expires_in: got %v, want 3600", resp["expires_in"])
	}

	// Verify grant is now active.
	grant, err := db.GetGrant("grant-happy")
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if grant.GrantState != store.GrantStateActive {
		t.Errorf("grant state: got %q, want active", grant.GrantState)
	}
	if grant.AccessTokenHash == "" {
		t.Error("expected non-empty access_token_hash after exchange")
	}
}

// --- Test 2: Code reuse ---

func TestTokenCodeReuse(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := s256Test(verifier)
	rawCode := insertPendingGrantWithCode(t, db, "grant-reuse", clientID, testRedirectURI, challenge)

	h := TokenHandler(db)

	// First exchange — should succeed.
	w1 := httptest.NewRecorder()
	h(w1, tokenExchangeRequest(rawCode, clientID, testRedirectURI, verifier))
	if w1.Code != http.StatusOK {
		t.Fatalf("first exchange: got %d; body: %s", w1.Code, w1.Body.String())
	}

	// Second exchange — same code should fail.
	w2 := httptest.NewRecorder()
	h(w2, tokenExchangeRequest(rawCode, clientID, testRedirectURI, verifier))
	if w2.Code == http.StatusOK {
		t.Fatal("second exchange with same code should fail")
	}
	resp := parseTokenResponse(t, w2.Body.Bytes())
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected error=invalid_grant, got %v", resp["error"])
	}
}

// --- Test 3: PKCE mismatch ---

func TestTokenPKCEMismatch(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := s256Test(verifier)
	rawCode := insertPendingGrantWithCode(t, db, "grant-pkce", clientID, testRedirectURI, challenge)

	h := TokenHandler(db)
	w := httptest.NewRecorder()
	req := tokenExchangeRequest(rawCode, clientID, testRedirectURI, "wrong-verifier-value")
	h(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("PKCE mismatch should fail")
	}
	resp := parseTokenResponse(t, w.Body.Bytes())
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected error=invalid_grant, got %v", resp["error"])
	}
}

// --- Test 4: client_id mismatch ---

func TestTokenClientIDMismatch(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := s256Test(verifier)
	rawCode := insertPendingGrantWithCode(t, db, "grant-clientid", clientID, testRedirectURI, challenge)

	h := TokenHandler(db)
	w := httptest.NewRecorder()
	req := tokenExchangeRequest(rawCode, "wrong-client-id", testRedirectURI, verifier)
	h(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("client_id mismatch should fail")
	}
	resp := parseTokenResponse(t, w.Body.Bytes())
	if resp["error"] != "invalid_client" {
		t.Errorf("expected error=invalid_client, got %v", resp["error"])
	}
}

// --- Test 5: redirect_uri mismatch ---

func TestTokenRedirectURIMismatch(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := s256Test(verifier)
	rawCode := insertPendingGrantWithCode(t, db, "grant-redirect", clientID, testRedirectURI, challenge)

	h := TokenHandler(db)
	w := httptest.NewRecorder()
	req := tokenExchangeRequest(rawCode, clientID, "https://evil.example.com/callback", verifier)
	h(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("redirect_uri mismatch should fail")
	}
	resp := parseTokenResponse(t, w.Body.Bytes())
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected error=invalid_grant, got %v", resp["error"])
	}
}

// --- Test 6: JSON body → 415 ---

func TestTokenJSONBodyRejected(t *testing.T) {
	db := openTestDB(t)
	h := TokenHandler(db)

	body := `{"grant_type":"authorization_code","code":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("JSON body: got %d, want 415", w.Code)
	}
}

// --- Test 7: Refresh rotation happy path ---

func TestTokenRefreshRotationHappy(t *testing.T) {
	db := openTestDB(t)

	// Create an active grant with known token values.
	rawRefresh := "test-refresh-token-initial"
	h2 := sha256.Sum256([]byte(rawRefresh))
	refreshHash := hex.EncodeToString(h2[:])

	rawAccess := "test-access-token-initial"
	ha := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(ha[:])

	insertActiveGrant(t, db, "grant-refresh-happy", accessHash, refreshHash)

	handler := TokenHandler(db)
	w := httptest.NewRecorder()
	req := tokenRefreshRequest(rawRefresh)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("refresh rotation: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	resp := parseTokenResponse(t, w.Body.Bytes())
	newAccessToken, ok := resp["access_token"].(string)
	if !ok || newAccessToken == "" {
		t.Fatal("expected new access_token")
	}
	newRefreshToken, ok := resp["refresh_token"].(string)
	if !ok || newRefreshToken == "" {
		t.Fatal("expected new refresh_token")
	}

	// New tokens must differ from the original.
	if newAccessToken == rawAccess {
		t.Error("new access_token should differ from original")
	}
	if newRefreshToken == rawRefresh {
		t.Error("new refresh_token should differ from original")
	}

	// Old refresh token is now invalid.
	w2 := httptest.NewRecorder()
	req2 := tokenRefreshRequest(rawRefresh)
	handler(w2, req2)

	if w2.Code == http.StatusOK {
		t.Fatal("old refresh token should be invalid after rotation")
	}
	resp2 := parseTokenResponse(t, w2.Body.Bytes())
	if resp2["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant for old refresh token, got %v", resp2["error"])
	}
}

// --- Test 8: Refresh with unknown token ---

func TestTokenRefreshUnknownToken(t *testing.T) {
	db := openTestDB(t)
	handler := TokenHandler(db)

	w := httptest.NewRecorder()
	req := tokenRefreshRequest("completely-unknown-refresh-token")
	handler(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("unknown refresh token should fail")
	}
	resp := parseTokenResponse(t, w.Body.Bytes())
	if resp["error"] != "invalid_grant" {
		t.Errorf("expected invalid_grant, got %v", resp["error"])
	}
}

// --- Test 9: Concurrent refresh rotation — exactly one wins ---

func TestTokenRefreshConcurrentRotation(t *testing.T) {
	db := openTestDB(t)

	rawRefresh := "concurrent-refresh-token"
	h2 := sha256.Sum256([]byte(rawRefresh))
	refreshHash := hex.EncodeToString(h2[:])

	rawAccess := "concurrent-access-token"
	ha := sha256.Sum256([]byte(rawAccess))
	accessHash := hex.EncodeToString(ha[:])

	insertActiveGrant(t, db, "grant-concurrent", accessHash, refreshHash)

	handler := TokenHandler(db)

	const goroutines = 10
	results := make([]int, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req := tokenRefreshRequest(rawRefresh)
			handler(w, req)
			results[i] = w.Code
		}()
	}
	wg.Wait()

	successCount := 0
	for _, code := range results {
		if code == http.StatusOK {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("expected exactly 1 successful rotation, got %d (results: %v)", successCount, results)
	}
}
