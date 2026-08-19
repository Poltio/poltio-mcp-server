package tools_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Poltio/poltio-mcp-server/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// The elements endpoint validates one element per request, so each mapping is
// posted on its own and a partial result is reported rather than dropped.
func TestSetDataSourceElements_PostsOnePerElementAndReportsPartial(t *testing.T) {
	var posted []any
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			if path != "/platform/data-sources/7/elements" {
				t.Fatalf("unexpected path %q", path)
			}
			posted = append(posted, body)
			if m, ok := body.(map[string]any); ok && m["element"] == "bad" {
				return nil, errors.New("API error 422")
			}
			return []byte(`{"id":1}`), nil
		},
	}
	res, err := tools.SetDataSourceElements(mock)(context.Background(), callRequest(map[string]any{
		"data_source_id": float64(7),
		"elements_json":  `[{"element":"sku","type":"id"},{"element":"bad","type":"name"}]`,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(posted) != 2 {
		t.Fatalf("want 2 posts, got %d", len(posted))
	}
	if first, ok := posted[0].(map[string]any); !ok || first["type"] != "id" {
		t.Fatalf("element body not sent unwrapped: %v", posted[0])
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "Created 1 of 2") || !strings.Contains(text, "bad") {
		t.Fatalf("partial result not reported: %s", text)
	}
}

func TestAddProductFinderField_RequiresFieldOrElement(t *testing.T) {
	mock := &mockClient{postFunc: func(string, any) ([]byte, error) { return []byte(`{}`), nil }}
	_, err := tools.AddProductFinderField(mock)(context.Background(), callRequest(map[string]any{
		"product_finder_id": float64(3),
		"type":              "filter_string",
	}))
	if err == nil {
		t.Fatal("expected an error when neither field nor element is given")
	}
}

func TestAddProductFinderField_SendsBooleansAndElement(t *testing.T) {
	var got map[string]any
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			if path != "/platform/dsc/3/fields" {
				t.Fatalf("unexpected path %q", path)
			}
			got = body.(map[string]any)
			return []byte(`{}`), nil
		},
	}
	_, err := tools.AddProductFinderField(mock)(context.Background(), callRequest(map[string]any{
		"product_finder_id": float64(3),
		"type":              "filter_numeric",
		"element":           float64(42),
		"is_sortable":       float64(1),
		"index":             float64(0),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got["element"] != 42 || got["is_sortable"] != true || got["index"] != false {
		t.Fatalf("body wrong: %v", got)
	}
}

// An import already running answers 400; the source is in the state the caller
// wanted, so the tool reports it rather than failing.
func TestPublishDataSource_AlreadyImportingIsNotAnError(t *testing.T) {
	mock := &mockClient{
		postFunc: func(string, any) ([]byte, error) {
			return nil, errors.New(`API error 400: {"msg":"Data source import is already pending or in progress"}`)
		},
	}
	res, err := tools.PublishDataSource(mock)(context.Background(), callRequest(map[string]any{
		"data_source_id": float64(7),
	}))
	if err != nil {
		t.Fatalf("400 should not surface as a tool error: %v", err)
	}
	if !strings.Contains(res.Content[0].(mcp.TextContent).Text, "already importing") {
		t.Fatalf("unexpected result: %v", res.Content[0])
	}
}

// Any other 400 is a real failure and must not be reported as success.
func TestPublishDataSource_OtherBadRequestStillFails(t *testing.T) {
	mock := &mockClient{
		postFunc: func(string, any) ([]byte, error) {
			return nil, errors.New(`API error 400: {"msg":"Something else went wrong"}`)
		},
	}
	_, err := tools.PublishDataSource(mock)(context.Background(), callRequest(map[string]any{
		"data_source_id": float64(7),
	}))
	if err == nil {
		t.Fatal("an unrelated 400 must surface as an error")
	}
}
