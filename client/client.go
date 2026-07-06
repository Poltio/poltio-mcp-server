package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"
)

const defaultBaseURL = "https://api-stage.poltio.com"

type PoltioClient struct {
	baseURL    string
	token      string
	mu         sync.RWMutex
	orgID      string
	httpClient *http.Client
}

func New(token string) *PoltioClient {
	return newClient(token, defaultBaseURL)
}

// NewForTest creates a client pointing at a custom base URL. Use in tests only.
func NewForTest(token, orgID, baseURL string) *PoltioClient {
	c := newClient(token, baseURL)
	c.orgID = orgID
	return c
}

func newClient(token, baseURL string) *PoltioClient {
	return &PoltioClient{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *PoltioClient) SetOrgID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orgID = id
}

// GetOrganizations fetches the organizations the current user belongs to,
// ordered by last_used_at desc. It uses /platform/account/profile which does
// not require an Organization-Id header, making it safe to call at startup.
func (c *PoltioClient) GetOrganizations() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/platform/account/profile", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	data, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var profile struct {
		Organizations json.RawMessage `json:"organizations"`
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	return profile.Organizations, nil
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

func (c *PoltioClient) Put(path string, body any) ([]byte, error) {
	return c.sendWithBody(http.MethodPut, path, body)
}

func (c *PoltioClient) Post(path string, body any) ([]byte, error) {
	return c.sendWithBody(http.MethodPost, path, body)
}

func (c *PoltioClient) Delete(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	return c.do(req)
}

func (c *PoltioClient) sendWithBody(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
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
	c.mu.RLock()
	orgID := c.orgID
	c.mu.RUnlock()
	req.Header.Set("Authorization", "Bearer "+c.token)
	if orgID != "" {
		req.Header.Set("Organization-Id", orgID)
	}
}

func (c *PoltioClient) PostFormFile(path, fieldName, filename string, content []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(content); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.setHeaders(req)
	return c.do(req)
}

func (c *PoltioClient) PostFormMultipart(path string, fields map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.setHeaders(req)
	return c.do(req)
}

func (c *PoltioClient) PostFormFileFields(path, fieldName, filename string, content []byte, fields map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(content); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.setHeaders(req)
	return c.do(req)
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
