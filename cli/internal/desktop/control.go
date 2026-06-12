package desktop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const controlFileName = "desktop-control.json"

type ControlFile struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}

type Client struct {
	control ControlFile
	http    *http.Client
}

func DefaultControlFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".nextunnel", controlFileName), nil
}

func NewClient(path string) (*Client, error) {
	if path == "" {
		var err error
		path, err = DefaultControlFilePath()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取桌面端控制文件失败，请先启动桌面端：%w", err)
	}
	var control ControlFile
	if err := json.Unmarshal(data, &control); err != nil {
		return nil, fmt.Errorf("解析桌面端控制文件失败：%w", err)
	}
	if !strings.HasPrefix(control.URL, "http://127.0.0.1:") {
		return nil, fmt.Errorf("桌面端控制地址不安全：%s", control.URL)
	}
	return &Client{
		control: control,
		http:    &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (c *Client) Get(path string, out any) error {
	return c.do(http.MethodGet, path, nil, out)
}

func (c *Client) Post(path string, body any, out any) error {
	return c.do(http.MethodPost, path, body, out)
}

func (c *Client) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.control.URL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.control.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("调用桌面端控制 API 失败：%w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("桌面端控制 API 返回 HTTP %d：%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("解析桌面端响应失败：%w", err)
		}
	}
	return nil
}
