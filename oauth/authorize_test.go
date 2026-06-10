package oauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Poltio/poltio-mcp-server/store"
)

// registerTestClient registers a client with the given redirect URIs in the test DB,
// returning its client_id. Using claudeAICallback as the canonical test redirect URI.
func registerTestClient(t *testing.T, db *store.Store, redirectURIs []string) string {
	t.Helper()
	clientID := "test-client-" + t.Name()
	now := time.Now().UTC()
	err := db.CreateClient(&store.OAuthClient{
		ClientID:     clientID,
		RedirectURIs: redirectURIs,
		CreatedAt:    now,
		ExpiresAt:    now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("registerTestClient: %v", err)
	}
	return clientID
}

const testRedirectURI = "https://claude.ai/api/mcp/auth_callback"

// validAuthorizeURL builds a valid /authorize URL for the given clientID.
func validAuthorizeURL(clientID string) string {
	return "/authorize?" +
		"client_id=" + clientID +
		"&redirect_uri=" + testRedirectURI +
		"&response_type=code" +
		"&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" +
		"&code_challenge_method=S256" +
		"&state=xyzzy"
}

func TestAuthorizeHappy(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	h := AuthorizeHandler(db, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, validAuthorizeURL(clientID), nil)
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Connect Poltio") {
		t.Errorf("expected consent HTML, got: %s", body)
	}

	// Check session cookie is set.
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "__Host-session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected __Host-session cookie, none found")
	}
	if sessionCookie.Value == "" {
		t.Error("session cookie value should not be empty")
	}
	if !sessionCookie.HttpOnly {
		t.Error("session cookie should be HttpOnly")
	}
	if sessionCookie.Path != "/" {
		t.Errorf("session cookie Path: got %q, want %q", sessionCookie.Path, "/")
	}
}

func TestAuthorizeUnregisteredClient(t *testing.T) {
	db := openTestDB(t)
	h := AuthorizeHandler(db, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id=unknown-client&redirect_uri="+testRedirectURI+"&response_type=code&code_challenge=abc&code_challenge_method=S256&state=st", nil)
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unregistered client: got %d, want 400", w.Code)
	}
	// Must not redirect.
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("unregistered client: unexpected redirect to %q", loc)
	}
}

func TestAuthorizeRedirectURIMismatch(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	h := AuthorizeHandler(db, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id="+clientID+"&redirect_uri=https://evil.example.com/callback&response_type=code&code_challenge=abc&code_challenge_method=S256&state=st", nil)
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("redirect_uri mismatch: got %d, want 400", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("redirect_uri mismatch: unexpected redirect to %q", loc)
	}
}

func TestAuthorizeMissingState(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	h := AuthorizeHandler(db, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id="+clientID+"&redirect_uri="+testRedirectURI+"&response_type=code&code_challenge=abc&code_challenge_method=S256", nil)
	h(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("missing state: got %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_request") {
		t.Errorf("missing state: expected error=invalid_request in redirect, got %q", loc)
	}
}

func TestAuthorizeMissingCodeChallenge(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	h := AuthorizeHandler(db, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id="+clientID+"&redirect_uri="+testRedirectURI+"&response_type=code&code_challenge_method=S256&state=st", nil)
	h(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("missing code_challenge: got %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_request") {
		t.Errorf("missing code_challenge: expected error=invalid_request in redirect, got %q", loc)
	}
}

func TestAuthorizeWrongCodeChallengeMethod(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	h := AuthorizeHandler(db, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id="+clientID+"&redirect_uri="+testRedirectURI+"&response_type=code&code_challenge=abc&code_challenge_method=plain&state=st", nil)
	h(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("wrong method: got %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_request") {
		t.Errorf("wrong method: expected error=invalid_request in redirect, got %q", loc)
	}
}

func TestAuthorizeWrongResponseType(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	h := AuthorizeHandler(db, 0)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id="+clientID+"&redirect_uri="+testRedirectURI+"&response_type=token&code_challenge=abc&code_challenge_method=S256&state=st", nil)
	h(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("wrong response_type: got %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "error=unsupported_response_type") {
		t.Errorf("wrong response_type: expected error=unsupported_response_type in redirect, got %q", loc)
	}
}

func TestAuthorizeConcurrentIsolation(t *testing.T) {
	db := openTestDB(t)
	clientID := registerTestClient(t, db, []string{testRedirectURI})
	h := AuthorizeHandler(db, 0)

	const goroutines = 10
	sessionIDs := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, validAuthorizeURL(clientID), nil)
			h(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("goroutine %d: status %d", i, w.Code)
				return
			}
			for _, c := range w.Result().Cookies() {
				if c.Name == "__Host-session" {
					sessionIDs[i] = c.Value
					break
				}
			}
		}()
	}
	wg.Wait()

	// All session IDs must be non-empty and unique.
	seen := make(map[string]bool)
	for i, id := range sessionIDs {
		if id == "" {
			t.Errorf("goroutine %d: empty session ID", i)
			continue
		}
		if seen[id] {
			t.Errorf("goroutine %d: duplicate session ID %q", i, id)
		}
		seen[id] = true
	}
}
