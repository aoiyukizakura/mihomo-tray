package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/getlantern/systray"

	"mihomo-tray/config"
	"mihomo-tray/menu"
	"mihomo-tray/mihomo"
	"mihomo-tray/state"
)

var (
	shell32        = syscall.NewLazyDLL("shell32.dll")
	procShellExecW = shell32.NewProc("ShellExecuteW")
)

const (
	SW_HIDE = 0
)

func main() {
	// If not running as admin, re-launch with elevation.
	if !isAdmin() {
		relaunchAsAdmin()
		return
	}

	// Load configuration (always succeeds — uses defaults on error).
	cfg := config.Load()

	// Track startup errors for the tooltip.
	var startupErr string

	// Ensure mihomo is running.
	if !mihomo.IsRunning() {
		if err := mihomo.Start(); err != nil {
			startupErr = err.Error()
		}
	}

	// Create API client.
	client := mihomo.NewClient(cfg.ControllerPort)

	// Get our own path for auto-start registration.
	exePath := menu.GetExePath()

	// Start systray. onReady is called on a background goroutine;
	// onExit is called when systray.Quit() is invoked.
	systray.Run(
		func() { onReady(cfg, client, exePath, startupErr) },
		func() { onExit() },
	)
}

func onReady(cfg *config.Config, client *mihomo.Client, exePath string, startupErr string) {
	// Set initial tooltip.
	systray.SetTooltip("Mihomo Tray")

	// Build the menu.
	mi := menu.Build(client, cfg.MixedPort, exePath)

	// Start state poller.
	poller := state.NewPoller(client)
	poller.Start()

	// Subscribe to state changes and refresh the menu.
	stateCh := poller.Subscribe()
	// Do an immediate refresh with the initial state.
	menu.Refresh(mi, poller.GetState(), exePath)

	go func() {
		for s := range stateCh {
			menu.Refresh(mi, s, exePath)
		}
	}()

	// Surface any startup or config errors in the tooltip.
	tip := "Mihomo Tray"
	if startupErr != "" {
		tip += " - ⚠ " + startupErr
	} else if cfg.ConfigError != "" {
		tip += " - ⚠ " + cfg.ConfigError
	}
	systray.SetTooltip(tip)
}

func onExit() {
	// Cleanup is minimal — mihomo keeps running unless user chose "退出并停止".
	// The polling goroutine will stop when the process exits.
}

// isAdmin checks whether the current process is running with administrator
// privileges by trying to open \\.\PHYSICALDRIVE0, which requires admin rights.
func isAdmin() bool {
	f, err := os.Open(`\\.\PHYSICALDRIVE0`)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// relaunchAsAdmin re-executes the current binary with the "runas" verb,
// which triggers a UAC elevation prompt.
func relaunchAsAdmin() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法获取可执行文件路径: %v\n", err)
		os.Exit(1)
	}

	exePtr, _ := syscall.UTF16PtrFromString(exePath)
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	cwdPtr, _ := syscall.UTF16PtrFromString("")

	// ShellExecuteW(hwnd, lpOperation, lpFile, lpParameters, lpDirectory, nShowCmd)
	procShellExecW.Call(
		0,                              // hwnd = NULL
		uintptr(unsafe.Pointer(verbPtr)), // lpOperation = "runas"
		uintptr(unsafe.Pointer(exePtr)),   // lpFile = this exe
		0,                              // lpParameters = NULL
		uintptr(unsafe.Pointer(cwdPtr)),   // lpDirectory = current dir
		SW_HIDE,                        // nShowCmd = SW_HIDE (but UAC dialog will show)
	)

	// Exit the non-elevated instance.
	os.Exit(0)
}
