package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/store"
)

//go:embed templates/consent.html
var consentFS embed.FS

// consentTmpl is parsed once at init time.
var consentTmpl = template.Must(
	template.ParseFS(consentFS, "templates/consent.html"),
)

// consentData is the template data struct.
type consentData struct {
	ErrMsg string
}

// renderConsent writes the consent HTML to w.
func renderConsent(w http.ResponseWriter, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := consentTmpl.Execute(w, consentData{ErrMsg: errMsg}); err != nil {
		log.Printf("consent: template execute error: %v", err)
	}
}

// poltioProfile is the shape of /platform/account/profile.
type poltioProfile struct {
	ID            int `json:"id"`
	Organizations []struct {
		ID int `json:"id"`
	} `json:"organizations"`
}

const poltioDefaultBaseURL = "https://api-stage.poltio.com"

// fetchPoltioProfile calls /platform/account/profile with the given token.
// Returns client.ErrPoltioUnauthorized on 401/403, client.ErrPoltioUnavailable on 5xx/transport.
func fetchPoltioProfile(token, baseURL string) (*poltioProfile, error) {
	if baseURL == "" {
		baseURL = poltioDefaultBaseURL
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/platform/account/profile", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", client.ErrPoltioUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", client.ErrPoltioUnavailable, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, client.ErrPoltioUnauthorized
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: status %d", client.ErrPoltioUnavailable, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poltio: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", client.ErrPoltioUnavailable, err)
	}

	var profile poltioProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("poltio: parse profile: %w", err)
	}
	return &profile, nil
}

// ConsentHandler returns the POST /consent handler.
// db is used to read the pending session and create grant/auth-code records.
// serverURL is used for the reconnect URL in error messages.
// key is the 32-byte AES-256-GCM encryption key from store.KeyFromEnv().
// sessionTTL should match the value passed to AuthorizeHandler.
// poltioBaseURL: if empty, uses the default Poltio API URL (pass "" in production, custom URL in tests).
func ConsentHandler(db *store.Store, serverURL string, key []byte, sessionTTL time.Duration, poltioBaseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 1. Read __Host-session cookie.
		cookie, err := r.Cookie("__Host-session")
		if err != nil {
			http.Error(w, "session cookie required", http.StatusBadRequest)
			return
		}

		// 2. Look up pending session.
		sess, err := db.GetPendingSession(cookie.Value)
		if err != nil {
			log.Printf("consent: get pending session: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if sess == nil {
			http.Error(w, "session expired or not found", http.StatusBadRequest)
			return
		}

		// 3. Parse form body.
		if err := r.ParseForm(); err != nil {
			renderConsent(w, "Could not parse form submission.")
			return
		}

		// 4. Read poltio_token (never log the value).
		poltioToken := r.FormValue("poltio_token")
		if poltioToken == "" {
			renderConsent(w, "Poltio API token is required.")
			return
		}

		// 5. Validate token by fetching profile — retry on unavailable.
		var profile *poltioProfile
		{
			const maxAttempts = 3
			const retryDelay = 500 * time.Millisecond
			for attempt := range maxAttempts {
				profile, err = fetchPoltioProfile(poltioToken, poltioBaseURL)
				if err == nil {
					break
				}
				if errors.Is(err, client.ErrPoltioUnavailable) {
					if attempt < maxAttempts-1 {
						time.Sleep(retryDelay)
						continue
					}
					log.Printf("consent: poltio unavailable after %d attempts: %v", maxAttempts, err)
					renderConsent(w, "Poltio is temporarily unavailable, try again shortly.")
					return
				}
				if errors.Is(err, client.ErrPoltioUnauthorized) {
					renderConsent(w, "This token is invalid or expired — generate a new one from your Poltio account panel.")
					return
				}
				// Other errors treated as unavailable.
				log.Printf("consent: unexpected error fetching profile: %v", err)
				renderConsent(w, "Poltio is temporarily unavailable, try again shortly.")
				return
			}
		}

		// 6. Check zero orgs.
		if len(profile.Organizations) == 0 {
			renderConsent(w, "Your Poltio account has no organizations. Create one first.")
			return
		}

		// 7. Extract org and account IDs.
		poltioOrgID := strconv.Itoa(profile.Organizations[0].ID)
		poltioAccountID := strconv.Itoa(profile.ID)

		// 8. Encrypt poltio_token.
		grantID := uuid.New().String()
		enc, err := store.Encrypt([]byte(poltioToken), []byte(grantID), key)
		if err != nil {
			log.Printf("consent: encrypt token: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// 9. Create grant.
		now := time.Now().UTC()
		grant := &store.OAuthGrant{
			GrantID:         grantID,
			PoltioTokenEnc:  enc,
			PoltioOrgID:     poltioOrgID,
			PoltioAccountID: poltioAccountID,
			GrantState:      store.GrantStatePending,
			CreatedAt:       now,
		}
		if err := db.CreateGrant(grant); err != nil {
			log.Printf("consent: create grant: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// 10. Generate raw auth code (32 random bytes, hex-encoded).
		rawCodeBytes := make([]byte, 32)
		if _, err := rand.Read(rawCodeBytes); err != nil {
			log.Printf("consent: generate auth code: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		rawCodeHex := hex.EncodeToString(rawCodeBytes)

		// 11. Create auth code record (store SHA-256 of raw code only).
		h := sha256.Sum256([]byte(rawCodeHex))
		codeHash := hex.EncodeToString(h[:])
		authCode := &store.AuthCode{
			CodeHash:      codeHash,
			ClientID:      sess.ClientID,
			PKCEChallenge: sess.CodeChallenge,
			State:         sess.State,
			RedirectURI:   sess.RedirectURI,
			GrantID:       grantID,
			CreatedAt:     now,
			ExpiresAt:     now.Add(5 * time.Minute),
		}
		if err := db.CreateAuthCode(authCode); err != nil {
			log.Printf("consent: create auth code: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// 12. Delete pending session.
		if err := db.DeletePendingSession(sess.SessionID); err != nil {
			// Non-fatal: log and continue — the session will expire on its own.
			log.Printf("consent: delete pending session: %v", err)
		}

		// 13. Redirect with auth code and state.
		redirectURL := sess.RedirectURI + "?code=" + rawCodeHex + "&state=" + sess.State
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}
