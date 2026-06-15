package tools

import "net/http"

// SetImageHTTPClient swaps the client used to fetch image_url and returns a
// function that restores the original. Tests that exercise the URL path against
// a loopback httptest server use this to bypass the SSRF guard, which otherwise
// (correctly) rejects loopback targets.
func SetImageHTTPClient(c *http.Client) (restore func()) {
	old := imageHTTPClient
	imageHTTPClient = c
	return func() { imageHTTPClient = old }
}
