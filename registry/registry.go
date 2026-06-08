package registry

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	internetSettingsKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	runKey              = `Software\Microsoft\Windows\CurrentVersion\Run`
	autoStartValueName  = `MihomoTray`
)

// EnableSystemProxy turns on the Windows system proxy, pointing it to
// 127.0.0.1:port.
func EnableSystemProxy(port int) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开 Internet Settings 注册表键失败: %w", err)
	}
	defer key.Close()

	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("设置 ProxyEnable 失败: %w", err)
	}
	server := fmt.Sprintf("127.0.0.1:%d", port)
	if err := key.SetStringValue("ProxyServer", server); err != nil {
		return fmt.Errorf("设置 ProxyServer 失败: %w", err)
	}
	// Set ProxyOverride to <local> to bypass local addresses.
	if err := key.SetStringValue("ProxyOverride", "<local>"); err != nil {
		return fmt.Errorf("设置 ProxyOverride 失败: %w", err)
	}
	return nil
}

// DisableSystemProxy turns off the Windows system proxy.
func DisableSystemProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开 Internet Settings 注册表键失败: %w", err)
	}
	defer key.Close()

	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("设置 ProxyEnable 失败: %w", err)
	}
	return nil
}

// IsSystemProxyEnabled returns true if the system proxy is currently turned on.
func IsSystemProxyEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		return false
	}
	return val == 1
}

// SetAutoStart enables or disables auto-start for this application via the
// HKCU Run registry key.
func SetAutoStart(enabled bool, exePath string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开 Run 注册表键失败: %w", err)
	}
	defer key.Close()

	if enabled {
		if err := key.SetStringValue(autoStartValueName, exePath); err != nil {
			return fmt.Errorf("设置开机自启动失败: %w", err)
		}
	} else {
		if err := key.DeleteValue(autoStartValueName); err != nil {
			// If the value doesn't exist, that's fine — not an error.
			if err != registry.ErrNotExist {
				return fmt.Errorf("删除开机自启动失败: %w", err)
			}
		}
	}
	return nil
}

// IsAutoStartEnabled returns true if this app is registered for auto-start.
func IsAutoStartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(autoStartValueName)
	return err == nil
}
