package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

// ContentClient is the interface the tool handlers use to call the Poltio API.
// client.PoltioClient satisfies this interface.
type ContentClient interface {
	Get(path string, query url.Values) ([]byte, error)
	Post(path string, body any) ([]byte, error)
}

func ListContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if t := req.GetString("type", ""); t != "" {
			q.Set("type", t)
		}
		if search := req.GetString("q", ""); search != "" {
			q.Set("q", search)
		}
		if order := req.GetString("order", ""); order != "" {
			q.Set("order", order)
		}
		if sort := req.GetString("sort", ""); sort != "" {
			q.Set("sort", sort)
		}
		data, err := c.Get("/platform/content", q)
		if err != nil {
			return nil, fmt.Errorf("list_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID, nil)
		if err != nil {
			return nil, fmt.Errorf("get_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contentType, err := req.RequireString("type")
		if err != nil || contentType == "" {
			return nil, fmt.Errorf("type is required")
		}
		title, err := req.RequireString("title")
		if err != nil || title == "" {
			return nil, fmt.Errorf("title is required")
		}
		body := map[string]any{
			"type":  contentType,
			"title": title,
		}
		if desc := req.GetString("desc", ""); desc != "" {
			body["desc"] = desc
		}
		if name := req.GetString("name", ""); name != "" {
			body["name"] = name
		}
		data, err := c.Post("/platform/content", body)
		if err != nil {
			return nil, fmt.Errorf("create_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func PublishContent(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Post("/platform/content/"+publicID+"/publish", nil)
		if err != nil {
			return nil, fmt.Errorf("publish_content: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func ListDrafts(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := url.Values{}
		if page := req.GetInt("page", 0); page > 0 {
			q.Set("page", strconv.Itoa(page))
		}
		if perPage := req.GetInt("per_page", 0); perPage > 0 {
			q.Set("per_page", strconv.Itoa(perPage))
		}
		if t := req.GetString("type", ""); t != "" {
			q.Set("type", t)
		}
		if search := req.GetString("q", ""); search != "" {
			q.Set("q", search)
		}
		data, err := c.Get("/platform/content/drafts", q)
		if err != nil {
			return nil, fmt.Errorf("list_drafts: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
