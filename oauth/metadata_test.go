package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestPRMHandler(t *testing.T) {
	prm, _ := MetadataHandlers("https://mcp.example.com")
	w := httptest.NewRecorder()
	prm(w, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil))
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: %q", ct)
	}
	var doc ProtectedResourceMetadata
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.Resource != "https://mcp.example.com" {
		t.Errorf("resource: %q", doc.Resource)
	}
}

func TestASMHandler(t *testing.T) {
	_, asm := MetadataHandlers("https://mcp.example.com")
	w := httptest.NewRecorder()
	asm(w, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	var doc AuthorizationServerMetadata
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !contains(doc.CodeChallengeMethodsSupported, "S256") {
		t.Errorf("S256 not in code_challenge_methods_supported: %v", doc.CodeChallengeMethodsSupported)
	}
	if doc.RegistrationEndpoint != "https://mcp.example.com/register" {
		t.Errorf("registration_endpoint: %q", doc.RegistrationEndpoint)
	}
}

func TestUnauthedMCPReturns401(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := UnauthorizedMCPMiddleware("https://mcp.example.com", inner)

	// no Authorization header → 401
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", w.Code)
	}
	wwwAuth := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(wwwAuth, "resource_metadata=") {
		t.Errorf("WWW-Authenticate missing resource_metadata: %q", wwwAuth)
	}

	// with Authorization header → passes through to inner
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer token123")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status with auth: got %d, want 200", w.Code)
	}
}

func TestValidateServerURL(t *testing.T) {
	if err := ValidateServerURL("https://mcp.example.com"); err != nil {
		t.Errorf("https should pass: %v", err)
	}
	if err := ValidateServerURL("http://mcp.example.com"); err == nil {
		t.Error("http should fail in non-dev mode")
	}
	if err := ValidateServerURL(""); err == nil {
		t.Error("empty should fail")
	}
}

func contains(ss []string, target string) bool {
	return slices.Contains(ss, target)
}
