package client_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Poltio/poltio-mcp-server/client"
)

func TestGet_SetsAuthHeaders(t *testing.T) {
	var gotAuth, gotOrgID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotOrgID = r.Header.Get("Organization-Id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok123", "42", srv.URL)
	body, err := c.Get("/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer tok123" {
		t.Errorf("Authorization: want %q, got %q", "Bearer tok123", gotAuth)
	}
	if gotOrgID != "42" {
		t.Errorf("Organization-Id: want %q, got %q", "42", gotOrgID)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body: want %q, got %q", `{"ok":true}`, string(body))
	}
}

func TestGet_WithQueryParams(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok", "1", srv.URL)
	q := url.Values{}
	q.Set("page", "2")
	q.Set("per_page", "5")
	_, err := c.Get("/test", q)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery.Get("page") != "2" {
		t.Errorf("page: want 2, got %q", gotQuery.Get("page"))
	}
	if gotQuery.Get("per_page") != "5" {
		t.Errorf("per_page: want 5, got %q", gotQuery.Get("per_page"))
	}
}

func TestGet_NonOKStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := client.NewForTest("bad", "1", srv.URL)
	_, err := c.Get("/test", nil)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

func TestPost_SendsJSONBody(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"public_id":"abc"}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok", "1", srv.URL)
	payload := map[string]any{"title": "My Poll", "type": "poll"}
	_, err := c.Post("/test", payload)
	if err != nil {
		t.Fatal(err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", gotContentType)
	}
	if !strings.Contains(string(gotBody), "My Poll") {
		t.Errorf("body should contain 'My Poll', got %q", string(gotBody))
	}
}

func TestPost_NilBody_NoContentType(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := client.NewForTest("tok", "1", srv.URL)
	_, err := c.Post("/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type should be empty for nil body, got %q", gotContentType)
	}
}
