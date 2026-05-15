package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

type OrgClient interface {
	GetOrganizations() ([]byte, error)
	SetOrgID(id string)
}

func ListOrganizations(c OrgClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.GetOrganizations()
		if err != nil {
			return nil, fmt.Errorf("list_organizations: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func SwitchOrganization(c OrgClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireInt("id")
		if err != nil {
			return nil, fmt.Errorf("id is required")
		}
		c.SetOrgID(strconv.Itoa(id))
		return mcp.NewToolResultText(fmt.Sprintf("Switched to organization %d.", id)), nil
	}
}
