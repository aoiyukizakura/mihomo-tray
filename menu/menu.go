package menu

import (
	"fmt"
	"os"

	"github.com/getlantern/systray"

	"mihomo-tray/icons"
	"mihomo-tray/mihomo"
	"mihomo-tray/registry"
	"mihomo-tray/state"
)

// menuItems holds references to all dynamic menu items.
type menuItems struct {
	statusDisplay *systray.MenuItem

	systemProxy *systray.MenuItem

	modeSubmenu *systray.MenuItem
	modeRule    *systray.MenuItem
	modeGlobal  *systray.MenuItem
	modeDirect  *systray.MenuItem

	tunMode   *systray.MenuItem
	autoStart *systray.MenuItem
}

// Build creates the full tray menu and returns references to dynamic items.
// It also wires up all click handlers.
func Build(client *mihomo.Client, proxyPort int, exePath string) *menuItems {
	mi := &menuItems{}

	// --- Status display (disabled) ---
	mi.statusDisplay = systray.AddMenuItem("Mihomo 状态: 检测中...", "")
	mi.statusDisplay.Disable()

	systray.AddSeparator()

	// --- System proxy toggle ---
	mi.systemProxy = systray.AddMenuItemCheckbox("系统代理", "开启/关闭 Windows 系统代理", false)
	go func() {
		for range mi.systemProxy.ClickedCh {
			if mi.systemProxy.Checked() {
				// Currently checked → disable
				if err := registry.DisableSystemProxy(); err == nil {
					mi.systemProxy.Uncheck()
				}
			} else {
				// Currently unchecked → enable
				if err := registry.EnableSystemProxy(proxyPort); err == nil {
					mi.systemProxy.Check()
				}
			}
		}
	}()

	// --- Mode submenu ---
	mi.modeSubmenu = systray.AddMenuItem("代理模式", "切换 Mihomo 代理模式")
	mi.modeRule = mi.modeSubmenu.AddSubMenuItemCheckbox("Rule (规则)", "规则模式", false)
	mi.modeGlobal = mi.modeSubmenu.AddSubMenuItemCheckbox("Global (全局)", "全局模式", false)
	mi.modeDirect = mi.modeSubmenu.AddSubMenuItemCheckbox("Direct (直连)", "直连模式", false)

	// Mode switching with radio-button behavior.
	go func() {
		for range mi.modeRule.ClickedCh {
			if err := client.SetMode("rule"); err == nil {
				setModeChecked(mi, "rule")
			}
		}
	}()
	go func() {
		for range mi.modeGlobal.ClickedCh {
			if err := client.SetMode("global"); err == nil {
				setModeChecked(mi, "global")
			}
		}
	}()
	go func() {
		for range mi.modeDirect.ClickedCh {
			if err := client.SetMode("direct"); err == nil {
				setModeChecked(mi, "direct")
			}
		}
	}()

	// --- TUN mode toggle ---
	mi.tunMode = systray.AddMenuItemCheckbox("TUN 模式", "开启/关闭 TUN 虚拟网卡模式", false)
	go func() {
		for range mi.tunMode.ClickedCh {
			enable := !mi.tunMode.Checked()
			if err := client.SetTUN(enable); err == nil {
				if enable {
					mi.tunMode.Check()
				} else {
					mi.tunMode.Uncheck()
				}
			}
		}
	}()

	systray.AddSeparator()

	// --- Auto-start toggle ---
	mi.autoStart = systray.AddMenuItemCheckbox("开机自启动", "允许开机时自动启动", registry.IsAutoStartEnabled())
	go func() {
		for range mi.autoStart.ClickedCh {
			enabled := !mi.autoStart.Checked()
			if err := registry.SetAutoStart(enabled, exePath); err == nil {
				if enabled {
					mi.autoStart.Check()
				} else {
					mi.autoStart.Uncheck()
				}
			}
		}
	}()

	// --- Reload config ---
	reloadItem := systray.AddMenuItem("重载配置", "重新加载 Mihomo 配置文件")
	go func() {
		for range reloadItem.ClickedCh {
			_ = client.ReloadConfig()
		}
	}()

	systray.AddSeparator()

	// --- Exit submenu ---
	exitSubmenu := systray.AddMenuItem("退出", "退出程序")
	exitAndStop := exitSubmenu.AddSubMenuItem("退出并停止 Mihomo", "退出程序并结束 mihomo.exe 进程")
	exitOnly := exitSubmenu.AddSubMenuItem("仅退出程序", "退出程序但保持 mihomo.exe 运行")

	go func() {
		for range exitAndStop.ClickedCh {
			_ = mihomo.Stop()
			systray.Quit()
		}
	}()
	go func() {
		for range exitOnly.ClickedCh {
			systray.Quit()
		}
	}()

	return mi
}

// setModeChecked unchecks all mode items and checks only the active one.
func setModeChecked(mi *menuItems, mode string) {
	mi.modeRule.Uncheck()
	mi.modeGlobal.Uncheck()
	mi.modeDirect.Uncheck()

	switch mode {
	case "rule":
		mi.modeRule.Check()
	case "global":
		mi.modeGlobal.Check()
	case "direct":
		mi.modeDirect.Check()
	}
}

// Refresh updates all menu item states and the tray icon based on the current AppState.
func Refresh(mi *menuItems, s state.AppState, exePath string) {
	// Update status line.
	if s.MihomoRunning {
		modeText := s.MihomoMode
		if modeText == "" || modeText == "unknown" {
			modeText = "未知"
		}
		mi.statusDisplay.SetTitle(fmt.Sprintf("Mihomo 状态: 运行中 (%s)", modeText))
	} else {
		mi.statusDisplay.SetTitle("Mihomo 状态: 未运行")
	}

	// Update system proxy checkbox.
	if s.SystemProxy {
		mi.systemProxy.Check()
	} else {
		mi.systemProxy.Uncheck()
	}

	// Update mode radio buttons.
	if s.MihomoRunning {
		mi.modeSubmenu.Enable()
		setModeChecked(mi, s.MihomoMode)
	} else {
		mi.modeSubmenu.Disable()
	}

	// Update TUN mode checkbox.
	if s.MihomoRunning {
		mi.tunMode.Enable()
		if s.TunMode {
			mi.tunMode.Check()
		} else {
			mi.tunMode.Uncheck()
		}
	} else {
		mi.tunMode.Disable()
	}

	// Update auto-start checkbox.
	if registry.IsAutoStartEnabled() {
		mi.autoStart.Check()
	} else {
		mi.autoStart.Uncheck()
	}

	// Update tray icon.
	updateIcon(s)
}

// updateIcon sets the tray icon based on state.
func updateIcon(s state.AppState) {
	if s.TunMode {
		systray.SetIcon(icons.Blue)
	} else if s.SystemProxy && s.MihomoRunning {
		systray.SetIcon(icons.Green)
	} else {
		systray.SetIcon(icons.Gray)
	}
}

// GetExePath returns the full path to the current executable.
func GetExePath() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	return p
}
