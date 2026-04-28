package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// newRequest creates an authenticated HTTP request with the client's base URL.
func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	u := c.BaseURL + path
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	return req, nil
}

// Client is a thin HTTP wrapper around the rememberize REST API.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// NewClient creates a Client from the loaded config. Errors if either
// the API key or server URL is missing — callers expect to make
// authenticated API calls and a naked request would either get rejected
// by the server (best case) or, worse, hit an unauthenticated endpoint
// the server didn't mean to expose. Pre-flighting on the CLI side
// surfaces the misconfiguration before the network call.
func NewClient() (*Client, error) {
	return newClientFromConfig(loadConfig())
}

// newClientFromConfig is the testable seam: takes a *Config explicitly
// instead of calling loadConfig(), so unit tests can construct edge
// cases without touching ~/.rememberize/config.toml.
func newClientFromConfig(cfg *Config) (*Client, error) {
	if cfg.Auth.APIKey == "" {
		return nil, fmt.Errorf("no API key configured. Run 'rememberize pair <code>' to authenticate — get a code from https://rememberize.app/app/connections/new — or set REMEMBERIZE_API_KEY=<key> for scripted use")
	}
	if cfg.Auth.APIURL == "" {
		return nil, fmt.Errorf("no server URL configured. Run 'rememberize pair <code>' to authenticate, or 'rememberize config set auth.api_url https://platform.rememberize.app', or set REMEMBERIZE_API_URL=<url> for scripted use")
	}
	return &Client{
		BaseURL: cfg.Auth.APIURL,
		APIKey:  cfg.Auth.APIKey,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// ---------------------------------------------------------------------------
// Response types (matching API JSON shapes)
// ---------------------------------------------------------------------------

// Memory mirrors the server-side memory type.
type Memory struct {
	ID        string  `json:"id"`
	Namespace string  `json:"namespace"`
	Type      string  `json:"type"`
	Content   string  `json:"content"`
	Metadata  *string `json:"metadata,omitempty"`
	SourceID  *string `json:"source_id,omitempty"`
	Version   int     `json:"version"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// SearchResult wraps a memory with its relevance score.
type SearchResult struct {
	Memory      Memory  `json:"memory"`
	Score       float64 `json:"score"`
	FTSScore    float64 `json:"fts_score,omitempty"`
	VectorScore float64 `json:"vector_score,omitempty"`
}

// Namespace represents a memory namespace.
type Namespace struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// Connection represents an external connection.
type Connection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	APIKey      string `json:"api_key,omitempty"`
	Permissions string `json:"permissions"`
	LastSeen    string `json:"last_seen,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID           string  `json:"id"`
	ConnectionID string  `json:"connection_id"`
	Action       string  `json:"action"`
	MemoryID     *string `json:"memory_id,omitempty"`
	Namespace    *string `json:"namespace,omitempty"`
	Metadata     *string `json:"metadata,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// apiError is returned when the server responds with an error body.
type apiError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *apiError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Body)
}

// do executes an HTTP request with auth and JSON headers, returning the raw body.
func (c *Client) do(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	u := c.BaseURL + path

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	logger.Debug("http request", "method", method, "url", u)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		logger.Debug("http request failed", "method", method, "url", u, "err", err)
		return nil, 0, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	logger.Debug("http response", "method", method, "url", u, "status", resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to extract error message from JSON body.
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		msg := ""
		if json.Unmarshal(respBody, &errResp) == nil {
			if errResp.Error != "" {
				msg = errResp.Error
			}
			if errResp.Message != "" && msg != "" {
				msg = msg + ": " + errResp.Message
			} else if errResp.Message != "" {
				msg = errResp.Message
			}
		}
		return nil, resp.StatusCode, &apiError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
			Message:    msg,
		}
	}

	return respBody, resp.StatusCode, nil
}

// ---------------------------------------------------------------------------
// Memory methods
// ---------------------------------------------------------------------------

// CreateMemoryRequest is the request body for creating a memory.
type CreateMemoryRequest struct {
	Content   string  `json:"content"`
	Namespace string  `json:"namespace,omitempty"`
	Type      string  `json:"type,omitempty"`
	Metadata  *string `json:"metadata,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// CreateMemory creates a new memory via POST /api/memories.
func (c *Client) CreateMemory(content, namespace, memType string, metadata, expiresAt *string) (*Memory, error) {
	req := CreateMemoryRequest{
		Content:   content,
		Namespace: namespace,
		Type:      memType,
		Metadata:  metadata,
		ExpiresAt: expiresAt,
	}

	body, _, err := c.do("POST", "/api/memories", req)
	if err != nil {
		return nil, fmt.Errorf("create memory: %w", err)
	}

	var mem Memory
	if err := json.Unmarshal(body, &mem); err != nil {
		return nil, fmt.Errorf("parse create memory response: %w", err)
	}

	return &mem, nil
}

// ListMemories lists memories via GET /api/memories.
func (c *Client) ListMemories(namespace, memType string, limit, offset int) ([]Memory, error) {
	params := url.Values{}
	if namespace != "" {
		params.Set("namespace", namespace)
	}
	if memType != "" {
		params.Set("type", memType)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	path := "/api/memories"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, _, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}

	var resp struct {
		Memories []Memory `json:"memories"`
		Count    int      `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list memories response: %w", err)
	}

	return resp.Memories, nil
}

// GetMemory retrieves a single memory via GET /api/memories/:id.
func (c *Client) GetMemory(id string) (*Memory, error) {
	body, _, err := c.do("GET", "/api/memories/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}

	var mem Memory
	if err := json.Unmarshal(body, &mem); err != nil {
		return nil, fmt.Errorf("parse get memory response: %w", err)
	}

	return &mem, nil
}

// DeleteMemory deletes a memory via DELETE /api/memories/:id.
func (c *Client) DeleteMemory(id string) error {
	_, _, err := c.do("DELETE", "/api/memories/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

// SearchMemories searches memories via GET /api/memories/search.
func (c *Client) SearchMemories(query, namespace, memType string, limit int, semantic bool) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	if namespace != "" {
		params.Set("namespace", namespace)
	}
	if memType != "" {
		params.Set("type", memType)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if semantic {
		params.Set("semantic", "true")
	}

	path := "/api/memories/search?" + params.Encode()

	body, _, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}

	var resp struct {
		Results []SearchResult `json:"results"`
		Count   int            `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	return resp.Results, nil
}

// ---------------------------------------------------------------------------
// Namespace methods
// ---------------------------------------------------------------------------

// ListNamespaces lists all namespaces via GET /api/namespaces.
func (c *Client) ListNamespaces() ([]Namespace, error) {
	body, _, err := c.do("GET", "/api/namespaces", nil)
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	var resp struct {
		Namespaces []Namespace `json:"namespaces"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list namespaces response: %w", err)
	}

	return resp.Namespaces, nil
}

// ---------------------------------------------------------------------------
// Connection methods
// ---------------------------------------------------------------------------

// ListConnections lists all connections via GET /api/connections.
func (c *Client) ListConnections() ([]Connection, error) {
	body, _, err := c.do("GET", "/api/connections", nil)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	var resp struct {
		Connections []Connection `json:"connections"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list connections response: %w", err)
	}

	return resp.Connections, nil
}

// ---------------------------------------------------------------------------
// Audit methods
// ---------------------------------------------------------------------------

// ListAudit lists audit entries via GET /api/audit.
func (c *Client) ListAudit(limit int, action, connectionID string) ([]AuditEntry, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if action != "" {
		params.Set("action", action)
	}
	if connectionID != "" {
		params.Set("connection_id", connectionID)
	}

	path := "/api/audit"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, _, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}

	var resp struct {
		Entries []AuditEntry `json:"entries"`
		Count   int          `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list audit response: %w", err)
	}

	return resp.Entries, nil
}
