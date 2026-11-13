package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"

	"github.com/Poltio/poltio-mcp-server/client"
)

func TestAPIFailureReportsOnly5xxWithoutUserContent(t *testing.T) {
	tr := &sentry.MockTransport{}
	if err := sentry.Init(sentry.ClientOptions{Dsn: "https://k@o1.ingest.sentry.io/1", Transport: tr}); err != nil {
		t.Fatal(err)
	}
	defer sentry.CurrentHub().BindClient(nil)

	status := http.StatusInternalServerError
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"email":"lead@example.com"}`))
	}))
	defer srv.Close()
	c := client.NewForTest("secret-token", "1", srv.URL)

	if _, err := c.Get("/platform/content", url.Values{"q": {"secret-query"}}); err == nil {
		t.Fatal("want an error for a 500")
	}
	events := tr.Events()
	if len(events) != 1 || events[0].Message != "Poltio API 500" || events[0].Tags["path"] != "/platform/content" {
		t.Fatalf("want one 'Poltio API 500' event tagged with the path, got %+v", events)
	}
	b, _ := json.Marshal(events[0])
	for _, leak := range []string{"lead@example.com", "secret-query", "secret-token"} {
		if strings.Contains(string(b), leak) {
			t.Errorf("event leaks %q: %s", leak, b)
		}
	}

	status = http.StatusUnprocessableEntity
	_, _ = c.Get("/platform/content", nil)
	if n := len(tr.Events()); n != 1 {
		t.Fatalf("a 4xx is the caller's: want no new event, got %d events", n)
	}
}
