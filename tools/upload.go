package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

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

// isDisallowedIP reports whether an address is one the server must never be
// coaxed into connecting to via image_url. This blocks SSRF against internal
// services — most importantly the cloud metadata endpoint (169.254.169.254,
// caught by IsLinkLocalUnicast) when the server runs in hosted/HTTP mode.
func isDisallowedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// imageHTTPClient fetches images referenced by image_url. The timeout bounds a
// slow or hung remote so the tool fails fast instead of blocking the session.
// Its dialer's Control hook runs after DNS resolution, on the actual IP being
// dialed, so it rejects private/loopback/link-local targets and defeats DNS
// rebinding (a public hostname re-resolving to an internal IP) too.
var imageHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			Control: func(_, address string, _ syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					return err
				}
				ip := net.ParseIP(host)
				if ip == nil || isDisallowedIP(ip) {
					return fmt.Errorf("refusing to connect to non-public address %s", host)
				}
				return nil
			},
		}).DialContext,
	},
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

// loadFromFile reads a local image file, guarding against oversized files by
// stat-ing before reading so a huge file is rejected without slurping it.
func loadFromFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read image_path %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("image_path %q is a directory, not a file", path)
	}
	if info.Size() > maxImageSizeBytes {
		return nil, fmt.Errorf("image exceeds maximum allowed size of 5 MB")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read image_path %q: %w", path, err)
	}
	return b, nil
}

// loadFromURL fetches a remote image, capping the read one byte past the size
// limit so an oversized response is rejected without buffering it whole.
func loadFromURL(ctx context.Context, url string) ([]byte, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("image_url must be an http or https URL")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("image exceeds maximum allowed size of 5 MB")
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
		return nil, fmt.Errorf("image exceeds maximum allowed size of 5 MB")
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
//   - decoded image must be <= 5 MB
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
			return nil, fmt.Errorf("image exceeds maximum allowed size of 5 MB")
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
