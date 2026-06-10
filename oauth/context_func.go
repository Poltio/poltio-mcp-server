package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strings"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/store"
)

// context keys for bridge sentinels
type needsAuthKey struct{}
type needsReconnectKey struct{}

// BridgeContextFunc returns an HTTPContextFunc that validates the Bearer token
// against the grant store, decrypts the Poltio credentials, and stores a
// per-request *client.PoltioClient in context (or a sentinel on failure).
func BridgeContextFunc(db *store.Store, encKey []byte, baseURL string) func(context.Context, *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" || token == auth {
			return context.WithValue(ctx, needsAuthKey{}, true)
		}

		h := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(h[:])

		grant, err := db.GetGrantByAccessToken(tokenHash)
		if err != nil {
			log.Printf("bridge: grant lookup error: %v", err)
			return context.WithValue(ctx, needsAuthKey{}, true)
		}
		if grant == nil {
			return context.WithValue(ctx, needsAuthKey{}, true)
		}

		switch grant.GrantState {
		case store.GrantStateNeedsReconnect, store.GrantStateRevoked:
			return context.WithValue(ctx, needsReconnectKey{}, true)
		case store.GrantStateActive:
			plaintext, err := store.Decrypt(grant.PoltioTokenEnc, []byte(grant.GrantID), encKey)
			if err != nil {
				log.Printf("bridge: decrypt failure for grant %s: %v", grant.GrantID, err)
				// Active grant whose credentials can't be decrypted → treat as needs_reconnect.
				return context.WithValue(ctx, needsReconnectKey{}, true)
			}
			orgID := grant.PoltioOrgID
			if grant.OrgOverride != "" {
				orgID = grant.OrgOverride
			}
			pc := client.NewForRequest(string(plaintext), orgID, baseURL)
			ctx = client.WithContext(ctx, pc)
			ctx = client.WithGrantID(ctx, grant.GrantID)
			return ctx
		default:
			return context.WithValue(ctx, needsAuthKey{}, true)
		}
	}
}

// NeedsAuth returns true if the context has the needs-auth sentinel.
func NeedsAuth(ctx context.Context) bool {
	v, _ := ctx.Value(needsAuthKey{}).(bool)
	return v
}

// NeedsReconnect returns true if the context has the needs-reconnect sentinel.
func NeedsReconnect(ctx context.Context) bool {
	v, _ := ctx.Value(needsReconnectKey{}).(bool)
	return v
}
