package oauth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Poltio/poltio-mcp-server/store"
)

func openTestDB(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestRegisterValidClaudeCallback(t *testing.T) {
	h := RegisterHandler(openTestDB(t), RegisterConfig{})
	body := `{"redirect_uris":["https://claude.ai/api/mcp/auth_callback"]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	h(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var resp DCRResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ClientID == "" {
		t.Error("expected non-empty client_id")
	}
}

func TestRegisterValidLoopbackCallback(t *testing.T) {
	h := RegisterHandler(openTestDB(t), RegisterConfig{})
	for _, uri := range []string{
		"http://localhost/callback",
		"http://localhost:8080/callback",
		"http://127.0.0.1/callback",
		"http://127.0.0.1:9000/callback",
	} {
		body, _ := json.Marshal(map[string]any{"redirect_uris": []string{uri}})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		h(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("uri %q: status %d", uri, w.Code)
		}
	}
}

func TestRegisterRejectsUnknownRedirectURI(t *testing.T) {
	h := RegisterHandler(openTestDB(t), RegisterConfig{})
	for _, uri := range []string{
		"http://attacker.com/cb",
		"https://evil.example.com/callback",
		"",
		"javascript:alert(1)",
	} {
		body, _ := json.Marshal(map[string]any{"redirect_uris": []string{uri}})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		h(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("uri %q: expected 400, got %d", uri, w.Code)
		}
	}
}

func TestRegisterRejectsFormEncoded(t *testing.T) {
	h := RegisterHandler(openTestDB(t), RegisterConfig{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("redirect_uris=https://claude.ai/api/mcp/auth_callback"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("form-encoded: expected 415, got %d", w.Code)
	}
}

func TestRegisterRateLimit(t *testing.T) {
	h := RegisterHandler(openTestDB(t), RegisterConfig{
		RateMax:    3,
		RateWindow: time.Minute,
	})
	body := `{"redirect_uris":["https://claude.ai/api/mcp/auth_callback"]}`
	var lastCode int
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "1.2.3.4:9999"
		h(w, req)
		lastCode = w.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after rate limit exceeded, got %d", lastCode)
	}
}

func TestIsAllowedRedirectURI(t *testing.T) {
	allowed := []string{
		"https://claude.ai/api/mcp/auth_callback",
		"http://localhost/callback",
		"http://localhost:8080/callback",
		"http://127.0.0.1/callback",
		"http://127.0.0.1:9000/callback",
	}
	for _, u := range allowed {
		if !isAllowedRedirectURI(u) {
			t.Errorf("expected allowed: %q", u)
		}
	}
	rejected := []string{
		"http://attacker.com/callback",
		"https://localhost/callback", // https loopback not in RFC 8252 pattern
		"http://localhost/other",
		"",
	}
	for _, u := range rejected {
		if isAllowedRedirectURI(u) {
			t.Errorf("expected rejected: %q", u)
		}
	}
}
