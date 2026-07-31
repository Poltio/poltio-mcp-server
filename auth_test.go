package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

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

// stubAPI points the client at a fake Poltio API for the duration of a test and
// forgets any client cached under tok, so tokens can't leak between tests.
func stubAPI(t *testing.T, tok string, h http.HandlerFunc) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	t.Setenv("POLTIO_API_BASE_URL", ts.URL)
	t.Cleanup(func() {
		clientsMu.Lock()
		delete(clients, tok)
		clientsMu.Unlock()
	})
}

// apiAccepts answers the organization lookup that activates a client.
func apiAccepts(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`{"organizations":[{"id":1}]}`))
}

func TestAuthenticatedMCPPassesThrough(t *testing.T) {
	const tok = "good-token"
	stubAPI(t, tok, apiAccepts)

	reached := false
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(httptest.NewRecorder(), r)

	if !reached {
		t.Error("authenticated request did not reach the MCP server")
	}
}

// An expired or revoked token has to produce a 401 challenge. If it reaches the
// MCP server instead, the client sees a successful initialize, never re-runs
// the OAuth flow, and fails every tool call with an error it cannot act on.
func TestRejectedTokenGetsChallenge(t *testing.T) {
	const tok = "expired-token"
	stubAPI(t, tok, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	reached := false
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(w, r)

	if reached {
		t.Error("a rejected token reached the MCP server")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	// RFC 6750: a token was presented and rejected, so the challenge says why.
	auth := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(auth, `error="invalid_token"`) {
		t.Errorf("WWW-Authenticate = %q, want an invalid_token error code", auth)
	}
	if !strings.Contains(auth, "resource_metadata=") {
		t.Errorf("WWW-Authenticate = %q, missing resource_metadata", auth)
	}
}

// An unreachable API is not an authentication failure. Answering 401 would send
// the client through the OAuth flow to no effect. The upstream detail must stay
// out of the response — it can carry internal URLs and paths.
func TestUpstreamFailureIsNotAChallenge(t *testing.T) {
	const tok = "api-down-token"
	const secret = "internal-host.svc.cluster.local"
	stubAPI(t, tok, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(secret))
	})

	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(w, r)

	if w.Code == http.StatusUnauthorized {
		t.Error("an upstream outage was reported as an authentication failure")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
	if auth := w.Header().Get("WWW-Authenticate"); auth != "" {
		t.Errorf("upstream outage sent a challenge: %q", auth)
	}
	if body := w.Body.String(); strings.Contains(body, secret) {
		t.Errorf("upstream error text reached the client: %s", body)
	}
}

// Tokens expire — every two weeks, for everyone. Once one is cached, the HTTP
// layer stops re-validating it, so a tool call failing with ErrUnauthorized is
// the only signal left that it died. If that does not evict the cache entry,
// every later call fails identically until the pod restarts and the client is
// never told to re-authenticate.
func TestExpiredTokenIsEvictedSoReauthCanHappen(t *testing.T) {
	const tok = "expires-mid-session"
	stubAPI(t, tok, apiAccepts)

	// Prime the cache the way a first successful request would.
	if _, err := clientForToken(tok); err != nil {
		t.Fatalf("priming the cache: %v", err)
	}
	clientsMu.Lock()
	_, cached := clients[tok]
	clientsMu.Unlock()
	if !cached {
		t.Fatal("token was not cached, test proves nothing")
	}

	// The token dies upstream; the next tool call surfaces it wrapped, exactly
	// as the tools package does with %w.
	handler := withAuth(func(*client.PoltioClient) toolHandler {
		return func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return nil, fmt.Errorf("list_content: %w", client.ErrUnauthorized)
		}
	})

	ctx := context.WithValue(context.Background(), tokenCtxKey{}, tok)
	_, err := handler(ctx, mcp.CallToolRequest{})
	if err == nil {
		t.Fatal("expected the handler error to propagate")
	}
	// An OAuth caller has no POLTIO_API_TOKEN to fix, so the sentinel's own
	// advice must not reach them.
	if strings.Contains(err.Error(), "POLTIO_API_TOKEN") {
		t.Errorf("OAuth caller got stdio advice: %v", err)
	}
	if !strings.Contains(err.Error(), "reconnect") {
		t.Errorf("error does not tell the user to reconnect: %v", err)
	}

	clientsMu.Lock()
	_, still := clients[tok]
	clientsMu.Unlock()
	if still {
		t.Error("expired token still cached; the next request will skip validation and never get a 401")
	}
}

