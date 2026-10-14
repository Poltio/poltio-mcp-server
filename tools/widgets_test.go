package tools_test

import (
	"context"
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
