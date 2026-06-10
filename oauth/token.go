package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Poltio/poltio-mcp-server/store"
)

// tokenResponse is the JSON body returned on a successful token exchange or refresh.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// writeTokenError writes an OAuth2 error JSON response.
func writeTokenError(w http.ResponseWriter, errCode string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": errCode}) //nolint:errcheck
}

// s256 computes base64url-no-pad(sha256(verifier)), implementing PKCE S256 method.
func s256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateTokenPair produces a random 32-byte hex access token and refresh token,
// along with their SHA-256 hashes for storage.
func generateTokenPair() (accessToken, accessHash, refreshToken, refreshHash string, err error) {
	rawAccess := make([]byte, 32)
	if _, err = rand.Read(rawAccess); err != nil {
		return
	}
	accessToken = hex.EncodeToString(rawAccess)
	h := sha256.Sum256([]byte(accessToken))
	accessHash = hex.EncodeToString(h[:])

	rawRefresh := make([]byte, 32)
	if _, err = rand.Read(rawRefresh); err != nil {
		return
	}
	refreshToken = hex.EncodeToString(rawRefresh)
	h2 := sha256.Sum256([]byte(refreshToken))
	refreshHash = hex.EncodeToString(h2[:])
	return
}

// TokenHandler returns http.HandlerFunc for POST /token.
// Supports grant_type=authorization_code and grant_type=refresh_token.
func TokenHandler(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Reject application/json — must be form-encoded.
		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "application/json") {
			http.Error(w, "unsupported media type — use application/x-www-form-urlencoded", http.StatusUnsupportedMediaType)
			return
		}

		if err := r.ParseForm(); err != nil {
			writeTokenError(w, "invalid_request", http.StatusBadRequest)
			return
		}

		grantType := r.FormValue("grant_type")

		switch grantType {
		case "authorization_code":
			handleAuthCode(w, r, db)
		case "refresh_token":
			handleRefreshToken(w, r, db)
		default:
			writeTokenError(w, "unsupported_grant_type", http.StatusBadRequest)
		}
	}
}

func handleAuthCode(w http.ResponseWriter, r *http.Request, db *store.Store) {
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	if code == "" || clientID == "" || redirectURI == "" || codeVerifier == "" {
		writeTokenError(w, "invalid_request", http.StatusBadRequest)
		return
	}

	// SHA-256 hash the raw code value.
	h := sha256.Sum256([]byte(code))
	codeHash := hex.EncodeToString(h[:])

	// Atomically consume the auth code (SELECT + DELETE).
	authCode, err := db.ConsumeAuthCode(codeHash)
	if err != nil {
		if errors.Is(err, store.ErrAuthCodeNotFound) {
			writeTokenError(w, "invalid_grant", http.StatusBadRequest)
			return
		}
		log.Printf("token: consume auth code: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Verify client_id.
	if authCode.ClientID != clientID {
		writeTokenError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	// Verify redirect_uri (byte-exact).
	if authCode.RedirectURI != redirectURI {
		writeTokenError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Verify PKCE: S256(code_verifier) must match stored challenge.
	if s256(codeVerifier) != authCode.PKCEChallenge {
		writeTokenError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Generate access + refresh token pair.
	accessToken, accessHash, refreshToken, refreshHash, err := generateTokenPair()
	if err != nil {
		log.Printf("token: generate token pair: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Activate the grant: pending → active.
	if err := db.ActivateGrant(authCode.GrantID, accessHash, refreshHash); err != nil {
		log.Printf("token: activate grant: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck
		AccessToken:  accessToken,
		TokenType:    "bearer",
		ExpiresIn:    3600,
		RefreshToken: refreshToken,
	})
}

func handleRefreshToken(w http.ResponseWriter, r *http.Request, db *store.Store) {
	refreshToken := r.FormValue("refresh_token")
	if refreshToken == "" {
		writeTokenError(w, "invalid_request", http.StatusBadRequest)
		return
	}

	// Hash the presented refresh token.
	h := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(h[:])

	// Look up grant by refresh token hash.
	grant, err := db.GetGrantByRefreshToken(tokenHash)
	if err != nil {
		log.Printf("token: get grant by refresh token: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if grant == nil {
		// Token completely unknown.
		writeTokenError(w, "invalid_grant", http.StatusBadRequest)
		return
	}
	if grant.GrantState != store.GrantStateActive {
		// Grant exists but is not active (needs_reconnect, revoked, etc.).
		writeTokenError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// Generate new access + refresh token pair.
	newAccessToken, newAccessHash, newRefreshToken, newRefreshHash, err := generateTokenPair()
	if err != nil {
		log.Printf("token: generate token pair: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Atomically rotate: only succeeds if the grant is still active with the same refresh hash.
	rotated, err := db.RotateTokens(grant.GrantID, tokenHash, newAccessHash, newRefreshHash)
	if err != nil {
		log.Printf("token: rotate tokens: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !rotated {
		// Lost the race (concurrent rotation already committed).
		writeTokenError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck
		AccessToken:  newAccessToken,
		TokenType:    "bearer",
		ExpiresIn:    3600,
		RefreshToken: newRefreshToken,
	})
}
