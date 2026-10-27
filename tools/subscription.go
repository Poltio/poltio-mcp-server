package tools

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func GetSubscription(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/subscription", nil)
		if err != nil {
			return nil, fmt.Errorf("get_subscription: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ListSubscriptionTiers(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var q url.Values
		if v := strings.ToUpper(req.GetString("currency", "")); v != "" {
			if v != "USD" && v != "EUR" {
				return nil, fmt.Errorf("currency must be USD or EUR")
			}
			q = url.Values{"currency": {v}}
		}
		data, err := c.Get("/platform/subscription/tiers", q)
		if err != nil {
			return nil, fmt.Errorf("list_subscription_tiers: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
