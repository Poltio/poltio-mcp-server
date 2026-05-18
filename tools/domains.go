package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListDomains(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/domains", nil)
		if err != nil {
			return nil, fmt.Errorf("list_domains: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateDomain(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		domain, err := req.RequireString("domain")
		if err != nil || domain == "" {
			return nil, fmt.Errorf("domain is required (e.g. poltio.yourdomain.com)")
		}
		body := map[string]any{"domain": domain}
		if v := req.GetInt("is_default", -1); v >= 0 {
			body["is_default"] = v == 1
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		data, err := c.Post("/platform/domains", body)
		if err != nil {
			return nil, fmt.Errorf("create_domain: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UpdateDomain(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		domainID, err := req.RequireInt("domain_id")
		if err != nil {
			return nil, fmt.Errorf("domain_id is required")
		}
		body := map[string]any{}
		if v := req.GetString("domain", ""); v != "" {
			body["domain"] = v
		}
		if v := req.GetInt("is_default", -1); v >= 0 {
			body["is_default"] = v == 1
		}
		if v := req.GetInt("is_active", -1); v >= 0 {
			body["is_active"] = v == 1
		}
		data, err := c.Put("/platform/domains/"+strconv.Itoa(domainID), body)
		if err != nil {
			return nil, fmt.Errorf("update_domain: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteDomain(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		domainID, err := req.RequireInt("domain_id")
		if err != nil {
			return nil, fmt.Errorf("domain_id is required")
		}
		data, err := c.Delete("/platform/domains/" + strconv.Itoa(domainID))
		if err != nil {
			return nil, fmt.Errorf("delete_domain: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
