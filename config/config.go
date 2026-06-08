package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the parsed configuration values needed by the tray app.
type Config struct {
	MixedPort      int    // default: 7890
	ControllerPort int    // default: 9090
	ControllerAddr string // e.g. "127.0.0.1:9090"
	ConfigError    string // non-empty if config parse had issues
}

// Load reads the mihomo config.yaml and extracts the relevant fields.
// It never returns an error — on any failure, defaults are used and
// ConfigError is populated with a description.
func Load() *Config {
	cfg := &Config{
		MixedPort:      7890,
		ControllerPort: 9090,
		ControllerAddr: "127.0.0.1:9090",
	}

	home, err := os.UserHomeDir()
	if err != nil {
		cfg.ConfigError = fmt.Sprintf("无法获取用户目录: %v", err)
		return cfg
	}

	configPath := filepath.Join(home, ".config", "mihomo", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg.ConfigError = fmt.Sprintf("无法读取配置文件 (%s): %v", configPath, err)
		return cfg
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		cfg.ConfigError = fmt.Sprintf("配置文件解析失败: %v", err)
		return cfg
	}

	// Parse mixed-port
	if mp, ok := raw["mixed-port"]; ok {
		switch v := mp.(type) {
		case int:
			cfg.MixedPort = v
		case int64:
			cfg.MixedPort = int(v)
		case float64:
			cfg.MixedPort = int(v)
		case string:
			if p, err := strconv.Atoi(v); err == nil {
				cfg.MixedPort = p
			}
		}
	}

	// Parse external-controller (format: "127.0.0.1:9090" or "0.0.0.0:9090")
	if ec, ok := raw["external-controller"]; ok {
		if addr, ok := ec.(string); ok && addr != "" {
			cfg.ControllerAddr = addr
			// Extract port from address
			parts := strings.Split(addr, ":")
			if len(parts) >= 2 {
				if port, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
					cfg.ControllerPort = port
				}
			}
		}
	}

	return cfg
}
