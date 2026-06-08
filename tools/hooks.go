package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

// Sheet Hooks

func ListSheetHooks(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		data, err := c.Get("/platform/hooks/sheet", q)
		if err != nil {
			return nil, fmt.Errorf("list_sheet_hooks: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateSheetHook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		sheetID, err := req.RequireString("sheet_id")
		if err != nil || sheetID == "" {
			return nil, fmt.Errorf("sheet_id is required (Google Sheet ID)")
		}
		body := map[string]any{"public_id": publicID, "sheet_id": sheetID, "is_active": 1}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v
		}
		data, err := c.Post("/platform/hooks/sheet", body)
		if err != nil {
			return nil, fmt.Errorf("create_sheet_hook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetSheetHook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		data, err := c.Get("/platform/hooks/sheet/"+strconv.Itoa(hookID), nil)
		if err != nil {
			return nil, fmt.Errorf("get_sheet_hook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateSheetHook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("sheet_id", ""); v != "" {
			body["sheet_id"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetString("public_id", ""); v != "" {
			body["public_id"] = v
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v
		}
		data, err := c.Put("/platform/hooks/sheet/"+strconv.Itoa(hookID), body)
		if err != nil {
			return nil, fmt.Errorf("update_sheet_hook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteSheetHook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		data, err := c.Delete("/platform/hooks/sheet/" + strconv.Itoa(hookID))
		if err != nil {
			return nil, fmt.Errorf("delete_sheet_hook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetSheetHookLogs(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/hooks/sheet/"+strconv.Itoa(hookID)+"/logs", q)
		if err != nil {
			return nil, fmt.Errorf("get_sheet_hook_logs: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// Webhooks

func ListWebhooks(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		data, err := c.Get("/platform/hooks/web", q)
		if err != nil {
			return nil, fmt.Errorf("list_webhooks: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateWebhook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		webhookURL, err := req.RequireString("url")
		if err != nil || webhookURL == "" {
			return nil, fmt.Errorf("url is required")
		}
		body := map[string]any{"url": webhookURL, "is_active": true}
		if v := req.GetString("public_id", ""); v != "" {
			body["public_id"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("delay", -1); v >= 0 {
			body["delay"] = v
		}
		if v := req.GetInt("send_leads", -1); v >= 0 {
			body["send_leads"] = v == 1
		}
		if v := req.GetInt("send_answers", -1); v >= 0 {
			body["send_answers"] = v == 1
		}
		if v := req.GetInt("account_wide", -1); v >= 0 {
			body["account_wide"] = v == 1
		}
		if v := req.GetInt("incomplete_send", -1); v >= 0 {
			body["incomplete_send"] = v == 1
		}
		if v := req.GetInt("incomplete_delay", -1); v >= 0 {
			body["incomplete_delay"] = v
		}
		if v := req.GetInt("use_oauth", -1); v >= 0 {
			body["use_oauth"] = v == 1
		}
		if v := req.GetString("oauth_login_endpoint", ""); v != "" {
			body["oauth_login_endpoint"] = v
		}
		if v := req.GetString("oauth_request_body_json", ""); v != "" {
			var oauthBody map[string]any
			if err := json.Unmarshal([]byte(v), &oauthBody); err == nil {
				body["oauth_request_body"] = oauthBody
			}
		}
		if v := req.GetString("oauth_request_headers_json", ""); v != "" {
			var oauthHeaders map[string]any
			if err := json.Unmarshal([]byte(v), &oauthHeaders); err == nil {
				body["oauth_request_headers"] = oauthHeaders
			}
		}
		data, err := c.Post("/platform/hooks/web", body)
		if err != nil {
			return nil, fmt.Errorf("create_webhook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetWebhook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		data, err := c.Get("/platform/hooks/web/"+strconv.Itoa(hookID), nil)
		if err != nil {
			return nil, fmt.Errorf("get_webhook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateWebhook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("url", ""); v != "" {
			body["url"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			body["name"] = v
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		if v := req.GetInt("delay", -1); v >= 0 {
			body["delay"] = v
		}
		if v := req.GetInt("send_leads", -1); v >= 0 {
			body["send_leads"] = v == 1
		}
		if v := req.GetInt("send_answers", -1); v >= 0 {
			body["send_answers"] = v == 1
		}
		if v := req.GetString("public_id", ""); v != "" {
			body["public_id"] = v
		}
		if v := req.GetInt("account_wide", -1); v >= 0 {
			body["account_wide"] = v == 1
		}
		if v := req.GetInt("incomplete_send", -1); v >= 0 {
			body["incomplete_send"] = v == 1
		}
		if v := req.GetInt("incomplete_delay", -1); v >= 0 {
			body["incomplete_delay"] = v
		}
		if v := req.GetInt("use_oauth", -1); v >= 0 {
			body["use_oauth"] = v == 1
		}
		if v := req.GetString("oauth_login_endpoint", ""); v != "" {
			body["oauth_login_endpoint"] = v
		}
		if v := req.GetString("oauth_request_body_json", ""); v != "" {
			var oauthBody map[string]any
			if err := json.Unmarshal([]byte(v), &oauthBody); err == nil {
				body["oauth_request_body"] = oauthBody
			}
		}
		if v := req.GetString("oauth_request_headers_json", ""); v != "" {
			var oauthHeaders map[string]any
			if err := json.Unmarshal([]byte(v), &oauthHeaders); err == nil {
				body["oauth_request_headers"] = oauthHeaders
			}
		}
		data, err := c.Put("/platform/hooks/web/"+strconv.Itoa(hookID), body)
		if err != nil {
			return nil, fmt.Errorf("update_webhook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteWebhook(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		data, err := c.Delete("/platform/hooks/web/" + strconv.Itoa(hookID))
		if err != nil {
			return nil, fmt.Errorf("delete_webhook: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetWebhookLogs(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hookID, err := req.RequireInt("hook_id")
		if err != nil {
			return nil, fmt.Errorf("hook_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/hooks/web/"+strconv.Itoa(hookID)+"/logs", q)
		if err != nil {
			return nil, fmt.Errorf("get_webhook_logs: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
