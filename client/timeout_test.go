package client

import "testing"

// The HTTP client must stay bounded. Token validation runs in the server's
// request path, so an unbounded call against an API that hangs rather than
// refuses would accumulate requests indefinitely while the health endpoint
// still answers 200 — the pod keeps being sent traffic it cannot serve.
func TestHTTPClientHasTimeout(t *testing.T) {
	if got := newClient("tok", "https://example.com").httpClient.Timeout; got <= 0 {
		t.Fatalf("httpClient.Timeout = %v, want a bounded timeout", got)
	}
}
