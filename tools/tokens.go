package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListTokens(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/tokens", q)
		if err != nil {
			return nil, fmt.Errorf("list_tokens: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateToken(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		body := map[string]any{"name": name}
		if v := req.GetString("expires", ""); v != "" {
			body["expires"] = v
		}
		data, err := c.Post("/platform/tokens", body)
		if err != nil {
			return nil, fmt.Errorf("create_token: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RevokeToken(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tokenID, err := req.RequireInt("token_id")
		if err != nil {
			return nil, fmt.Errorf("token_id is required")
		}
		data, err := c.Delete("/platform/tokens/" + strconv.Itoa(tokenID))
		if err != nil {
			return nil, fmt.Errorf("revoke_token: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
