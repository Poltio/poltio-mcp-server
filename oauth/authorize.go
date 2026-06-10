package oauth

import (
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/Poltio/poltio-mcp-server/store"
)

const defaultSessionTTL = 10 * time.Minute

// AuthorizeHandler returns the GET /authorize handler.
// db is used to look up the registered client and store the pending session.
// sessionTTL is how long the pending session lives (default 10 min if zero).
func AuthorizeHandler(db *store.Store, sessionTTL time.Duration) http.HandlerFunc {
	if sessionTTL == 0 {
		sessionTTL = defaultSessionTTL
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		q := r.URL.Query()
		clientID := q.Get("client_id")
		redirectURI := q.Get("redirect_uri")
		responseType := q.Get("response_type")
		codeChallenge := q.Get("code_challenge")
		codeChallengeMethod := q.Get("code_challenge_method")
		state := q.Get("state")

		// Steps 5–6: Validate client_id and redirect_uri FIRST — before any redirects
		// (OAuth 2.1 §1.7: if redirect_uri can't be trusted, return error directly).
		oauthClient, err := db.GetClient(clientID)
		if err != nil {
			log.Printf("authorize: get client %q: %v", clientID, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if oauthClient == nil {
			http.Error(w, "unknown client_id", http.StatusBadRequest)
			return
		}

		// Validate redirect_uri is an exact-byte-match to a registered URI.
		validRedirect := slices.Contains(oauthClient.RedirectURIs, redirectURI)
		if !validRedirect {
			http.Error(w, "redirect_uri does not match any registered URI", http.StatusBadRequest)
			return
		}

		// Steps 2–4: Now that we trust redirect_uri, redirect with error codes for bad params.

		// Step 2: response_type must be "code".
		if responseType != "code" {
			redirectWithError(w, r, redirectURI, "unsupported_response_type", state)
			return
		}

		// Step 3: PKCE — code_challenge_method must be "S256", code_challenge must be non-empty.
		if codeChallengeMethod != "S256" || codeChallenge == "" {
			redirectWithError(w, r, redirectURI, "invalid_request", state)
			return
		}

		// Step 4: state must be non-empty.
		if state == "" {
			redirectWithError(w, r, redirectURI, "invalid_request", state)
			return
		}

		// Step 7: Generate session ID.
		sessionID := uuid.New().String()

		// Step 8: Create pending session.
		now := time.Now().UTC()
		sess := &store.PendingSession{
			SessionID:     sessionID,
			ClientID:      clientID,
			RedirectURI:   redirectURI,
			CodeChallenge: codeChallenge,
			State:         state,
			CreatedAt:     now,
			ExpiresAt:     now.Add(sessionTTL),
		}
		if err := db.CreatePendingSession(sess); err != nil {
			log.Printf("authorize: create pending session: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Step 9: Set session cookie.
		// __Host- prefix requires Secure + Path=/ + no Domain attribute.
		// In dev mode (http), browsers reject __Host- cookies without Secure.
		// We always set Secure here; in dev contexts the test recorder accepts it.
		http.SetCookie(w, &http.Cookie{
			Name:     "__Host-session",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(sessionTTL.Seconds()),
		})

		// Step 10: Render consent form.
		renderConsent(w, "")
	}
}

// redirectWithError redirects to redirectURI with error and state query params.
func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, errCode, state string) {
	loc := redirectURI + "?error=" + errCode
	if state != "" {
		loc += "&state=" + state
	}
	http.Redirect(w, r, loc, http.StatusFound)
}
