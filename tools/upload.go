package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const maxImageSizeBytes = 5 * 1024 * 1024

var (
	allowedImageExts = map[string]bool{
		"png":  true,
		"jpg":  true,
		"jpeg": true,
		"gif":  true,
		"webp": true,
	}
	bucketRegex = regexp.MustCompile(`^[a-zA-Z0-9\/_-]+$`)
)

type UploadClient interface {
	PostFormMultipart(path string, fields map[string]string) ([]byte, error)
	PostFormFile(path, fieldName, filename string, content []byte) ([]byte, error)
}

// UploadImage uploads a base64-encoded image to Poltio and returns the file path.
// Use the returned "file" value as the background field for content, questions, answers, or results.
//
// API limits enforced here:
//   - ext must be png, jpg, jpeg, gif, or webp
//   - decoded image must be <= 5 MB
//   - bucket, if provided, may only contain letters, numbers, /, _, and -
func UploadImage(c UploadClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f, err := req.RequireString("image_base64")
		if err != nil || f == "" {
			return nil, fmt.Errorf("image_base64 is required")
		}
		ext, err := req.RequireString("ext")
		if err != nil || ext == "" {
			return nil, fmt.Errorf("ext is required (png, jpg, jpeg, gif, webp)")
		}
		ext = strings.ToLower(strings.TrimPrefix(ext, "."))
		if !allowedImageExts[ext] {
			return nil, fmt.Errorf("ext must be one of: png, jpg, jpeg, gif, webp")
		}

		// Strip whitespace/newlines first so data URI prefix detection is robust
		// against wrapped or spaced prefixes (e.g. "data:image/png;\nbase64,").
		// The base64 CLI wraps output at 76 chars, and Poltio decodes strictly
		// ("invalid base64 encoding" on stray chars).
		raw := strings.Map(func(r rune) rune {
			switch r {
			case '\n', '\r', ' ', '\t':
				return -1
			}
			return r
		}, f)
		if raw == "" {
			return nil, fmt.Errorf("image_base64 is empty")
		}

		// Strip data URI prefix if already present, then rebuild it.
		if idx := strings.Index(raw, ";base64,"); idx != -1 {
			raw = raw[idx+8:]
		}
		if raw == "" {
			return nil, fmt.Errorf("image_base64 is empty")
		}

		// Cheap pre-flight size check to avoid allocating/decoding huge payloads.
		if base64.StdEncoding.DecodedLen(len(raw)) > maxImageSizeBytes {
			return nil, fmt.Errorf("image exceeds maximum allowed size of 5 MB")
		}

		// Reject malformed base64 before the API does. The decoded size is already
		// bounded by the pre-flight DecodedLen check above, so no further size check
		// is needed after decoding.
		decoded, err := base64.StdEncoding.Strict().DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 encoding: %w", err)
		}
		if len(decoded) == 0 {
			return nil, fmt.Errorf("image_base64 is empty")
		}

		// image/jpg is not a valid MIME type; the canonical form is image/jpeg.
		mime := ext
		if mime == "jpg" {
			mime = "jpeg"
		}
		dataURI := "data:image/" + mime + ";base64," + raw
		fields := map[string]string{"f": dataURI, "ext": ext}
		if bucket := req.GetString("bucket", ""); bucket != "" {
			if !bucketRegex.MatchString(bucket) {
				return nil, fmt.Errorf("bucket may only contain letters, numbers, slashes, underscores, and hyphens")
			}
			fields["bucket"] = bucket
		}
		data, err := c.PostFormMultipart("/platform/content/upload", fields)
		if err != nil {
			return nil, fmt.Errorf("upload_image: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
