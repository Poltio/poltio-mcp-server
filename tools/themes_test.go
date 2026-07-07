package tools_test

import (
	"context"
	"testing"

	"github.com/Poltio/poltio-mcp-server/tools"
)

// Regression: fields_json "null" unmarshals to a nil map and must not panic.

func TestCreateTheme_NullFieldsJSONDoesNotPanic(t *testing.T) {
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	handler := tools.CreateTheme(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{
		"name":        "My Theme",
		"fields_json": "null",
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTheme_NullFieldsJSONDoesNotPanic(t *testing.T) {
	mock := &mockClient{
		putFunc: func(path string, body any) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	handler := tools.UpdateTheme(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{
		"theme_id":    float64(1),
		"fields_json": "null",
		"name":        "Renamed",
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateContent_InvalidOptionsJSONReturnsError(t *testing.T) {
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	handler := tools.CreateContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{
		"type":         "poll",
		"title":        "My Poll",
		"options_json": "{not json",
	}))
	if err == nil {
		t.Fatal("expected error for invalid options_json, got nil")
	}
}

func TestUpdateContent_InvalidOptionsJSONReturnsError(t *testing.T) {
	mock := &mockClient{
		putFunc: func(path string, body any) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	handler := tools.UpdateContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{
		"public_id":    "abc123",
		"options_json": "{not json",
	}))
	if err == nil {
		t.Fatal("expected error for invalid options_json, got nil")
	}
}
