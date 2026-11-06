package tools_test

import (
	"context"
	"testing"

	"github.com/Poltio/poltio-mcp-server/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// These tools skip confirmation as non-destructive, so they must never delete.
func TestAdditiveToolsNeverRemoveExisting(t *testing.T) {
	cases := map[string]struct {
		tool func(tools.ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args map[string]any
	}{
		"add_answers_bulk": {tools.AddAnswersBulk, map[string]any{"public_id": "abc", "question_id": 1, "answers": "a", "remove_existing": 1}},
		"add_lead_codes":   {tools.AddLeadCodes, map[string]any{"lead_id": "1", "codes": "X", "remove_existing": 1}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var got map[string]any
			mock := &mockClient{postFunc: func(_ string, body any) ([]byte, error) {
				got = body.(map[string]any)
				return []byte(`{}`), nil
			}}
			if _, err := tc.tool(mock)(context.Background(), callRequest(tc.args)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got["remove_existing"] != false {
				t.Errorf("remove_existing = %v, want false", got["remove_existing"])
			}
		})
	}
}
