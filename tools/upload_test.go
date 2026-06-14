package tools_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Poltio/poltio-mcp-server/tools"
)

// 1x1 transparent GIF, 43 bytes decoded.
const tinyGIF = "R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"

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

func TestUploadImage_JpgMapsToJpegMIME(t *testing.T) {
	// Re-encode the tiny GIF as base64 but request jpg; the tool should build a
	// data:image/jpeg URI even though the underlying bytes are still a GIF.
	var gotFields map[string]string
	mock := &mockUploadClient{
		postFormMultipartFunc: func(path string, fields map[string]string) ([]byte, error) {
			gotFields = fields
			return []byte(`{}`), nil
		},
	}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyGIF,
		"ext":          "jpg",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotFields["f"], "data:image/jpeg;base64,") {
		t.Errorf("f field should use image/jpeg MIME for jpg ext, got %q", gotFields["f"])
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
		"image_base64": tinyGIF,
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

func TestUploadImage_MissingImageBase64(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{"ext": "png"}))
	if err == nil {
		t.Fatal("expected error for missing image_base64, got nil")
	}
}

func TestUploadImage_MissingExt(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{"image_base64": tinyGIF}))
	if err == nil {
		t.Fatal("expected error for missing ext, got nil")
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

func TestUploadImage_InvalidBucket(t *testing.T) {
	mock := &mockUploadClient{}
	handler := tools.UploadImage(mock)
	_, err := handler(context.Background(), callUploadRequest(map[string]any{
		"image_base64": tinyGIF,
		"ext":          "png",
		"bucket":       "bad bucket!",
	}))
	if err == nil {
		t.Fatal("expected error for invalid bucket, got nil")
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
