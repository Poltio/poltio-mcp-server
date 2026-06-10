package oauth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Poltio/poltio-mcp-server/client"
	"github.com/Poltio/poltio-mcp-server/store"
)

// handlerReturning returns a ToolHandlerFunc that always returns the given result and error.
func handlerReturning(result *mcp.CallToolResult, err error) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return result, err
	}
}

// callMiddleware applies the middleware to the given handler and invokes it.
func callMiddleware(mw server.ToolHandlerMiddleware, h server.ToolHandlerFunc, ctx context.Context) (*mcp.CallToolResult, error) {
	wrapped := mw(h)
	return wrapped(ctx, mcp.CallToolRequest{})
}

// Test 1: ErrPoltioUnauthorized → MarkNeedsReconnect called + tool error with "reconnect" text
func TestReconnectMiddleware_Unauthorized(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	// Create an active grant.
	grantID := "grant-u8-unauth"
	insertBridgeGrant(t, db, key, grantID, "poltio-token", "org-1")

	// Confirm it's active before.
	g, err := db.GetGrant(grantID)
	if err != nil || g == nil {
		t.Fatalf("GetGrant before middleware: err=%v g=%v", err, g)
	}
	if g.GrantState != store.GrantStateActive {
		t.Fatalf("expected active before, got %s", g.GrantState)
	}

	mw := ReconnectMiddleware(db)
	h := handlerReturning(nil, client.ErrPoltioUnauthorized)

	ctx := client.WithGrantID(context.Background(), grantID)
	result, retErr := callMiddleware(mw, h, ctx)

	// Should not propagate as a Go error.
	if retErr != nil {
		t.Fatalf("expected nil error from middleware, got: %v", retErr)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Result must be a tool error.
	// mcp.CallToolResult.IsError is a *bool in some versions; check via content.
	if !result.IsError {
		t.Fatal("expected result.IsError to be true")
	}

	// Text must contain "reconnect" (case-insensitive check via lowercase).
	text := extractResultText(result)
	if !strings.Contains(strings.ToLower(text), "reconnect") {
		t.Fatalf("expected 'reconnect' in error text, got: %q", text)
	}

	// Grant should now be needs_reconnect.
	g2, err := db.GetGrant(grantID)
	if err != nil {
		t.Fatalf("GetGrant after middleware: %v", err)
	}
	if g2.GrantState != store.GrantStateNeedsReconnect {
		t.Fatalf("expected needs_reconnect after unauthorized, got %s", g2.GrantState)
	}
}

// Test 2: ErrPoltioUnavailable → tool error with "unavailable" text, no state change
func TestReconnectMiddleware_Unavailable(t *testing.T) {
	db := openTestStore(t)
	key := testKey(t)

	grantID := "grant-u8-unavail"
	insertBridgeGrant(t, db, key, grantID, "poltio-token", "org-2")

	mw := ReconnectMiddleware(db)
	h := handlerReturning(nil, client.ErrPoltioUnavailable)

	ctx := client.WithGrantID(context.Background(), grantID)
	result, retErr := callMiddleware(mw, h, ctx)

	if retErr != nil {
		t.Fatalf("expected nil error from middleware, got: %v", retErr)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsError {
		t.Fatal("expected result.IsError to be true")
	}

	text := extractResultText(result)
	if !strings.Contains(strings.ToLower(text), "unavailable") {
		t.Fatalf("expected 'unavailable' in error text, got: %q", text)
	}

	// Grant state must be unchanged (still active).
	g, err := db.GetGrant(grantID)
	if err != nil {
		t.Fatalf("GetGrant: %v", err)
	}
	if g.GrantState != store.GrantStateActive {
		t.Fatalf("expected grant to remain active, got %s", g.GrantState)
	}
}

// Test 3: Other error → passed through unchanged (non-nil Go error)
func TestReconnectMiddleware_OtherError(t *testing.T) {
	db := openTestStore(t)

	mw := ReconnectMiddleware(db)
	sentinel := errors.New("unexpected tool error")
	h := handlerReturning(nil, sentinel)

	result, retErr := callMiddleware(mw, h, context.Background())

	if !errors.Is(retErr, sentinel) {
		t.Fatalf("expected sentinel error to be propagated, got: %v", retErr)
	}
	if result != nil {
		t.Fatalf("expected nil result for propagated error, got: %v", result)
	}
}

// Test 4: nil error → passed through unchanged
func TestReconnectMiddleware_NoError(t *testing.T) {
	db := openTestStore(t)

	mw := ReconnectMiddleware(db)
	expected := mcp.NewToolResultText("all good")
	h := handlerReturning(expected, nil)

	result, retErr := callMiddleware(mw, h, context.Background())

	if retErr != nil {
		t.Fatalf("expected nil error, got: %v", retErr)
	}
	if result != expected {
		t.Fatalf("expected original result to be returned unchanged")
	}
}

// extractResultText returns the concatenated text from all content items in a CallToolResult.
func extractResultText(r *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range r.Content {
		if t, ok := c.(mcp.TextContent); ok {
			sb.WriteString(t.Text)
		}
	}
	return sb.String()
}
