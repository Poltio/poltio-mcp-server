package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListReports(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		data, err := c.Get("/platform/reports", q)
		if err != nil {
			return nil, fmt.Errorf("list_reports: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateReport(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		report, err := req.RequireString("report")
		if err != nil || report == "" {
			return nil, fmt.Errorf("report is required (content-sessions, content-voters, voter-leads, answer-voters)")
		}
		body := map[string]any{"report": report}
		if v := req.GetString("public_id", ""); v != "" {
			body["public_id"] = v
		}
		if v := req.GetInt("base_id", 0); v > 0 {
			body["base_id"] = v
		}
		if v := req.GetString("target_ids", ""); v != "" {
			body["target_ids"] = v
		}
		data, err := c.Post("/platform/reports", body)
		if err != nil {
			return nil, fmt.Errorf("create_report: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
