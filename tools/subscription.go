package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListSubscriptionTiers(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/subscription/tiers", nil)
		if err != nil {
			return nil, fmt.Errorf("list_subscription_tiers: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateSubscription(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tierID, err := req.RequireInt("tier_id")
		if err != nil {
			return nil, fmt.Errorf("tier_id is required")
		}
		period, err := req.RequireString("period")
		if err != nil || period == "" {
			return nil, fmt.Errorf("period is required (month or year)")
		}
		if period != "month" && period != "year" {
			return nil, fmt.Errorf("period must be month or year")
		}
		data, err := c.Post("/platform/subscription/"+strconv.Itoa(tierID), map[string]any{"period": period})
		if err != nil {
			return nil, fmt.Errorf("create_subscription: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
