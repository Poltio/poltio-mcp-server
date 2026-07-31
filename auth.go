package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Poltio/poltio-mcp-server/client"
)

// resourceMetadataPath is where an MCP client looks for the OAuth protected
// resource metadata (RFC 9728) after receiving a 401 from /mcp.
const resourceMetadataPath = "/.well-known/oauth-protected-resource"

// iconPath serves the connector icon advertised in serverInfo. Embedded so the
// binary has no runtime dependency on poltio.com's asset layout.
const iconPath = "/icon.png"

//go:embed icon.png
var iconPNG []byte

// iconETag is derived from the bytes, not from a timestamp: the embedded icon
// is immutable for the life of the binary, so a content hash stays stable
// across restarts and is identical on every replica. A startup time would
// invalidate every cache on each deploy and disagree between pods, making a
// client see "modified" when nothing changed.
var iconETag = func() string {
	sum := sha256.Sum256(iconPNG)
	return `"` + hex.EncodeToString(sum[:16]) + `"`
}()

// iconURL is the absolute URL clients fetch the icon from.
func iconURL() string { return publicURL() + iconPath }

// readMethodsOnly limits an endpoint to reads and nothing more. Used for the
// health endpoints: they are probed by the load balancer and kubelet, never by
// a browser, so they have no reason to advertise CORS or answer a preflight.
func readMethodsOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

// readOnly is readMethodsOnly plus the CORS contract, for the endpoints a
// browser-based client genuinely fetches cross-origin (discovery metadata and
// the icon). Without it those return a body to POST and DELETE, and answer a
// preflight with the payload itself.
func readOnly(next http.HandlerFunc) http.HandlerFunc {
	const allow = "GET, HEAD, OPTIONS"
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		switch r.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Methods", allow)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet, http.MethodHead:
			next(w, r)
		default:
			w.Header().Set("Allow", allow)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

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
	c, err := clientForToken(tok)
	if err != nil {
		if errors.Is(err, client.ErrUnauthorized) {
			return nil, errReauthNeeded
		}
		return nil, err
	}
	return c, nil
}

// errReauthNeeded replaces client.ErrUnauthorized for callers that authenticated
// through the connector flow. The sentinel's own text tells the user to edit
// POLTIO_API_TOKEN and their MCP client settings, which is right for stdio and
// useless over OAuth — there is no env var to fix, only a connector to reconnect.
var errReauthNeeded = errors.New("your Poltio authorization is invalid or has expired — reconnect the Poltio connector to sign in again")

// clientForToken resolves and caches the client for one bearer token. The first
// call for a token hits the API to select an organization, which doubles as
// proof the token is good — errors wrap client.ErrUnauthorized so callers can
// tell a rejected token from an unreachable API.
func clientForToken(tok string) (*client.PoltioClient, error) {
	clientsMu.Lock()
	c, ok := clients[tok]
	clientsMu.Unlock()
	if ok {
		return c, nil
	}

	// Deliberately not holding the lock across the network call, so a slow
	// activation cannot stall every other token's requests.
	c = client.New(tok)
	if err := activateFirstOrg(c); err != nil {
		// Surface the real reason (bad token, no organization) instead of
		// caching a client whose every call would fail confusingly.
		return nil, err
	}

	// Check again under the lock: another request for this token may have
	// finished activating while this one was on the network. The first client
	// stored wins, because a caller may already be holding it and mutating its
	// active organization — overwriting it would discard that silently.
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if existing, ok := clients[tok]; ok {
		return existing, nil
	}
	clients[tok] = c
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

		res, err := newHandler(any(c).(T))(ctx, req)
		if tok, _ := ctx.Value(tokenCtxKey{}).(string); tok != "" && errors.Is(err, client.ErrUnauthorized) {
			// The token passed validation when it was cached and has since
			// expired or been revoked — which every token does, since they last
			// two weeks. Drop it so the next request misses the cache,
			// re-validates, and gets the 401 challenge that restarts the OAuth
			// flow. Without this the connector fails every call until the pod
			// restarts, with nothing telling the client to re-authenticate.
			evictToken(tok)
			// Same substitution clientForCtx makes: this caller has no env var
			// to fix. Only rewritten in OAuth mode — a stdio user genuinely does
			// need the sentinel's advice.
			return nil, errReauthNeeded
		}
		return res, err
	}
}

// evictToken forgets the cached client for a token.
func evictToken(tok string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, tok)
}

