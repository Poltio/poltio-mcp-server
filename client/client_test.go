package client_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/Poltio/poltio-mcp-server/client"
)

func TestGet_SetsAuthHeaders(t *testing.T) {
	var gotAuth, gotOrgID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotOrgID = r.Header.Get("Organization-Id")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok123", "42", srv.URL)
	body, err := c.Get("/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("Authorization: want %q, got %q", "Bearer tok123", gotAuth)
	}
	if gotOrgID != "42" {
		t.Errorf("Organization-Id: want %q, got %q", "42", gotOrgID)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body: want %q, got %q", `{"ok":true}`, string(body))
	}
}

func TestGet_WithQueryParams(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok", "1", srv.URL)
	q := url.Values{}
	q.Set("page", "2")
	q.Set("per_page", "5")
	_, err := c.Get("/test", q)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery.Get("page") != "2" {
		t.Errorf("page: want 2, got %q", gotQuery.Get("page"))
	}
	if gotQuery.Get("per_page") != "5" {
		t.Errorf("per_page: want 5, got %q", gotQuery.Get("per_page"))
	}
}

func TestGet_NonOKStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := client.NewForTest("bad", "1", srv.URL)
	_, err := c.Get("/test", nil)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

func TestPost_SendsJSONBody(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"public_id":"abc"}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok", "1", srv.URL)
	payload := map[string]any{"title": "My Poll", "type": "poll"}
	_, err := c.Post("/test", payload)
	if err != nil {
		t.Fatal(err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", gotContentType)
	}
	if !strings.Contains(string(gotBody), "My Poll") {
		t.Errorf("body should contain 'My Poll', got %q", string(gotBody))
	}
}

func TestPost_NilBody_NoContentType(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok", "1", srv.URL)
	_, err := c.Post("/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type should be empty for nil body, got %q", gotContentType)
	}
}

// Test 1: 401 returns ErrPoltioUnauthorized
func TestDo_401_ReturnsErrPoltioUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := client.NewForTest("bad-token", "1", srv.URL)
	_, err := c.Get("/test", nil)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !errors.Is(err, client.ErrPoltioUnauthorized) {
		t.Errorf("expected ErrPoltioUnauthorized, got %v", err)
	}
}

// Test 1b: 403 also returns ErrPoltioUnauthorized
func TestDo_403_ReturnsErrPoltioUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := client.NewForTest("bad-token", "1", srv.URL)
	_, err := c.Get("/test", nil)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !errors.Is(err, client.ErrPoltioUnauthorized) {
		t.Errorf("expected ErrPoltioUnauthorized, got %v", err)
	}
}

// Test 2: 503 returns ErrPoltioUnavailable
func TestDo_503_ReturnsErrPoltioUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"message":"service unavailable"}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok", "1", srv.URL)
	_, err := c.Get("/test", nil)
	if err == nil {
		t.Fatal("expected error for 503, got nil")
	}
	if !errors.Is(err, client.ErrPoltioUnavailable) {
		t.Errorf("expected ErrPoltioUnavailable, got %v", err)
	}
}

// Test 3: connection refused returns ErrPoltioUnavailable
func TestDo_ConnectionRefused_ReturnsErrPoltioUnavailable(t *testing.T) {
	// Start a server, capture URL, then close it before making a request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srvURL := srv.URL
	srv.Close()

	c := client.NewForTest("tok", "1", srvURL)
	_, err := c.Get("/test", nil)
	if err == nil {
		t.Fatal("expected error for connection refused, got nil")
	}
	if !errors.Is(err, client.ErrPoltioUnavailable) {
		t.Errorf("expected ErrPoltioUnavailable, got %v", err)
	}
}

// Test 4: WithContext + FromContext round-trip
func TestWithContext_FromContext_RoundTrip(t *testing.T) {
	c := client.NewForTest("tok", "org1", "http://example.com")
	ctx := client.WithContext(context.Background(), c)
	got, err := client.FromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != c {
		t.Error("FromContext returned a different client than what was stored")
	}
}

// Test 5: FromContext on empty context returns ErrBridgeDecryptFailure
func TestFromContext_EmptyContext_ReturnsErrBridgeDecryptFailure(t *testing.T) {
	_, err := client.FromContext(context.Background())
	if err == nil {
		t.Fatal("expected error for empty context, got nil")
	}
	if !errors.Is(err, client.ErrBridgeDecryptFailure) {
		t.Errorf("expected ErrBridgeDecryptFailure, got %v", err)
	}
}

// Test 6: Concurrency — 2 clients with different tokens/orgIDs, 100 concurrent requests each.
// Verifies no cross-contamination via the context-threading mechanism.
func TestConcurrency_NoClientCrossContamination(t *testing.T) {
	// Two separate test servers recording the Authorization and Organization-Id headers they receive.
	type capture struct {
		mu     sync.Mutex
		auths  []string
		orgIDs []string
	}

	newCapturingServer := func(c *capture) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			orgID := r.Header.Get("Organization-Id")
			c.mu.Lock()
			c.auths = append(c.auths, auth)
			c.orgIDs = append(c.orgIDs, orgID)
			c.mu.Unlock()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}))
	}

	capA := &capture{}
	capB := &capture{}
	srvA := newCapturingServer(capA)
	srvB := newCapturingServer(capB)
	defer srvA.Close()
	defer srvB.Close()

	clientA := client.NewForRequest("tokenA", "orgA", srvA.URL)
	clientB := client.NewForRequest("tokenB", "orgB", srvB.URL)

	ctxA := client.WithContext(context.Background(), clientA)
	ctxB := client.WithContext(context.Background(), clientB)

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n * 2)

	for range n {
		go func() {
			defer wg.Done()
			c, err := client.FromContext(ctxA)
			if err != nil {
				t.Errorf("FromContext(ctxA): %v", err)
				return
			}
			if _, err := c.Get("/test", nil); err != nil {
				t.Errorf("clientA.Get: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			c, err := client.FromContext(ctxB)
			if err != nil {
				t.Errorf("FromContext(ctxB): %v", err)
				return
			}
			if _, err := c.Get("/test", nil); err != nil {
				t.Errorf("clientB.Get: %v", err)
			}
		}()
	}
	wg.Wait()

	// Verify server A only received tokenA requests.
	capA.mu.Lock()
	defer capA.mu.Unlock()
	if len(capA.auths) != n {
		t.Errorf("server A: expected %d requests, got %d", n, len(capA.auths))
	}
	for i, auth := range capA.auths {
		if auth != "Bearer tokenA" {
			t.Errorf("server A request %d: wrong auth %q", i, auth)
		}
	}
	for i, orgID := range capA.orgIDs {
		if orgID != "orgA" {
			t.Errorf("server A request %d: wrong orgID %q", i, orgID)
		}
	}

	// Verify server B only received tokenB requests.
	capB.mu.Lock()
	defer capB.mu.Unlock()
	if len(capB.auths) != n {
		t.Errorf("server B: expected %d requests, got %d", n, len(capB.auths))
	}
	for i, auth := range capB.auths {
		if auth != "Bearer tokenB" {
			t.Errorf("server B request %d: wrong auth %q", i, auth)
		}
	}
	for i, orgID := range capB.orgIDs {
		if orgID != "orgB" {
			t.Errorf("server B request %d: wrong orgID %q", i, orgID)
		}
	}
}
