package mihomo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with the mihomo REST API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a Client for the given controller port.
func NewClient(port int) *Client {
	return &Client{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		HTTPClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// IsAlive checks whether the mihomo API is reachable by hitting GET /version.
func (c *Client) IsAlive() bool {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetMode returns the current proxy mode ("rule", "global", "direct", or "unknown").
func (c *Client) GetMode() string {
	type configResponse struct {
		Mode string `json:"mode"`
	}

	resp, err := c.HTTPClient.Get(c.BaseURL + "/configs")
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "unknown"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "unknown"
	}

	var cr configResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "unknown"
	}

	if cr.Mode != "" {
		return cr.Mode
	}
	return "unknown"
}

// SetMode switches the proxy mode. mode should be "rule", "global", or "direct".
func (c *Client) SetMode(mode string) error {
	payload := map[string]string{"mode": mode}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.BaseURL+"/configs", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 返回错误 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SetTUN enables or disables the TUN stack on the mihomo kernel.
func (c *Client) SetTUN(enable bool) error {
	payload := map[string]interface{}{
		"tun": map[string]interface{}{
			"enable": enable,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.BaseURL+"/configs", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("TUN 切换请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TUN 切换失败 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ReloadConfig triggers a config reload on the mihomo kernel.
func (c *Client) ReloadConfig() error {
	req, err := http.NewRequest(http.MethodPut, c.BaseURL+"/configs?force=true", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("重载配置请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("重载配置失败 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// HasTUN checks whether TUN mode is enabled by examining the /configs response
// for a tun stack entry.
func (c *Client) HasTUN() bool {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/configs")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// Look for "tun" in the mode or as a tun key in the response.
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}

	// Check if mode contains "tun"
	if mode, ok := raw["mode"].(string); ok && mode == "tun" {
		return true
	}

	// Check for tun configuration at top level
	if tun, ok := raw["tun"]; ok && tun != nil {
		if tunMap, ok := tun.(map[string]interface{}); ok {
			if enable, ok := tunMap["enable"]; ok {
				if b, ok := enable.(bool); ok && b {
					return true
				}
			}
			// If "tun" key exists and has no explicit "enable: false", assume enabled
			if _, hasEnable := tunMap["enable"]; !hasEnable {
				return true
			}
		} else {
			// tun key exists with non-map value — assume enabled
			return true
		}
	}

	return false
}