// stdio has no connector to reconnect and no per-request token — the sentinel's
// env-var advice is exactly right there, so the OAuth rewrite must not apply.
func TestStdioKeepsEnvVarAdvice(t *testing.T) {
	handler := withAuth(func(*client.PoltioClient) toolHandler {
		return func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return nil, fmt.Errorf("list_content: %w", client.ErrUnauthorized)
		}
	})

	// No token in context is what stdio mode looks like.
	_, err := handler(context.Background(), mcp.CallToolRequest{})
	if err == nil {
		t.Fatal("expected the handler error to propagate")
	}
	if !strings.Contains(err.Error(), "POLTIO_API_TOKEN") {
		t.Errorf("stdio lost the advice it needs: %v", err)
	}
}

// A valid token on an account with no organization is a permanent, user-fixable
// state — not an outage. 503 would invite retries that cannot succeed and can
// read as a service failure to whatever is watching.
func TestNoOrganizationIsForbidden(t *testing.T) {
	const tok = "orgless-token"
	stubAPI(t, tok, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"organizations":[]}`))
	})

	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	// Actionable by the user, so this one is worth telling them.
	if body := w.Body.String(); !strings.Contains(body, "no organization") {
		t.Errorf("403 body does not say what to fix: %s", body)
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
		ResourceName         string   `json:"resource_name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	// Trailing slash trimmed, and the identifier is the MCP endpoint rather than
	// the origin — a client comparing it against the URL the user typed must match.
	if got.Resource != "https://mcp.example.com/mcp" {
		t.Errorf("resource = %q, want %q", got.Resource, "https://mcp.example.com/mcp")
	}
	if got.ResourceName != "Poltio" {
		t.Errorf("resource_name = %q, want %q", got.ResourceName, "Poltio")
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

// Two first-use requests for the same token race: both miss the cache, both
// build a client, and both store. If the second store overwrites the first,
// any state already applied to the returned client — a switch_organization
// call, say — is silently dropped for every later request.
func TestClientForCtxConcurrentFirstUseSharesOneClient(t *testing.T) {
	const n = 2

	var inFlight atomic.Int32
	barrier := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold every caller inside the network call until all of them have
		// gotten past the cache miss, forcing the exact interleaving.
		if inFlight.Add(1) == n {
			close(barrier)
		}
		select {
		case <-barrier:
		case <-time.After(5 * time.Second): // don't hang CI on a regression
		}
		_, _ = w.Write([]byte(`{"organizations":[{"id":7}]}`))
	}))
	defer ts.Close()
	t.Setenv("POLTIO_API_BASE_URL", ts.URL)

	const tok = "concurrent-token"
	t.Cleanup(func() {
		clientsMu.Lock()
		delete(clients, tok)
		clientsMu.Unlock()
	})

	ctx := context.WithValue(context.Background(), tokenCtxKey{}, tok)
	got := make([]*client.PoltioClient, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, err := clientForCtx(ctx)
			if err != nil {
				t.Errorf("resolve %d: %v", i, err)
				return
			}
			got[i] = c
		}(i)
	}
	wg.Wait()

	if got[0] != got[1] {
		t.Errorf("concurrent first use returned different clients (%p, %p); one overwrote the other", got[0], got[1])
	}

	clientsMu.Lock()
	cached := clients[tok]
	clientsMu.Unlock()
	if cached != got[0] {
		t.Errorf("cached client %p is not the one handed to callers %p", cached, got[0])
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

// The icon must be fetchable without a token — a client renders it before the
// user has authenticated, so serving it behind the 401 would leave it blank.
func TestIconServedUnauthenticated(t *testing.T) {
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, iconPath, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	// Guard the embed: a missing or truncated file would still serve 200.
	if got := w.Body.Bytes(); len(got) < 8 || string(got[1:4]) != "PNG" {
		t.Errorf("body is not a PNG (%d bytes)", len(got))
	}
}

// The public discovery endpoints are reads: a write method must be refused
// rather than answered with the payload.
func TestPublicEndpointsRejectWriteMethods(t *testing.T) {
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	for _, path := range []string{iconPath, resourceMetadataPath} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(method, path, nil))

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s = %d, want 405", method, path, w.Code)
			}
			if w.Body.Len() > 64 {
				t.Errorf("%s %s returned a %d-byte body; payload should be withheld",
					method, path, w.Body.Len())
			}
			if allow := w.Header().Get("Allow"); allow == "" {
				t.Errorf("%s %s: 405 without an Allow header", method, path)
			}
		}
	}
}

