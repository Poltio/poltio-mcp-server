package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

const maxImageSizeBytes = 2 * 1024 * 1024

var allowedImageExts = map[string]bool{
	"png":  true,
	"jpg":  true,
	"jpeg": true,
	"gif":  true,
	"webp": true,
}

// imageHTTPClient fetches images referenced by image_url. The timeout bounds a
// slow or hung remote so the tool fails fast instead of blocking the session.
var imageHTTPClient = &http.Client{Timeout: 30 * time.Second}

type UploadClient interface {
	PostFormMultipart(path string, fields map[string]string) ([]byte, error)
	PostFormFile(path, fieldName, filename string, content []byte) ([]byte, error)
	PostFormFileFields(path, fieldName, filename string, content []byte, fields map[string]string) ([]byte, error)
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

// loadFromFile reads a local image file, rejecting anything that is not a
// regular file. A directory, device (e.g. /dev/zero), or named pipe would
// otherwise stat as size 0 and read forever into an OOM — and opening a FIFO
// blocks until a writer appears. We therefore stat *before* opening (stat never
// blocks) to reject non-regular paths up front, then re-stat the open
// descriptor to close the stat→open TOCTOU window, and bound the read one byte
// past the limit so an oversized file is rejected without slurping it.
//
// (Opening non-blocking would avoid the pre-stat, but that needs POSIX-only
// flags and this binary also targets Windows.)
func loadFromFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read image_path %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("image_path %q is not a regular file", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read image_path %q: %w", path, err)
	}
	defer f.Close() //nolint:errcheck

	info, err = f.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot read image_path %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("image_path %q is not a regular file", path)
	}

	b, err := io.ReadAll(io.LimitReader(f, maxImageSizeBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read image_path %q: %w", path, err)
	}
	if len(b) > maxImageSizeBytes {
		return nil, fmt.Errorf("image exceeds maximum allowed size of 2 MiB")
	}
	return b, nil
}

// loadFromURL fetches a remote image, capping the read one byte past the size
// limit so an oversized response is rejected without buffering it whole.
func loadFromURL(ctx context.Context, urlStr string) ([]byte, error) {
	// Parse rather than prefix-match so the scheme check is case-insensitive
	// (url.Parse lowercases the scheme), and reject anything without an http(s)
	// scheme and a host.
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid image_url: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("image_url must be an http or https URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid image_url: %w", err)
	}
	resp, err := imageHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch image_url: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image_url returned HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSizeBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read image_url: %w", err)
	}
	if len(b) > maxImageSizeBytes {
		return nil, fmt.Errorf("image exceeds maximum allowed size of 2 MiB")
	}
	return b, nil
}

// decodeBase64Image normalizes and strictly decodes a base64 image payload.
//
// It strips whitespace/newlines first so data URI prefix detection is robust
// against wrapped or spaced prefixes (e.g. "data:image/png;\nbase64,"). The
// base64 CLI wraps output at 76 chars, and Poltio decodes strictly ("invalid
// base64 encoding" on stray chars). A cheap DecodedLen pre-flight rejects
// oversized payloads before allocating/decoding them.
func decodeBase64Image(f string) ([]byte, error) {
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
	// Strip data URI prefix if present; the canonical payload is rebuilt later.
	if idx := strings.Index(raw, ";base64,"); idx != -1 {
		raw = raw[idx+8:]
	}
	if raw == "" {
		return nil, fmt.Errorf("image_base64 is empty")
	}
	if base64.StdEncoding.DecodedLen(len(raw)) > maxImageSizeBytes {
		return nil, fmt.Errorf("image exceeds maximum allowed size of 2 MiB")
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}
	return decoded, nil
}

// UploadImage uploads an image to Poltio and returns the file path. Use the
// returned "file" value as the background field for content, questions, answers,
// or results.
//
// The image is supplied by exactly one of:
//   - image_path: a local file path the server reads from disk (preferred — the
//     image bytes never pass through the model context, so uploads are fast and
//     don't fail on large files)
//   - image_url:  an http(s) URL the server fetches itself
//   - image_base64: raw base64 or a data URI, for clients that only have bytes
//
// The storage bucket is chosen by the API server, not the caller.
//
// API limits enforced here:
//   - decoded image must be <= 2 MiB
//   - content must sniff as png, jpeg, gif, or webp
//   - ext, if given, must be png, jpg, jpeg, gif, or webp; otherwise it is
//     derived from the sniffed content
func UploadImage(c UploadClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pathArg := strings.TrimSpace(req.GetString("image_path", ""))
		urlArg := strings.TrimSpace(req.GetString("image_url", ""))
		b64Arg := req.GetString("image_base64", "")
		hasB64 := strings.TrimSpace(b64Arg) != ""

		// Require exactly one source so the call is unambiguous.
		sources := 0
		for _, present := range []bool{pathArg != "", urlArg != "", hasB64} {
			if present {
				sources++
			}
		}
		if sources == 0 {
			return nil, fmt.Errorf("provide exactly one of image_path, image_url, or image_base64")
		}
		if sources > 1 {
			return nil, fmt.Errorf("provide only one of image_path, image_url, or image_base64")
		}

		var (
			decoded []byte
			err     error
		)
		switch {
		case pathArg != "":
			decoded, err = loadFromFile(pathArg)
		case urlArg != "":
			decoded, err = loadFromURL(ctx, urlArg)
		default:
			decoded, err = decodeBase64Image(b64Arg)
		}
		if err != nil {
			return nil, err
		}
		if len(decoded) == 0 {
			return nil, fmt.Errorf("image is empty")
		}
		// File/URL sources are size-checked while loading; base64 via its
		// DecodedLen pre-flight. This is the final backstop for all sources.
		if len(decoded) > maxImageSizeBytes {
			return nil, fmt.Errorf("image exceeds maximum allowed size of 2 MiB")
		}

		// Verify the bytes are actually a supported image. The backend derives
		// the extension from sniffed content, so reject non-image payloads here
		// with a precise error instead of an opaque API 400.
		sniffed := sniffImageType(decoded)
		if sniffed == "" {
			return nil, fmt.Errorf("data is not a supported image (png, jpg, jpeg, gif, webp)")
		}

		// ext is optional: validate it if given, otherwise derive it from the
		// sniffed content. The backend ignores this field (it sniffs too), so it
		// is informational only.
		ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(req.GetString("ext", "")), "."))
		if ext == "" {
			ext = sniffed
		} else if !allowedImageExts[ext] {
			return nil, fmt.Errorf("ext must be one of: png, jpg, jpeg, gif, webp")
		}

		// Re-encode the bytes to canonical base64 so the data URI is clean
		// regardless of source. The MIME comes from the sniffed content
		// (image/jpg is not valid; the canonical form is image/jpeg).
		dataURI := "data:image/" + sniffed + ";base64," + base64.StdEncoding.EncodeToString(decoded)
		fields := map[string]string{"f": dataURI, "ext": ext}
		data, err := c.PostFormMultipart("/platform/content/upload", fields)
		if err != nil {
			return nil, fmt.Errorf("upload_image: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