// oauthHandler wraps the MCP endpoint with the two pieces an MCP client needs
// to discover how to authenticate: protected resource metadata, and a 401
// pointing at it.
func oauthHandler(mcpServer http.Handler) http.Handler {
	metadataURL := publicURL() + resourceMetadataPath

	mux := http.NewServeMux()

	metadata := readOnly(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			// The resource identifier is the MCP endpoint itself, not the origin
			// — matching the shape Strava's working connector publishes.
			"resource":                 publicURL() + "/mcp",
			"authorization_servers":    []string{client.BaseURL()},
			"bearer_methods_supported": []string{"header"},
			"resource_name":            "Poltio",
		})
	})

	// Load balancer health checks. Before this server had a router every path
	// answered 200, so the Google Cloud health check passed by accident; adding
	// the mux made "/" a 404 and the backend went unhealthy ("no healthy
	// upstream"), even though the pods were Ready on their tcpSocket probe.
	//
	// "/{$}" matches the root and nothing else — a bare "/" pattern is a subtree
	// in ServeMux and would put us back to answering 200 for every unknown path,
	// which is what hid this problem in the first place.
	health := readMethodsOnly(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/{$}", health)
	mux.HandleFunc("/healthz", health)

	mux.HandleFunc(iconPath, readOnly(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Etag", iconETag)
		// ServeContent sets Content-Type and handles Range plus the conditional
		// requests (If-None-Match → 304) that the Etag above makes possible.
		// modtime stays zero: the Etag is the validator, and a fabricated
		// timestamp would only add a second, less accurate one.
		http.ServeContent(w, r, "icon.png", time.Time{}, bytes.NewReader(iconPNG))
	}))

	mux.HandleFunc(resourceMetadataPath, metadata)
	// RFC 9728 locates the metadata for a resource with a path by appending that
	// path to the well-known prefix. Clients that derive the URL themselves look
	// here instead of at the resource_metadata we hand them in the challenge.
	// Both slash forms, for the same reason /mcp and /mcp/ are both registered —
	// a client that normalises the resource URL derives the slashed variant.
	// The {$} anchors the pattern to that exact path: without it ServeMux would
	// treat it as a subtree and answer for /mcp/anything too.
	mux.HandleFunc(resourceMetadataPath+"/mcp", metadata)
	mux.HandleFunc(resourceMetadataPath+"/mcp/{$}", metadata)

	// Both patterns: ServeMux treats "/mcp" and "/mcp/" as distinct, and a
	// client that normalises the trailing slash would otherwise get a 404
	// instead of the authentication challenge. {$} anchors the slashed form to
	// that exact path — as a bare subtree it would also answer for /mcp/anything,
	// which is not an endpoint this server serves.
	guarded := requireBearer(mcpServer, metadataURL)
	mux.Handle("/mcp", guarded)
	mux.Handle("/mcp/{$}", guarded)

	return mux
}

// requireBearer rejects MCP requests whose token is missing or rejected by the
// Poltio API, with the challenge that starts the OAuth flow. Preflight requests
// pass through to the CORS handling already built into the MCP server.
//
// Validating here rather than in the tool handlers is what lets an expired
// token produce a 401: a client that only ever sees a successful initialize has
// no reason to re-run the OAuth flow, and would instead fail every tool call
// with an error it cannot act on.
func requireBearer(next http.Handler, metadataURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		tok := bearerToken(r)
		if tok == "" {
			// No error code: RFC 6750 says not to send one when the request
			// carried no credentials at all.
			challenge(w, metadataURL, "", "Authenticate with Poltio to use this MCP server.")
			return
		}

		// Resolves from the per-token cache after the first request, so this
		// costs one API call per new token rather than one per request. A token
		// that dies mid-session passes here until a tool call fails and evicts
		// it (see withAuth); the request after that re-validates and 401s.
		if _, err := clientForToken(tok); err != nil {
			if errors.Is(err, client.ErrUnauthorized) {
				challenge(w, metadataURL, "invalid_token", "Your Poltio authorization is invalid or has expired.")
				return
			}
			// The token is good but the account cannot use the server. Permanent
			// until the user acts, so 403 — a 503 invites retries that cannot
			// succeed, and can make health tooling read this as an outage.
			if errors.Is(err, errNoOrganization) {
				writeErr(w, http.StatusForbidden, "forbidden", err.Error())
				return
			}
			// Anything else is upstream: unreachable API, unparseable response.
			// Retryable, so 503 — but the detail stays in the log rather than
			// going to the caller, since it can carry internal URLs and paths.
			log.Printf("token validation failed: %v", err)
			writeErr(w, http.StatusServiceUnavailable, "server_error",
				"Poltio is temporarily unavailable. Try again shortly.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeErr sends a JSON error body. Descriptions are written for the end user;
// upstream error text belongs in the log, not in a response.
func writeErr(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

// challenge writes the 401 that tells an MCP client where to authenticate.
// errCode is the RFC 6750 code, omitted when the request carried no token.
func challenge(w http.ResponseWriter, metadataURL, errCode, description string) {
	auth := `Bearer `
	if errCode != "" {
		auth += `error="` + errCode + `", `
	}
	w.Header().Set("WWW-Authenticate", auth+`resource_metadata="`+metadataURL+`"`)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             "unauthorized",
		"error_description": description,
	})
}
