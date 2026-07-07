package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListWidgets(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if v := req.GetString("public_id", ""); v != "" {
			q.Set("public_id", v)
		}
		data, err := c.Get("/platform/widgets", q)
		if err != nil {
			return nil, fmt.Errorf("list_widgets: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		body := map[string]any{"public_id": publicID}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("is_default", -1); v >= 0 {
			body["is_default"] = v == 1
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		if v := req.GetString("urls", ""); v != "" {
			parts := strings.Split(v, ",")
			filtered := make([]map[string]any, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					filtered = append(filtered, map[string]any{"url": s})
				}
			}
			if len(filtered) > 0 {
				body["urls"] = filtered
			}
		}
		if v := req.GetString("starts_at", ""); v != "" {
			body["starts_at"] = v
		}
		if v := req.GetString("ends_at", ""); v != "" {
			body["ends_at"] = v
		}
		if v := req.GetString("overlay_options_json", ""); v != "" {
			var opts map[string]any
			if err := json.Unmarshal([]byte(v), &opts); err != nil {
				return nil, fmt.Errorf("overlay_options_json must be valid JSON: %w", err)
			}
			body["overlay_options"] = opts
		}
		data, err := c.Post("/platform/widgets", body)
		if err != nil {
			return nil, fmt.Errorf("create_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		data, err := c.Get("/platform/widgets/"+strconv.Itoa(widgetID), nil)
		if err != nil {
			return nil, fmt.Errorf("get_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("public_id", ""); v != "" {
			body["public_id"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("is_default", -1); v >= 0 {
			body["is_default"] = v == 1
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		if v := req.GetString("urls", ""); v != "" {
			parts := strings.Split(v, ",")
			filtered := make([]map[string]any, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					filtered = append(filtered, map[string]any{"url": s})
				}
			}
			if len(filtered) > 0 {
				body["urls"] = filtered
			}
		}
		if v := req.GetString("starts_at", ""); v != "" {
			body["starts_at"] = v
		}
		if v := req.GetString("ends_at", ""); v != "" {
			body["ends_at"] = v
		}
		if v := req.GetString("overlay_options_json", ""); v != "" {
			var opts map[string]any
			if err := json.Unmarshal([]byte(v), &opts); err != nil {
				return nil, fmt.Errorf("overlay_options_json must be valid JSON: %w", err)
			}
			body["overlay_options"] = opts
		}
		data, err := c.Put("/platform/widgets/"+strconv.Itoa(widgetID), body)
		if err != nil {
			return nil, fmt.Errorf("update_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteWidget(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		widgetID, err := req.RequireInt("widget_id")
		if err != nil {
			return nil, fmt.Errorf("widget_id is required")
		}
		data, err := c.Delete("/platform/widgets/" + strconv.Itoa(widgetID))
		if err != nil {
			return nil, fmt.Errorf("delete_widget: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
