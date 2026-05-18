package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

// StatsClient is satisfied by client.PoltioClient for stats-only calls.
type StatsClient interface {
	Get(path string, query url.Values) ([]byte, error)
}

func GetVoters(c StatsClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if v := req.GetInt("download", -1); v >= 0 {
			q.Set("download", strconv.Itoa(v))
		}
		data, err := c.Get("/platform/content/"+publicID+"/voters", q)
		if err != nil {
			return nil, fmt.Errorf("get_voters: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetConversionTimeStats(c StatsClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if contentID := req.GetInt("content_id", 0); contentID > 0 {
			q.Set("content_id", strconv.Itoa(contentID))
		}
		if v := req.GetString("start", ""); v != "" {
			q.Set("start", v)
		}
		if v := req.GetString("end", ""); v != "" {
			q.Set("end", v)
		}
		data, err := c.Get("/platform/conversion/time-stats", q)
		if err != nil {
			return nil, fmt.Errorf("get_conversion_time_stats: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
