package tools_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/Poltio/poltio-mcp-server/tools"
)

// Only the filters the caller set reach the query; an empty one would fail
// the API's date validation.
func TestGetWidgetStats_SendsOnlySetFilters(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	mock := &mockClient{getFunc: func(path string, query url.Values) ([]byte, error) {
		gotPath, gotQuery = path, query
		return []byte(`{}`), nil
	}}
	_, err := tools.GetWidgetStats(mock)(context.Background(), callRequest(map[string]any{
		"widget_id":   float64(42),
		"start":       "2026-09-01",
		"device_type": "mobile",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/platform/widgets/42/stats" {
		t.Fatalf("path = %q", gotPath)
	}
	if want := "device_type=mobile&start=2026-09-01"; gotQuery.Encode() != want {
		t.Fatalf("query = %q, want %q", gotQuery.Encode(), want)
	}
}

// The API requires public_id, name, is_default and urls on every PUT, so a
// one-field update must carry the stored values for the rest.
func TestUpdateWidget_FillsRequiredFieldsFromStoredWidget(t *testing.T) {
	var gotBody map[string]any
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			return []byte(`{"id":42,"parent_id":null,"public_id":"abc","name":"Home","is_default":false,"urls":[{"id":1,"url":"https://shop.com/*"}]}`), nil
		},
		putFunc: func(path string, body any) ([]byte, error) {
			gotBody = body.(map[string]any)
			return []byte(`{}`), nil
		},
	}
	_, err := tools.UpdateWidget(mock)(context.Background(), callRequest(map[string]any{
		"widget_id": float64(42),
		"name":      "Renamed",
		"ab_json":   "null",
	}))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(gotBody)
	want := `{"ab":null,"is_default":false,"name":"Renamed","public_id":"abc","urls":[{"url":"https://shop.com/*"}]}`
	if string(got) != want {
		t.Fatalf("body = %s\nwant   %s", got, want)
	}
}

// A variation must not send is_default or urls; the API prohibits both.
func TestUpdateWidget_VariationSkipsParentOnlyFields(t *testing.T) {
	var gotBody map[string]any
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			return []byte(`{"id":43,"parent_id":42,"public_id":"abc","name":"B","is_default":false,"urls":[]}`), nil
		},
		putFunc: func(path string, body any) ([]byte, error) {
			gotBody = body.(map[string]any)
			return []byte(`{}`), nil
		},
	}
	_, err := tools.UpdateWidget(mock)(context.Background(), callRequest(map[string]any{
		"widget_id": float64(43),
		"is_active": float64(0),
	}))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(gotBody)
	if want := `{"is_active":false,"name":"B","public_id":"abc"}`; string(got) != want {
		t.Fatalf("body = %s\nwant   %s", got, want)
	}
}
