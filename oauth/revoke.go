package oauth

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/Poltio/poltio-mcp-server/store"
)

// RevokeHandler returns http.HandlerFunc for POST /revoke (RFC 7009).
// Always returns 200 OK, even if the token is unknown (per RFC 7009 §2.2).
func RevokeHandler(db *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			// Still return 200 per RFC 7009 §2.2 — malformed token param means nothing to revoke.
			w.WriteHeader(http.StatusOK)
			return
		}

		token := r.FormValue("token")
		if token == "" {
			// Missing token — nothing to revoke; return 200.
			w.WriteHeader(http.StatusOK)
			return
		}

		// Hash the presented token.
		h := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(h[:])

		// Try access token lookup first.
		grant, err := db.GetGrantByAccessToken(tokenHash)
		if err != nil {
			log.Printf("revoke: get grant by access token: %v", err)
			// Per RFC 7009 §2.2.1 we SHOULD respond with error on server errors,
			// but returning 200 is also acceptable. Return 200 to avoid leaking info.
			w.WriteHeader(http.StatusOK)
			return
		}
		if grant != nil {
			if err := db.RevokeGrant(grant.GrantID); err != nil {
				log.Printf("revoke: revoke grant (access token): %v", err)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		// Try refresh token lookup.
		grant, err = db.GetGrantByRefreshToken(tokenHash)
		if err != nil {
			log.Printf("revoke: get grant by refresh token: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}
		if grant != nil {
			if err := db.RevokeGrant(grant.GrantID); err != nil {
				log.Printf("revoke: revoke grant (refresh token): %v", err)
			}
		}

		// Return 200 regardless (RFC 7009 §2.2 — even unknown tokens get 200).
		w.WriteHeader(http.StatusOK)
	}
}
