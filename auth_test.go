package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Poltio/poltio-mcp-server/client"
)

func TestBearerToken(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"Bearer abc123", "abc123"},
		{"bearer abc123", "abc123"}, // scheme is case-insensitive
		{"Bearer  abc123 ", "abc123"},
		{"", ""},
		{"Bearer", ""},
		{"Bearer ", ""},
		{"Basic abc123", ""},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if tc.header != "" {
			r.Header.Set("Authorization", tc.header)
		}
		if got := bearerToken(r); got != tc.want {
			t.Errorf("bearerToken(%q) = %q, want %q", tc.header, got, tc.want)
		}
	}
}

func TestWithBearerPopulatesContext(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer tok")
	if got, _ := withBearer(context.Background(), r).Value(tokenCtxKey{}).(string); got != "tok" {
		t.Errorf("token in context = %q, want %q", got, "tok")
	}

	bare := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	if v := withBearer(context.Background(), bare).Value(tokenCtxKey{}); v != nil {
		t.Errorf("unauthenticated request put %v in context, want nothing", v)
	}
}

// An unauthenticated /mcp call must return the challenge that starts the OAuth
// flow, otherwise the client never learns where to authenticate.
func TestUnauthenticatedMCPReturnsChallenge(t *testing.T) {
	t.Setenv("MCP_PUBLIC_URL", "https://mcp.example.com")

	reached := false
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if reached {
		t.Error("unauthenticated request reached the MCP server")
	}
	want := `Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource"`
	if got := w.Header().Get("WWW-Authenticate"); got != want {
		t.Errorf("WWW-Authenticate = %q, want %q", got, want)
	}
}

func TestAuthenticatedMCPPassesThrough(t *testing.T) {
	reached := false
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer tok")
	h.ServeHTTP(httptest.NewRecorder(), r)

	if !reached {
		t.Error("authenticated request did not reach the MCP server")
	}
}

// Preflight must not be answered with 401, or browser clients cannot even
// discover the challenge.
func TestPreflightPassesThrough(t *testing.T) {
	reached := false
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodOptions, "/mcp", nil))

	if !reached {
		t.Error("preflight did not reach the MCP server's CORS handling")
	}
}

func TestResourceMetadata(t *testing.T) {
	t.Setenv("MCP_PUBLIC_URL", "https://mcp.example.com/")
	t.Setenv("POLTIO_API_BASE_URL", "https://api-stage.example.com")

	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, resourceMetadataPath, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	// Trailing slash must be trimmed: the resource identifier has to match the
	// one the authorization server issues tokens for.
	if got.Resource != "https://mcp.example.com" {
		t.Errorf("resource = %q, want %q", got.Resource, "https://mcp.example.com")
	}
	if len(got.AuthorizationServers) != 1 || got.AuthorizationServers[0] != "https://api-stage.example.com" {
		t.Errorf("authorization_servers = %v", got.AuthorizationServers)
	}
}

// The cache is what makes switch_organization stick: SetOrgID mutates the
// cached client, so the same token must resolve to the same pointer, and the
// selected organization must travel on subsequent calls.
func TestClientForCtxCachesPerToken(t *testing.T) {
	var gotAuth, gotOrg string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/platform/account/profile" {
			_, _ = w.Write([]byte(`{"organizations":[{"id":123}]}`))
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotOrg = r.Header.Get("Organization-Id")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()
	t.Setenv("POLTIO_API_BASE_URL", ts.URL)

	const tok = "cache-test-token"
	t.Cleanup(func() {
		clientsMu.Lock()
		delete(clients, tok)
		clientsMu.Unlock()
	})

	ctx := context.WithValue(context.Background(), tokenCtxKey{}, tok)
	first, err := clientForCtx(ctx)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	second, err := clientForCtx(ctx)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if first != second {
		t.Fatalf("same token resolved to different clients (%p, %p)", first, second)
	}

	if _, err := second.Get("/platform/content", nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if gotAuth != "Bearer "+tok {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer "+tok)
	}
	// Set by activateFirstOrg on first resolve; proves org state survives.
	if gotOrg != "123" {
		t.Errorf("Organization-Id = %q, want %q", gotOrg, "123")
	}
}

// A different token must never share a client, or one user's organization
// selection would leak into another's requests.
func TestClientForCtxIsolatesTokens(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"organizations":[{"id":1}]}`))
	}))
	defer ts.Close()
	t.Setenv("POLTIO_API_BASE_URL", ts.URL)

	t.Cleanup(func() {
		clientsMu.Lock()
		delete(clients, "tok-a")
		delete(clients, "tok-b")
		clientsMu.Unlock()
	})

	a, err := clientForCtx(context.WithValue(context.Background(), tokenCtxKey{}, "tok-a"))
	if err != nil {
		t.Fatalf("resolve a: %v", err)
	}
	b, err := clientForCtx(context.WithValue(context.Background(), tokenCtxKey{}, "tok-b"))
	if err != nil {
		t.Fatalf("resolve b: %v", err)
	}
	if a == b {
		t.Error("different tokens share a client")
	}
}

// stdio mode has no per-request token; handlers must fall back to the client
// built from POLTIO_API_TOKEN rather than getting nil.
func TestClientForCtxFallsBackToBaseClient(t *testing.T) {
	prev := baseClient
	t.Cleanup(func() { baseClient = prev })

	baseClient = client.New("env-token")
	c, err := clientForCtx(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != baseClient {
		t.Errorf("no token in context resolved to %p, want the base client %p", c, baseClient)
	}
}
