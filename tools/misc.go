package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func SearchPlayground(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body := map[string]any{}
		if v := req.GetInt("content_id", 0); v > 0 {
			body["content_id"] = v
		}
		if v := req.GetString("public_id", ""); v != "" {
			body["public_id"] = v
		}
		if v := req.GetString("query_json", ""); v != "" {
			var queries []string
			if err := json.Unmarshal([]byte(v), &queries); err != nil {
				return nil, fmt.Errorf("query_json must be a JSON array of strings: %w", err)
			}
			body["query"] = queries
		}
		if v := req.GetString("filter_json", ""); v != "" {
			var filters []string
			if err := json.Unmarshal([]byte(v), &filters); err != nil {
				return nil, fmt.Errorf("filter_json must be a JSON array of strings: %w", err)
			}
			body["filter"] = filters
		}
		data, err := c.Post("/platform/search/playground", body)
		if err != nil {
			return nil, fmt.Errorf("search_playground: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CheckSnippetPage(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pageURL, err := req.RequireString("url")
		if err != nil || pageURL == "" {
			return nil, fmt.Errorf("url is required")
		}
		data, err := c.Post("/platform/snippet/check-page", map[string]any{"url": pageURL})
		if err != nil {
			return nil, fmt.Errorf("check_snippet_page: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateShortLink(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		longURL, err := req.RequireString("url")
		if err != nil || longURL == "" {
			return nil, fmt.Errorf("url is required (fully qualified URL to shorten)")
		}
		data, err := c.Post("/platform/link-shorten", map[string]any{"url": longURL})
		if err != nil {
			return nil, fmt.Errorf("create_short_link: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func TriggerDemo(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pageURL, err := req.RequireString("url")
		if err != nil || pageURL == "" {
			return nil, fmt.Errorf("url is required (your checkout success page URL)")
		}
		data, err := c.Post("/platform/trigger-demo", map[string]any{"url": pageURL})
		if err != nil {
			return nil, fmt.Errorf("trigger_demo: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
