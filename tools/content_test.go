package tools_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Poltio/poltio-mcp-server/tools"
)

// mockClient implements tools.ContentClient for testing.
type mockClient struct {
	getFunc  func(path string, query url.Values) ([]byte, error)
	postFunc func(path string, body any) ([]byte, error)
}

func (m *mockClient) Get(path string, query url.Values) ([]byte, error) {
	return m.getFunc(path, query)
}

func (m *mockClient) Post(path string, body any) ([]byte, error) {
	return m.postFunc(path, body)
}

func callRequest(args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

// --- list_content ---

func TestListContent_CallsCorrectPath(t *testing.T) {
	var gotPath string
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			gotPath = path
			return []byte(`{"data":[]}`), nil
		},
	}
	handler := tools.ListContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/platform/content" {
		t.Errorf("path: want /platform/content, got %q", gotPath)
	}
}

func TestListContent_ForwardsQueryParams(t *testing.T) {
	var gotQuery url.Values
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			gotQuery = query
			return []byte(`{"data":[]}`), nil
		},
	}
	handler := tools.ListContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{
		"page":     float64(2),
		"per_page": float64(5),
		"type":     "poll",
		"q":        "test search",
		"order":    "vote_count",
		"sort":     "asc",
	}))
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string]string{
		"page":     "2",
		"per_page": "5",
		"type":     "poll",
		"q":        "test search",
		"order":    "vote_count",
		"sort":     "asc",
	}
	for k, want := range checks {
		if got := gotQuery.Get(k); got != want {
			t.Errorf("query[%s]: want %q, got %q", k, want, got)
		}
	}
}

func TestListContent_PropagatesClientError(t *testing.T) {
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			return nil, errors.New("network error")
		},
	}
	handler := tools.ListContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- get_content ---

func TestGetContent_UsesPublicIDInPath(t *testing.T) {
	var gotPath string
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			gotPath = path
			return []byte(`{"content":{}}`), nil
		},
	}
	handler := tools.GetContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{"public_id": "abc123"}))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/platform/content/abc123" {
		t.Errorf("path: want /platform/content/abc123, got %q", gotPath)
	}
}

func TestGetContent_MissingPublicIDReturnsError(t *testing.T) {
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	handler := tools.GetContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{}))
	if err == nil {
		t.Fatal("expected error for missing public_id, got nil")
	}
}

// --- create_content ---

func TestCreateContent_PostsToCorrectPath(t *testing.T) {
	var gotPath string
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			gotPath = path
			return []byte(`{"public_id":"new1"}`), nil
		},
	}
	handler := tools.CreateContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{
		"type":  "poll",
		"title": "My Poll",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/platform/content" {
		t.Errorf("path: want /platform/content, got %q", gotPath)
	}
}

func TestCreateContent_MissingTypeReturnsError(t *testing.T) {
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	handler := tools.CreateContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{"title": "No type"}))
	if err == nil {
		t.Fatal("expected error for missing type, got nil")
	}
}

func TestCreateContent_MissingTitleReturnsError(t *testing.T) {
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	handler := tools.CreateContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{"type": "poll"}))
	if err == nil {
		t.Fatal("expected error for missing title, got nil")
	}
}

// --- publish_content ---

func TestPublishContent_PostsToCorrectPath(t *testing.T) {
	var gotPath string
	mock := &mockClient{
		postFunc: func(path string, body any) ([]byte, error) {
			gotPath = path
			return []byte(`{"public_id":"xyz"}`), nil
		},
	}
	handler := tools.PublishContent(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{"public_id": "xyz"}))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/platform/content/xyz/publish" {
		t.Errorf("path: want /platform/content/xyz/publish, got %q", gotPath)
	}
}

// --- list_drafts ---

func TestListDrafts_CallsCorrectPath(t *testing.T) {
	var gotPath string
	mock := &mockClient{
		getFunc: func(path string, query url.Values) ([]byte, error) {
			gotPath = path
			return []byte(`{"data":[]}`), nil
		},
	}
	handler := tools.ListDrafts(mock)
	_, err := handler(context.Background(), callRequest(map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/platform/content/drafts" {
		t.Errorf("path: want /platform/content/drafts, got %q", gotPath)
	}
}
