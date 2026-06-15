package tools_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Poltio/poltio-mcp-server/tools"
)

// pngBytes is a minimal byte sequence carrying valid PNG magic bytes, enough
// for content sniffing, used by file/url tests that need raw bytes on disk.
var pngBytes = append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 20)...)

// 1x1 transparent GIF, 43 bytes decoded.
const tinyGIF = "R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"

// Byte sequences carrying valid image magic bytes, enough for content sniffing.
var (
	tinyJPEG = base64.StdEncoding.EncodeToString(append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 20)...))
	tinyPNG  = base64.StdEncoding.EncodeToString(append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 20)...))
)

type mockUploadClient struct {
	postFormMultipartFunc func(path string, fields map[string]string) ([]byte, error)
}

func (m *mockUploadClient) PostFormMultipart(path string, fields map[string]string) ([]byte, error) {
	return m.postFormMultipartFunc(path, fields)
}

func (m *mockUploadClient) PostFormFile(path, fieldName, filename string, content []byte) ([]byte, error) {
	return nil, nil
}

func callUploadRequest(args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

func TestUploadImage_Success(t *testing.T) {
	var gotPath string
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotPath = path
			gotFields = fields
			return []byte(`{"file":"content/gcp/1234567890.gif"}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyGIF,
		"ext":          "gif",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/platform/content/upload" {
		t.Errorf("path: want %q, got %q", "/platform/content/upload", gotPath)
	}
	if gotFields["ext"] != "gif" {
		t.Errorf("ext field: want gif, got %q", gotFields["ext"])
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/gif;base64,") {
		t.Errorf("f field should start with data:image/gif;base64,, got %q", gotFields["f"])
	}
}

func TestUploadImage_StripsDataURIPrefix(t *testing.T) {
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": "data:image/gif;base64," + tinyGIF,
		"ext":          "gif",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/gif;base64,") {
		t.Errorf("f field should start with data:image/gif;base64,, got %q", gotFields["f"])
	}
}

func TestUploadImage_StripsWhitespace(t *testing.T) {
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	wrapped := tinyGIF[:10] + "\n" + tinyGIF[10:20] + " " + tinyGIF[20:]
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": wrapped,
		"ext":          "gif",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(gotFields["f"], " \n\r\t") {
		t.Errorf("f field should not contain whitespace, got %q", gotFields["f"])
	}
}

func TestUploadImage_StripsWhitespaceInDataURIPrefix(t *testing.T) {
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": "data:image/gif;\nbase64," + tinyGIF,
		"ext":          "gif",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/gif;base64,") {
		t.Errorf("f field should start with data:image/gif;base64,, got %q", gotFields["f"])
	}
}

func TestUploadImage_EmptyPayload(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": "   \n\t  ",
		"ext":          "png",
	}))
	if err == nil {
		t.Fatal("expected error for empty image_base64, got nil")
	}
}

func TestUploadImage_EmptyPayloadAfterPrefix(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": "data:image/png;base64,",
		"ext":          "png",
	}))
	if err == nil {
		t.Fatal("expected error for empty image data, got nil")
	}
}

func TestUploadImage_JpegMIMEFromContent(t *testing.T) {
	// The data URI MIME is derived from the sniffed content, mirroring the
	// backend, so JPEG bytes produce the canonical image/jpeg (never image/jpg).
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyJPEG,
		"ext":          "jpg",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/jpeg;base64,") {
		t.Errorf("f field should use image/jpeg MIME for JPEG content, got %q", gotFields["f"])
	}
}

func TestUploadImage_NormalizesUppercaseExt(t *testing.T) {
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyJPEG,
		"ext":          "JPG",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if gotFields["ext"] != "jpg" {
		t.Errorf("ext field should be normalized to lowercase, got %q", gotFields["ext"])
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/jpeg;base64,") {
		t.Errorf("f field should use image/jpeg MIME, got %q", gotFields["f"])
	}
}

func TestUploadImage_MIMEFromContentNotExt(t *testing.T) {
	// The backend ignores ext and sniffs the bytes; the tool mirrors that, so a
	// mismatched ext does not change the data URI MIME (content wins).
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyPNG,
		"ext":          "jpg",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/png;base64,") {
		t.Errorf("f field MIME should follow PNG content, not jpg ext, got %q", gotFields["f"])
	}
}

func TestUploadImage_RejectsNonImage(t *testing.T) {
	// Valid base64 that decodes to non-image bytes must be rejected before the
	// API sees it, mirroring the backend's finfo/getimagesizefromstring check.
	notAnImage := base64.StdEncoding.EncodeToString([]byte("this is plain text, not an image at all"))
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": notAnImage,
		"ext":          "png",
	}))
	if err == nil {
		t.Fatal("expected error for valid base64 that is not an image, got nil")
	}
}

func TestUploadImage_MissingImageBase64(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{"ext": "png"}))
	if err == nil {
		t.Fatal("expected error for missing image_base64, got nil")
	}
}

func TestUploadImage_OmittedExtDerivedFromContent(t *testing.T) {
	// ext is optional; when omitted it is derived from the sniffed content.
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(_ string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{"image_base64": tinyGIF}))
	if err != nil {
		t.Fatal(err)
	}
	if gotFields["ext"] != "gif" {
		t.Errorf("ext should be derived as gif, got %q", gotFields["ext"])
	}
}

func TestUploadImage_InvalidExt(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyGIF,
		"ext":          "bmp",
	}))
	if err == nil {
		t.Fatal("expected error for invalid ext, got nil")
	}
}

func TestUploadImage_InvalidBase64(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": "not-valid-base64!!!",
		"ext":          "png",
	}))
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestUploadImage_TooLarge(t *testing.T) {
	// 5 MB + 1 byte of zeros, base64 encoded.
	large := base64.StdEncoding.EncodeToString(make([]byte, 5*1024*1024+1))
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": large,
		"ext":          "png",
	}))
	if err == nil {
		t.Fatal("expected error for oversized image, got nil")
	}
}

func TestUploadImage_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(path, pngBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(_ string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{"file":"content/gcp/1.png"}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_path": path,
	}))
	if err != nil {
		t.Fatal(err)
	}
	// ext derived from sniffed content when omitted.
	if gotFields["ext"] != "png" {
		t.Errorf("ext should be derived as png, got %q", gotFields["ext"])
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/png;base64,") {
		t.Errorf("f field should start with data:image/png;base64,, got %q", gotFields["f"])
	}
}

func TestUploadImage_FromFileMissing(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_path": filepath.Join(t.TempDir(), "does-not-exist.png"),
	}))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestUploadImage_FromFileRejectsNonImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notimage.png")
	if err := os.WriteFile(path, []byte("plain text, not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_path": path,
	}))
	if err == nil {
		t.Fatal("expected error for non-image file, got nil")
	}
}

func TestUploadImage_FromFileRejectsNonRegular(t *testing.T) {
	// A FIFO stats as size 0 (passing a naive size check) but would read forever.
	// loadFromFile must reject it as a non-regular file before reading.
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("cannot create fifo: %v", err)
	}
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	done := make(chan error, 1)
	go func() {
		_, err := handler(context.Background(), callUploadRequest(map[string]any{
			"image_path": fifo,
		}))
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error for non-regular file, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("loadFromFile blocked on a fifo instead of rejecting it")
	}
}

func TestUploadImage_FromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(_ string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{"file":"content/gcp/1.png"}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_url": srv.URL,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/png;base64,") {
		t.Errorf("f field should start with data:image/png;base64,, got %q", gotFields["f"])
	}
}

func TestUploadImage_FromURLHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_url": srv.URL,
	}))
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

func TestUploadImage_FromURLRejectsNonHTTPScheme(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_url": "file:///etc/passwd",
	}))
	if err == nil {
		t.Fatal("expected error for non-http(s) scheme, got nil")
	}
}

func TestUploadImage_FromURLRejectsEmptyHost(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_url": "http://",
	}))
	if err == nil {
		t.Fatal("expected error for URL with no host, got nil")
	}
}

func TestUploadImage_FromURLAcceptsUppercaseScheme(t *testing.T) {
	// The scheme check is case-insensitive: an uppercase scheme must not be
	// rejected (url.Parse normalizes it to lowercase).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()
	mock := &mockUploadClient{
		postFormMultipartFunc: func(_ string, _ map[string]string) ([]byte, error) {
			return []byte(`{"file":"content/gcp/1.png"}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	upperScheme := "HTTP://" + strings.TrimPrefix(srv.URL, "http://")
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_url": upperScheme,
	}))
	if err != nil {
		t.Fatalf("uppercase scheme should be accepted, got: %v", err)
	}
}

func TestUploadImage_NoSource(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{"ext": "png"}))
	if err == nil {
		t.Fatal("expected error when no image source provided, got nil")
	}
}

func TestUploadImage_MultipleSources(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyGIF,
		"image_url":    "https://example.com/x.png",
	}))
	if err == nil {
		t.Fatal("expected error when multiple sources provided, got nil")
	}
}

func TestUploadImage_PropagatesClientError(t *testing.T) {
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			return nil, errors.New("network error")
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyGIF,
		"ext":          "png",
	}))
	if err == nil {
		t.Fatal("expected error from client, got nil")
	}
}
