package oauth

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/store"
)

// ReconnectMiddleware returns a ToolHandlerMiddleware that:
//   - On ErrPoltioUnauthorized: marks the grant as needs_reconnect + returns a reconnect error
//   - On ErrPoltioUnavailable: returns a transient error (no state change)
//
// It must be registered AFTER the auth-sentinel middleware so it only runs for authenticated requests.
func ReconnectMiddleware(db *store.Store) server.ToolHandlerMiddleware {
	serverURL := os.Getenv("SERVER_URL")
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := next(ctx, req)
			if err == nil {
				return result, nil
			}
			grantID := client.GrantIDFromContext(ctx)
			switch {
			case errors.Is(err, client.ErrPoltioUnauthorized):
				if grantID != "" && db != nil {
					if markErr := db.MarkNeedsReconnect(grantID); markErr != nil {
						log.Printf("bridge: MarkNeedsReconnect(%s): %v", grantID, markErr)
					}
				}
				reconnectURL := strings.TrimRight(serverURL, "/") + "/authorize"
				return mcp.NewToolResultError(
					"Your Poltio credentials have been revoked or expired. " +
						"Reconnect at " + reconnectURL + " to continue."), nil
			case errors.Is(err, client.ErrPoltioUnavailable):
				return mcp.NewToolResultError(
					"Poltio is temporarily unavailable. Please try again shortly."), nil
			default:
				return result, err
			}
		}
	}
}
