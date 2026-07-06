package tools

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func GetDashboard(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/dashboard", nil)
		if err != nil {
			return nil, fmt.Errorf("get_dashboard: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetDashboardSummary(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := req.GetString("start", ""); v != "" {
			q.Set("start", v)
		}
		if v := req.GetString("end", ""); v != "" {
			q.Set("end", v)
		}
		if v := req.GetInt("take", 0); v > 0 {
			q.Set("take", fmt.Sprintf("%d", v))
		}
		data, err := c.Get("/platform/dashboard/summary", q)
		if err != nil {
			return nil, fmt.Errorf("get_dashboard_summary: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetDashboardMetrics(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		period, err := req.RequireString("period")
		if err != nil || period == "" {
			return nil, fmt.Errorf("period is required (day, week, month, year)")
		}
		body := map[string]any{}
		if v := req.GetString("start", ""); v != "" {
			body["start"] = v
		}
		if v := req.GetString("end", ""); v != "" {
			body["end"] = v
		}
		if v := req.GetString("metrics", ""); v != "" {
			parts := strings.Split(v, ",")
			trimmed := make([]string, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					trimmed = append(trimmed, s)
				}
			}
			if len(trimmed) > 0 {
				body["metrics"] = trimmed
			}
		}
		if v := req.GetString("device_type", ""); v != "" {
			body["device_type"] = v
		}
		data, err := c.Post("/platform/dashboard/metrics/"+period, body)
		if err != nil {
			return nil, fmt.Errorf("get_dashboard_metrics: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetDashboardStats(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if v := req.GetString("start", ""); v != "" {
			q.Set("start", v)
		}
		if v := req.GetString("end", ""); v != "" {
			q.Set("end", v)
		}
		if v := req.GetString("device_type", ""); v != "" {
			q.Set("device_type", v)
		}
		data, err := c.Get("/platform/dashboard/stats", q)
		if err != nil {
			return nil, fmt.Errorf("get_dashboard_stats: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
