package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBodyBytes = 4 << 20

// Client 是带超时、Bearer Token 和 JSON 解码保护的轻量 HTTP 客户端。
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Get(path string, out any) error {
	return c.Do(http.MethodGet, path, nil, out)
}

func (c *Client) Post(path string, body any, out any) error {
	return c.Do(http.MethodPost, path, body, out)
}

func (c *Client) Delete(path string, out any) error {
	return c.Do(http.MethodDelete, path, nil, out)
}

func (c *Client) Do(method, path string, body any, out any) error {
	if c.baseURL == "" {
		return fmt.Errorf("base url is required")
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	return decodeMaybeWrapped(data, out)
}

func decodeMaybeWrapped(data []byte, out any) error {
	var wrapped APIResponse
	if err := json.Unmarshal(data, &wrapped); err == nil && (wrapped.Success || wrapped.Error != "" || wrapped.Data != nil) {
		if !wrapped.Success && wrapped.Error != "" {
			return errors.New(wrapped.Error)
		}
		if len(wrapped.Data) == 0 {
			return nil
		}
		if err := json.Unmarshal(wrapped.Data, out); err != nil {
			return fmt.Errorf("decode wrapped response: %w", err)
		}
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