// The load balancer health-checks the backend over HTTP. A non-200 here takes
// the whole service out ("no healthy upstream") even while pods are Ready, so
// these paths must answer 200 without a token — and nothing else may.
func TestHealthEndpoints(t *testing.T) {
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	for _, path := range []string{"/", "/healthz"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, w.Code)
		}
	}

	// Health endpoints are probed by infrastructure, never cross-origin from a
	// browser, so they must not advertise CORS — and a write method is still a
	// 405 rather than a 200.
	for _, path := range []string{"/", "/healthz"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "" {
			t.Errorf("GET %s advertises Access-Control-Allow-Origin: %q", path, origin)
		}

		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s = %d, want 405", path, w.Code)
		}
	}

	// The root must be an exact match. A bare "/" ServeMux pattern is a subtree
	// and would answer 200 for every unknown path — the behaviour that hid the
	// health-check breakage until the backend went down. "/mcp/foo" is here for
	// the same reason: the MCP endpoint is two exact paths, not a subtree.
	for _, path := range []string{"/nope", "/oauth/authorize", "/mcpx", "/mcp/foo"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404 (root pattern is over-matching)", path, w.Code)
		}
	}
}

// A client that normalises the resource URL derives the slashed metadata path;
// both forms must serve, and neither may swallow unrelated sub-paths.
func TestResourceMetadataPathVariants(t *testing.T) {
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	serves := []string{
		resourceMetadataPath,
		resourceMetadataPath + "/mcp",
		resourceMetadataPath + "/mcp/",
	}
	for _, path := range serves {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, w.Code)
		}
	}

	// {$} anchors the slashed pattern; a subtree match would answer here too.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, resourceMetadataPath+"/mcp/foo", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GET %s/mcp/foo = %d, want 404", resourceMetadataPath, w.Code)
	}
}

// A preflight must be answered with the CORS contract, not with the payload.
func TestIconPreflight(t *testing.T) {
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, iconPath, nil)
	r.Header.Set("Origin", "https://claude.ai")
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("preflight returned a %d-byte body, want none", w.Body.Len())
	}
	if m := w.Header().Get("Access-Control-Allow-Methods"); m == "" {
		t.Error("preflight missing Access-Control-Allow-Methods")
	}
}

// The Etag is what makes conditional requests work; without a 304 every client
// re-downloads the icon on each revalidation.
func TestIconConditionalRequest(t *testing.T) {
	h := oauthHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, iconPath, nil))
	etag := first.Header().Get("Etag")
	if etag == "" {
		t.Fatal("no Etag on the icon response")
	}

	second := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, iconPath, nil)
	r.Header.Set("If-None-Match", etag)
	h.ServeHTTP(second, r)

	if second.Code != http.StatusNotModified {
		t.Errorf("If-None-Match got %d, want 304", second.Code)
	}
	if second.Body.Len() != 0 {
		t.Errorf("304 carried a %d-byte body", second.Body.Len())
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
