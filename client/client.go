package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const defaultBaseURL = "https://api-stage.poltio.com"

type PoltioClient struct {
	baseURL    string
	token      string
	orgID      string
	httpClient *http.Client
}

func New(token, orgID string) *PoltioClient {
	return newClient(token, orgID, defaultBaseURL)
}

// NewForTest creates a client pointing at a custom base URL. Use in tests only.
func NewForTest(token, orgID, baseURL string) *PoltioClient {
	return newClient(token, orgID, baseURL)
}

func newClient(token, orgID, baseURL string) *PoltioClient {
	return &PoltioClient{
		baseURL:    baseURL,
		token:      token,
		orgID:      orgID,
		httpClient: &http.Client{},
	}
}

func (c *PoltioClient) Get(path string, query url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	return c.do(req)
}

func (c *PoltioClient) Post(path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.setHeaders(req)
	return c.do(req)
}

func (c *PoltioClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Organization-Id", c.orgID)
}

func (c *PoltioClient) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
