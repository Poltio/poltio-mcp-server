package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ProtectedResourceMetadata is the RFC 9728 document served at
// /.well-known/oauth-protected-resource.
type ProtectedResourceMetadata struct {
	Resource              string   `json:"resource"`
	AuthorizationServers  []string `json:"authorization_servers"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

// AuthorizationServerMetadata is the RFC 8414 document served at
// /.well-known/oauth-authorization-server.
type AuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// MetadataHandlers returns handlers for both well-known metadata endpoints.
// serverURL must be the public HTTPS base URL (e.g. "https://mcp.example.com").
func MetadataHandlers(serverURL string) (prmHandler, asmHandler http.HandlerFunc) {
	prm := ProtectedResourceMetadata{
		Resource:              serverURL,
		AuthorizationServers:  []string{serverURL},
		BearerMethodsSupported: []string{"header"},
	}
	asm := AuthorizationServerMetadata{
		Issuer:                serverURL,
		AuthorizationEndpoint: serverURL + "/authorize",
		TokenEndpoint:         serverURL + "/token",
		RegistrationEndpoint:  serverURL + "/register",
		RevocationEndpoint:    serverURL + "/revoke",
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
	}

	prmJSON, _ := json.Marshal(prm)
	asmJSON, _ := json.Marshal(asm)

	prmHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(prmJSON) //nolint:errcheck
	}
	asmHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(asmJSON) //nolint:errcheck
	}
	return prmHandler, asmHandler
}

// ValidateServerURL returns an error if serverURL is empty or does not start
// with "https://". Pass BRIDGE_DEV_MODE=true to permit "http://" in local dev.
func ValidateServerURL(serverURL string) error {
	if serverURL == "" {
		return fmt.Errorf("oauth: SERVER_URL env var must be set")
	}
	if !strings.HasPrefix(serverURL, "https://") {
		return fmt.Errorf("oauth: SERVER_URL must start with https:// (got %q); set BRIDGE_DEV_MODE=true to allow http:// in local development", serverURL)
	}
	return nil
}

// UnauthorizedMCPMiddleware wraps an http.Handler: requests missing a Bearer
// token receive a 401 with the WWW-Authenticate header pointing at the PRM
// document. The wrapped handler is responsible for full token validation.
func UnauthorizedMCPMiddleware(serverURL string, next http.Handler) http.Handler {
	wwwAuth := `Bearer resource_metadata="` + serverURL + `/.well-known/oauth-protected-resource"`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("WWW-Authenticate", wwwAuth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
