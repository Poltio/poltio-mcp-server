package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

type UploadClient interface {
	PostFormMultipart(path string, fields map[string]string) ([]byte, error)
	PostFormFile(path, fieldName, filename string, content []byte) ([]byte, error)
}

// UploadImage uploads a base64-encoded image to Poltio and returns the file path.
// Use the returned "file" value as the background field for content, questions, answers, or results.
func UploadImage(c UploadClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f, err := req.RequireString("image_base64")
		if err != nil || f == "" {
			return nil, fmt.Errorf("image_base64 is required")
		}
		ext, err := req.RequireString("ext")
		if err != nil || ext == "" {
			return nil, fmt.Errorf("ext is required (e.g. png, jpg, webp)")
		}
		ext = strings.TrimPrefix(ext, ".")
		// Strip data URI prefix if already present, then rebuild it
		raw := f
		if idx := strings.Index(raw, ";base64,"); idx != -1 {
			raw = raw[idx+8:]
		}
		dataURI := "data:image/" + ext + ";base64," + raw
		fields := map[string]string{"f": dataURI, "ext": ext}
		if bucket := req.GetString("bucket", ""); bucket != "" {
			fields["bucket"] = bucket
		}
		data, err := c.PostFormMultipart("/platform/content/upload", fields)
		if err != nil {
			return nil, fmt.Errorf("upload_image: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
