package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

// secretKeyNote warns that the share secret is shown once. The API returns
// secret_key only from the create call — it is stored hashed, so a lost key
// cannot be recovered and the share has to be revoked and re-created.
const secretKeyNote = "\n\nNote: secret_key is returned only by this call. It is stored hashed, so it cannot be read back later — a lost key means revoking the share and creating a new one."

func ListContentShares(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		data, err := c.Get("/platform/content/"+publicID+"/shares", nil)
		if err != nil {
			return nil, fmt.Errorf("list_content_shares: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateContentShare(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		body := map[string]any{"name": name}
		if v := req.GetString("time_frame", ""); v != "" {
			body["time_frame"] = v
		}
		data, err := c.Post("/platform/content/"+publicID+"/shares", body)
		if err != nil {
			return nil, fmt.Errorf("create_content_share: %w", err)
		}
		return mcp.NewToolResultText(string(data) + secretKeyNote), nil
	}
}

func RevokeContentShare(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		publicID, err := req.RequireString("public_id")
		if err != nil || publicID == "" {
			return nil, fmt.Errorf("public_id is required")
		}
		shareID, err := req.RequireInt("share_id")
		if err != nil {
			return nil, fmt.Errorf("share_id is required")
		}
		path := "/platform/content/" + publicID + "/shares/" + strconv.Itoa(shareID)
		data, err := c.Delete(path)
		if err != nil {
			return nil, fmt.Errorf("revoke_content_share: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
