package oauth

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Poltio/poltio-mcp-server/store"
)

// claudeAICallback is the canonical Claude.ai redirect URI per Anthropic's connector spec.
const claudeAICallback = "https://claude.ai/api/mcp/auth_callback"

// isAllowedRedirectURI returns true if uri is in the closed allowlist.
// Comparison is byte-exact after percent-decoding (OAuth 2.1 §2.1).
// Allowed: claude.ai callback, RFC 8252 loopback (localhost/127.0.0.1, any port, path /callback).
func isAllowedRedirectURI(raw string) bool {
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		return false
	}
	if decoded == claudeAICallback {
		return true
	}
	// RFC 8252 loopback: http://localhost[:<port>]/callback or http://127.0.0.1[:<port>]/callback
	u, err := url.Parse(decoded)
	if err != nil || u.Scheme != "http" {
		return false
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host // no port
	}
	if host != "localhost" && host != "127.0.0.1" {
		return false
	}
	return u.Path == "/callback"
}

// DCRRequest is the RFC 7591 client registration request body.
type DCRRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
}

// DCRResponse is the RFC 7591 client registration response.
type DCRResponse struct {
	ClientID string `json:"client_id"`
}

// rateLimiter tracks per-IP request counts within a sliding window.
type rateLimiter struct {
	mu       sync.Mutex
	counts   map[string][]time.Time
	max      int
	window   time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		counts: make(map[string][]time.Time),
		max:    max,
		window: window,
	}
}

// allow returns true if the IP is under the rate limit, false if exceeded.
func (r *rateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-r.window)
	times := r.counts[ip]
	// Prune old entries
	filtered := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= r.max {
		r.counts[ip] = filtered
		return false
	}
	r.counts[ip] = append(filtered, now)
	return true
}

// RegisterConfig holds configuration for the /register handler.
type RegisterConfig struct {
	ClientTTL   time.Duration // how long a registered client_id lives; default 24h
	RateMax     int           // max registrations per IP per window; default 20
	RateWindow  time.Duration // rate-limit window; default 1 minute
}

func (cfg *RegisterConfig) withDefaults() RegisterConfig {
	c := *cfg
	if c.ClientTTL == 0 {
		c.ClientTTL = 24 * time.Hour
	}
	if c.RateMax == 0 {
		c.RateMax = 20
	}
	if c.RateWindow == 0 {
		c.RateWindow = time.Minute
	}
	return c
}

// RegisterHandler returns an http.HandlerFunc for POST /register (RFC 7591 DCR).
func RegisterHandler(db *store.Store, cfg RegisterConfig) http.HandlerFunc {
	cfg = cfg.withDefaults()
	limiter := newRateLimiter(cfg.RateMax, cfg.RateWindow)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			http.Error(w, "unsupported media type — request body must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		// Rate limit by IP
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}
		if !limiter.allow(ip) {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		var req DCRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "invalid_request", "request body is not valid JSON", http.StatusBadRequest)
			return
		}
		if len(req.RedirectURIs) == 0 {
			writeJSONError(w, "invalid_redirect_uri", "redirect_uris is required", http.StatusBadRequest)
			return
		}
		for _, u := range req.RedirectURIs {
			if u == "" || !isAllowedRedirectURI(u) {
				writeJSONError(w, "invalid_redirect_uri", fmt.Sprintf("redirect_uri %q is not in the allowed set", u), http.StatusBadRequest)
				return
			}
		}

		clientID := uuid.New().String()
		now := time.Now().UTC()
		client := &store.OAuthClient{
			ClientID:     clientID,
			RedirectURIs: req.RedirectURIs,
			CreatedAt:    now,
			ExpiresAt:    now.Add(cfg.ClientTTL),
		}
		if err := db.CreateClient(client); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(DCRResponse{ClientID: clientID}) //nolint:errcheck
	}
}

func writeJSONError(w http.ResponseWriter, errCode, description string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"error":             errCode,
		"error_description": description,
	})
}
