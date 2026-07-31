package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Poltio/poltio-mcp-server/client"
)

// resourceMetadataPath is where an MCP client looks for the OAuth protected
// resource metadata (RFC 9728) after receiving a 401 from /mcp.
const resourceMetadataPath = "/.well-known/oauth-protected-resource"

// tokenCtxKey carries the caller's bearer token from the HTTP request into the
// tool handlers. Absent in stdio mode, where the env token is used instead.
type tokenCtxKey struct{}

// publicURL is this server's own canonical URL, used as the OAuth resource
// identifier. Override when running somewhere other than production.
func publicURL() string {
	if v := os.Getenv("MCP_PUBLIC_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://mcp.poltio.com"
}

// bearerToken returns the token from an Authorization header, or "" if absent.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

// withBearer lifts the caller's token into the context so tool handlers can
// build a client for that specific user.
func withBearer(ctx context.Context, r *http.Request) context.Context {
	if tok := bearerToken(r); tok != "" {
		return context.WithValue(ctx, tokenCtxKey{}, tok)
	}
	return ctx
}

// baseClient serves stdio mode, where the token comes from POLTIO_API_TOKEN and
// there is exactly one user.
var baseClient *client.PoltioClient

// clients memoises one client per bearer token. A client owns the active
// organization, so sharing one across users would let a switch_organization
// call leak into everyone else's requests.
//
// ponytail: unbounded map, one small struct per distinct token. Add TTL
// eviction if a long-running server accumulates enough tokens to matter.
var (
	clientsMu sync.Mutex
	clients   = map[string]*client.PoltioClient{}
)

// clientForCtx returns the client for the caller's token, creating and
// activating an organization for it on first use.
func clientForCtx(ctx context.Context) (*client.PoltioClient, error) {
	tok, _ := ctx.Value(tokenCtxKey{}).(string)
	if tok == "" {
		return baseClient, nil
	}

	clientsMu.Lock()
	c, ok := clients[tok]
	clientsMu.Unlock()
	if ok {
		return c, nil
	}

	// Deliberately not holding the lock across the network call. Two concurrent
	// first calls for the same token may both build a client; either is valid.
	c = client.New(tok)
	if err := activateFirstOrg(c); err != nil {
		// Surface the real reason (bad token, no organization) instead of
		// caching a client whose every call would fail confusingly. The
		// client.ErrUnauthorized text tells stdio users to edit an env var,
		// which an OAuth caller cannot do.
		if errors.Is(err, client.ErrUnauthorized) {
			return nil, errors.New("your Poltio authorization is invalid or has expired — reconnect the Poltio connector to sign in again")
		}
		return nil, err
	}

	clientsMu.Lock()
	clients[tok] = c
	clientsMu.Unlock()
	return c, nil
}

// toolHandler is an alias, not a defined type: the tools package returns this
// signature literally, and generic inference only unifies identical types.
type toolHandler = func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

// withAuth adapts a tool constructor that wants a client into a handler that
// resolves the caller's client per request. T is the narrow client interface
// each tools package function accepts, inferred from the constructor.
func withAuth[T any](newHandler func(T) toolHandler) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		c, err := clientForCtx(ctx)
		if err != nil {
			return nil, err
		}
		return newHandler(any(c).(T))(ctx, req)
	}
}

// oauthHandler wraps the MCP endpoint with the two pieces an MCP client needs
// to discover how to authenticate: protected resource metadata, and a 401
// pointing at it.
func oauthHandler(mcpServer http.Handler) http.Handler {
	metadataURL := publicURL() + resourceMetadataPath

	mux := http.NewServeMux()

	mux.HandleFunc(resourceMetadataPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resource":              publicURL(),
			"authorization_servers": []string{client.BaseURL()},
		})
	})

	// Both patterns: ServeMux treats "/mcp" and "/mcp/" as distinct, and a
	// client that normalises the trailing slash would otherwise get a 404
	// instead of the authentication challenge.
	guarded := requireBearer(mcpServer, metadataURL)
	mux.Handle("/mcp", guarded)
	mux.Handle("/mcp/", guarded)

	return mux
}

// requireBearer rejects unauthenticated MCP requests with the challenge that
// starts the OAuth flow. Preflight requests pass through to the CORS handling
// already built into the MCP server.
func requireBearer(next http.Handler, metadataURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || bearerToken(r) != "" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+metadataURL+`"`)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "unauthorized",
			"error_description": "Authenticate with Poltio to use this MCP server.",
		})
	})
}
