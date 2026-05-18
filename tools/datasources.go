package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func ListDataSources(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := c.Get("/platform/data-sources", nil)
		if err != nil {
			return nil, fmt.Errorf("list_data_sources: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func CreateDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || name == "" {
			return nil, fmt.Errorf("name is required")
		}
		source, err := req.RequireString("source")
		if err != nil || source == "" {
			return nil, fmt.Errorf("source is required (fully qualified URL for the feed)")
		}
		feedType, err := req.RequireString("type")
		if err != nil || feedType == "" {
			return nil, fmt.Errorf("type is required (xml, json)")
		}
		body := map[string]any{"name": name, "source": source, "type": feedType}
		if v := req.GetString("notes", ""); v != "" {
			body["notes"] = v
		}
		data, err := c.Post("/platform/data-sources", body)
		if err != nil {
			return nil, fmt.Errorf("create_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func DeleteDataSource(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		data, err := c.Delete("/platform/data-sources/" + strconv.Itoa(dataSourceID))
		if err != nil {
			return nil, fmt.Errorf("delete_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func AddDataSourceNote(c ContentClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataSourceID, err := req.RequireInt("data_source_id")
		if err != nil {
			return nil, fmt.Errorf("data_source_id is required")
		}
		notes, err := req.RequireString("notes")
		if err != nil || notes == "" {
			return nil, fmt.Errorf("notes is required")
		}
		path := "/platform/data-sources/" + strconv.Itoa(dataSourceID) + "/note"
		data, err := c.Post(path, map[string]any{"notes": notes})
		if err != nil {
			return nil, fmt.Errorf("add_data_source_note: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func UploadDataSource(c UploadClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileBase64, err := req.RequireString("file_base64")
		if err != nil || fileBase64 == "" {
			return nil, fmt.Errorf("file_base64 is required (base64-encoded file content)")
		}
		filename, err := req.RequireString("filename")
		if err != nil || filename == "" {
			return nil, fmt.Errorf("filename is required (e.g. feed.json, data.csv)")
		}
		content, err := base64.StdEncoding.DecodeString(fileBase64)
		if err != nil {
			return nil, fmt.Errorf("file_base64 is not valid base64: %w", err)
		}
		data, err := c.PostFormFile("/platform/data-sources/upload", "file", filename, content)
		if err != nil {
			return nil, fmt.Errorf("upload_data_source: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
