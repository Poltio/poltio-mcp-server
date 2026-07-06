package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListThemes(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/theme", q)
		if err != nil {
			return nil, fmt.Errorf("list_themes: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetDefaultTheme(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/theme/default", nil)
		if err != nil {
			return nil, fmt.Errorf("get_default_theme: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetTheme(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		themeID, err := req.RequireInt("theme_id")
		if err != nil {
			return nil, fmt.Errorf("theme_id is required")
		}
		data, err := c.Get("/platform/theme/"+strconv.Itoa(themeID), nil)
		if err != nil {
			return nil, fmt.Errorf("get_theme: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// CreateTheme accepts a JSON string of theme fields.
// Get the full field list by calling get_default_theme first.
func CreateTheme(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		extra := map[string]any{}
		if raw := req.GetString("fields_json", ""); raw != "" {
			if err := json.Unmarshal([]byte(raw), &extra); err != nil {
				return nil, fmt.Errorf("fields_json must be valid JSON: %w", err)
			}
		}
		extra["name"] = name
		body := map[string]any{"name": name, "theme": extra}
		data, err := c.Post("/platform/theme", body)
		if err != nil {
			return nil, fmt.Errorf("create_theme: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// UpdateTheme accepts a JSON string of theme fields to change.
func UpdateTheme(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		themeID, err := req.RequireInt("theme_id")
		if err != nil {
			return nil, fmt.Errorf("theme_id is required")
		}
		raw, err := req.RequireString("fields_json")
		if err != nil || raw == "" {
			return nil, fmt.Errorf("fields_json is required (JSON object of theme fields to update)")
		}
		var fields map[string]any
		if err := json.Unmarshal([]byte(raw), &fields); err != nil {
			return nil, fmt.Errorf("fields_json must be valid JSON: %w", err)
		}
		body := map[string]any{"theme": fields}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
			fields["name"] = v
		}
		data, err := c.Put("/platform/theme/"+strconv.Itoa(themeID), body)
		if err != nil {
			return nil, fmt.Errorf("update_theme: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func FindTheme(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pageURL, err := req.RequireString("url")
		if err != nil || pageURL == "" {
			return nil, fmt.Errorf("url is required")
		}
		data, err := c.Post("/platform/theme/find", map[string]any{"url": pageURL})
		if err != nil {
			return nil, fmt.Errorf("find_theme: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteTheme(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		themeID, err := req.RequireInt("theme_id")
		if err != nil {
			return nil, fmt.Errorf("theme_id is required")
		}
		data, err := c.Delete("/platform/theme/" + strconv.Itoa(themeID))
		if err != nil {
			return nil, fmt.Errorf("delete_theme: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
