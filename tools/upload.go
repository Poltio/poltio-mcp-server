package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const maxImageSizeBytes = 5 * 1024 * 1024

var allowedImageExts = map[string]bool{
	"png":  true,
	"jpg":  true,
	"jpeg": true,
	"gif":  true,
	"webp": true,
}

type UploadClient interface {
	PostFormMultipart(path string, fields map[string]string) ([]byte, error)
	PostFormFile(path, fieldName, filename string, content []byte) ([]byte, error)
}

// sniffImageType inspects the magic bytes of decoded image data and returns the
// canonical format ("png", "jpeg", "gif", "webp") or "" if unrecognized. This
// mirrors the backend's finfo/getimagesizefromstring content check, so a
// model-fabricated or corrupt payload that happens to be valid base64 but isn't
// a real image fails here with a clear message instead of an opaque API 400.
func sniffImageType(b []byte) string {
	switch {
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "png"
	case len(b) >= 3 && bytes.Equal(b[:3], []byte{0xFF, 0xD8, 0xFF}):
		return "jpeg"
	case len(b) >= 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))):
		return "gif"
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "webp"
	}
	return ""
}

// UploadImage uploads a base64-encoded image to Poltio and returns the file path.
// Use the returned "file" value as the background field for content, questions, answers, or results.
//
// The storage bucket is chosen by the API server, not the caller.
//
// API limits enforced here:
//   - ext must be png, jpg, jpeg, gif, or webp
//   - decoded image must be <= 5 MB
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

		// Verify the decoded bytes are actually a supported image. The backend
		// derives the extension from sniffed content (ignoring the request's ext),
		// so reject non-image payloads here with a precise error.
		sniffed := sniffImageType(decoded)
		if sniffed == "" {
			return nil, fmt.Errorf("decoded data is not a supported image (png, jpg, jpeg, gif, webp)")
		}

		// image/jpg is not a valid MIME type; the canonical form is image/jpeg.
		// Use the sniffed type so the data URI matches the actual content.
		mime := sniffed
		dataURI := "data:image/" + mime + ";base64," + raw
		fields := map[string]string{"f": dataURI, "ext": ext}
		data, err := c.PostFormMultipart("/platform/content/upload", fields)
		if err != nil {
			return nil, fmt.Errorf("upload_image: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
