package tools_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Poltio/poltio-mcp-server/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// A JSON array is a batch and goes under "items"; a single object goes under
// "item". Sending the wrong shape makes the API read a batch as one malformed
// product, so this is the part worth pinning down.
func TestCreateDataSourceItems_WrapsArrayAndObjectDifferently(t *testing.T) {
	for _, tc := range []struct {
		name    string
		json    string
		wantKey string
	}{
		{"array is a batch", `[{"source_id":"1","name":"A","url":"http://x/1"}]`, "items"},
		{"object is one item", `{"source_id":"1","name":"A","url":"http://x/1"}`, "item"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			mock := &mockClient{
				postFunc: func(path string, body any) ([]byte, error) {
					if path != "/platform/data-sources/7/items" {
						t.Fatalf("unexpected path %q", path)
					}
					got = body.(map[string]any)
					return []byte(`{}`), nil
				},
			}
			_, err := tools.CreateDataSourceItems(mock)(context.Background(), callRequest(map[string]any{
				"data_source_id": float64(7),
				"items_json":     tc.json,
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := got[tc.wantKey]; !ok {
				t.Fatalf("want payload under %q, got %v", tc.wantKey, got)
			}
		})
	}
}

// The API rejects a batch over 100 outright, losing every item in it. Caught
// here so the caller is told to split it rather than getting a bare 400.
func TestCreateDataSourceItems_RejectsOversizedBatchWithoutCalling(t *testing.T) {
	called := false
	mock := &mockClient{postFunc: func(string, any) ([]byte, error) {
		called = true
		return []byte(`{}`), nil
	}}
	items := "[" + strings.Repeat(`{"source_id":"1"},`, 100) + `{"source_id":"1"}]`
	_, err := tools.CreateDataSourceItems(mock)(context.Background(), callRequest(map[string]any{
		"data_source_id": float64(7),
		"items_json":     items,
	}))
	if err == nil {
		t.Fatal("101 items must be rejected")
	}
	if called {
		t.Fatal("the oversized batch must not reach the API")
	}
	if !strings.Contains(err.Error(), "100") {
		t.Fatalf("error should name the limit: %v", err)
	}
}

func TestCreateDataSourceItems_RejectsMalformedJSON(t *testing.T) {
	mock := &mockClient{postFunc: func(string, any) ([]byte, error) { return []byte(`{}`), nil }}
	_, err := tools.CreateDataSourceItems(mock)(context.Background(), callRequest(map[string]any{
		"data_source_id": float64(7),
		"items_json":     `{"source_id":`,
	}))
	if err == nil {
		t.Fatal("malformed JSON must fail before the request is sent")
	}
}

// A feed's source_id can carry a slash or a space; unescaped it would change
// which path segment the API sees.
func TestDeleteDataSourceItem_EscapesSourceIDInPath(t *testing.T) {
	var got string
	mock := &mockClient{deleteFunc: func(path string) ([]byte, error) {
		got = path
		return []byte(`{}`), nil
	}}
	_, err := tools.DeleteDataSourceItem(mock)(context.Background(), callRequest(map[string]any{
		"data_source_id": float64(7),
		"item":           "SKU/12 A",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got != "/platform/data-sources/7/items/SKU%2F12%20A" {
		t.Fatalf("source_id not escaped: %q", got)
	}
}

// The API names the filter's target data_source_item_element_id; the tool takes
// the shorter "element", matching add_product_finder_field.
func TestAddProductFinderFilter_MapsElementToAPIField(t *testing.T) {
	var got map[string]any
	mock := &mockClient{postFunc: func(path string, body any) ([]byte, error) {
		if path != "/platform/dsc/3/filters" {
			t.Fatalf("unexpected path %q", path)
		}
		got = body.(map[string]any)
		return []byte(`{}`), nil
	}}
	_, err := tools.AddProductFinderFilter(mock)(context.Background(), callRequest(map[string]any{
		"product_finder_id": float64(3),
		"element":           float64(42),
		"operator":          "contains",
		"value":             "Acme",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got["data_source_item_element_id"] != 42 {
		t.Fatalf("element not mapped: %v", got)
	}
	if got["operator"] != "contains" || got["value"] != "Acme" {
		t.Fatalf("body wrong: %v", got)
	}
}

func TestAddProductFinderFilter_RequiresElement(t *testing.T) {
	mock := &mockClient{postFunc: func(string, any) ([]byte, error) { return []byte(`{}`), nil }}
	_, err := tools.AddProductFinderFilter(mock)(context.Background(), callRequest(map[string]any{
		"product_finder_id": float64(3),
		"value":             "Acme",
	}))
	if err == nil {
		t.Fatal("a filter with no element must be rejected")
	}
}

// secret_key is shown once, so the note telling the caller that has to travel
// with the response rather than live only in the tool description.
func TestCreateContentShare_AppendsSecretKeyNote(t *testing.T) {
	mock := &mockClient{postFunc: func(path string, body any) ([]byte, error) {
		if path != "/platform/content/abc123/shares" {
			t.Fatalf("unexpected path %q", path)
		}
		if m := body.(map[string]any); m["time_frame"] != "15 days" {
			t.Fatalf("time_frame not forwarded: %v", m)
		}
		return []byte(`{"secret_key":"poltio_cs_x"}`), nil
	}}
	res, err := tools.CreateContentShare(mock)(context.Background(), callRequest(map[string]any{
		"public_id":  "abc123",
		"name":       "Agency",
		"time_frame": "15 days",
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "poltio_cs_x") || !strings.Contains(text, "only by this call") {
		t.Fatalf("secret key note missing: %s", text)
	}
}
