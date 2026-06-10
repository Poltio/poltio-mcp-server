package oauth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Poltio/poltio-mcp-server/store"
)

// testEncKey is a fixed 32-byte AES-256 key for tests.
var testEncKey = []byte("01234567890123456789012345678901")

// fakeProfileJSON returns JSON for a profile with one org.
func fakeProfileJSON(accountID, orgID int) string {
	return fmt.Sprintf(`{"id":%d,"organizations":[{"id":%d}]}`, accountID, orgID)
}

// setupConsentSession creates a pending session in the DB and returns
// a request with the __Host-session cookie already set.
func setupConsentSession(t *testing.T, db *store.Store, clientID, redirectURI, state string) *store.PendingSession {
	t.Helper()
	now := time.Now().UTC()
	sess := &store.PendingSession{
		SessionID:     "sess-" + t.Name(),
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		State:         state,
		CreatedAt:     now,
		ExpiresAt:     now.Add(10 * time.Minute),
	}
	if err := db.CreatePendingSession(sess); err != nil {
		t.Fatalf("setupConsentSession: %v", err)
	}
	return sess
}

// consentRequest builds a POST /consent request with the session cookie and poltio_token form field.
func consentRequest(sessionID, poltioToken string) *http.Request {
	form := url.Values{"poltio_token": {poltioToken}}
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "__Host-session", Value: sessionID})
	return req
}

func TestConsentHappy(t *testing.T) {
	// Set up fake Poltio server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer valid-token" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, fakeProfileJSON(42, 99))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	sess := setupConsentSession(t, db, clientID, testRedirectURI, "mystate")

	h := ConsentHandler(db, "http://localhost", testEncKey, 10*time.Minute, srv.URL)

	w := httptest.NewRecorder()
	req := consentRequest(sess.SessionID, "valid-token")
	h(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status: got %d, want 302; body: %s", w.Code, w.Body.String())
	}

	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, testRedirectURI) {
		t.Fatalf("redirect location %q should start with %q", loc, testRedirectURI)
	}

	// Extract code and state from redirect.
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	rawCode := parsed.Query().Get("code")
	if rawCode == "" {
		t.Fatal("expected code in redirect")
	}
	state := parsed.Query().Get("state")
	if state != "mystate" {
		t.Errorf("state: got %q, want %q", state, "mystate")
	}

	// Verify the auth code exists in store by consuming it.
	h2 := sha256.Sum256([]byte(rawCode))
	codeHash := hex.EncodeToString(h2[:])
	authCode, err := db.ConsumeAuthCode(codeHash)
	if err != nil {
		t.Fatalf("ConsumeAuthCode: %v", err)
	}
	if authCode.ClientID != clientID {
		t.Errorf("auth code client_id: got %q, want %q", authCode.ClientID, clientID)
	}

	// Verify grant is in pending state.
	grant, err := db.GetGrant(authCode.GrantID)
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if grant == nil {
		t.Fatal("expected grant, got nil")
	}
	if grant.GrantState != store.GrantStatePending {
		t.Errorf("grant state: got %q, want %q", grant.GrantState, store.GrantStatePending)
	}

	// Verify pending session was deleted.
	deleted, err := db.GetPendingSession(sess.SessionID)
	if err != nil {
		t.Fatalf("GetPendingSession after consent: %v", err)
	}
	if deleted != nil {
		t.Error("pending session should have been deleted after consent")
	}
}

func TestConsentMissingCookie(t *testing.T) {
	db := openTestDB(t)
	h := ConsentHandler(db, "http://localhost", testEncKey, 10*time.Minute, "")

	form := url.Values{"poltio_token": {"some-token"}}
	req := httptest.NewRequest(http.MethodPost, "/consent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// No cookie set.

	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing cookie: got %d, want 400", w.Code)
	}
}

func TestConsentInvalidCookie(t *testing.T) {
	db := openTestDB(t)
	h := ConsentHandler(db, "http://localhost", testEncKey, 10*time.Minute, "")

	w := httptest.NewRecorder()
	req := consentRequest("nonexistent-session-id", "some-token")
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid cookie: got %d, want 400", w.Code)
	}
}

func TestConsentPoltio401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	sess := setupConsentSession(t, db, clientID, testRedirectURI, "st1")

	h := ConsentHandler(db, "http://localhost", testEncKey, 10*time.Minute, srv.URL)

	w := httptest.NewRecorder()
	req := consentRequest(sess.SessionID, "bad-token")
	h(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("poltio 401: got %d, want 200 (re-render)", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "invalid or expired") {
		t.Errorf("poltio 401: expected 'invalid or expired' in body, got: %s", body)
	}
	// Must not redirect.
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("poltio 401: unexpected redirect to %q", loc)
	}
}

func TestConsentPoltio503(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	sess := setupConsentSession(t, db, clientID, testRedirectURI, "st2")

	h := ConsentHandler(db, "http://localhost", testEncKey, 10*time.Minute, srv.URL)

	w := httptest.NewRecorder()
	req := consentRequest(sess.SessionID, "any-token")

	start := time.Now()
	h(w, req)
	elapsed := time.Since(start)

	// Should have retried 3 times total.
	if n := calls.Load(); n != 3 {
		t.Errorf("expected 3 attempts, got %d", n)
	}
	// Should have waited ~1s (2 × 500ms).
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected ~1s elapsed for retries, got %v", elapsed)
	}

	if w.Code != http.StatusOK {
		t.Errorf("poltio 503: got %d, want 200 (re-render)", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "temporarily unavailable") {
		t.Errorf("poltio 503: expected 'temporarily unavailable' in body, got: %s", body)
	}
}

func TestConsentZeroOrganizations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return profile with zero orgs.
		resp := map[string]any{
			"id":            7,
			"organizations": []any{},
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	sess := setupConsentSession(t, db, clientID, testRedirectURI, "st3")

	h := ConsentHandler(db, "http://localhost", testEncKey, 10*time.Minute, srv.URL)

	w := httptest.NewRecorder()
	req := consentRequest(sess.SessionID, "token-with-no-orgs")
	h(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("zero orgs: got %d, want 200 (re-render)", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "no organizations") {
		t.Errorf("zero orgs: expected 'no organizations' in body, got: %s", body)
	}
}

func TestConsentExpiredSession(t *testing.T) {
	db := openTestDB(t)
	// Insert a session that is already expired.
	now := time.Now().UTC()
	sess := &store.PendingSession{
		SessionID:     "expired-sess",
		ClientID:      "some-client",
		RedirectURI:   testRedirectURI,
		CodeChallenge: "abc",
		State:         "s",
		CreatedAt:     now.Add(-20 * time.Minute),
		ExpiresAt:     now.Add(-10 * time.Minute), // expired 10 min ago
	}
	if err := db.CreatePendingSession(sess); err != nil {
		t.Fatalf("CreatePendingSession: %v", err)
	}

	h := ConsentHandler(db, "http://localhost", testEncKey, 10*time.Minute, "")

	w := httptest.NewRecorder()
	req := consentRequest("expired-sess", "some-token")
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expired session: got %d, want 400", w.Code)
	}
}
