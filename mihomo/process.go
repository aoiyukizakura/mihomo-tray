package mihomo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// IsRunning checks whether a mihomo.exe process is currently running.
func IsRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq mihomo.exe", "/NH")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "mihomo.exe")
}

// findMihomo locates mihomo.exe by searching, in order:
//  1. PATH (works for scoop, manual installs added to PATH)
//  2. Same directory as the current executable (portable layout)
//  3. Common scoop shims path
//
// Returns the full path, or an error if not found.
func findMihomo() (string, error) {
	// 1. Look up in PATH first — covers scoop shims and any PATH install.
	if p, err := exec.LookPath("mihomo.exe"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("mihomo"); err == nil {
		return p, nil
	}

	// 2. Check the executable's own directory (portable layout).
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "mihomo.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// 3. Check common scoop shims directory.
	if home, err := os.UserHomeDir(); err == nil {
		scoopShim := filepath.Join(home, "scoop", "shims", "mihomo.exe")
		if _, err := os.Stat(scoopShim); err == nil {
			return scoopShim, nil
		}
	}

	return "", fmt.Errorf("未找到 mihomo.exe：请确保 mihomo 已安装并添加到 PATH 环境变量中（scoop 用户请确认 scoop/shims 在 PATH 中）")
}

// Start launches mihomo.exe in the background with its console window hidden.
// It searches for the binary via PATH, the current directory, and common scoop
// locations. Returns an error if the binary cannot be found or fails to start.
func Start() error {
	mihomoPath, err := findMihomo()
	if err != nil {
		return err
	}

	cmd := exec.Command(mihomoPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	// Detach from parent so mihomo keeps running even if tray exits.
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 mihomo 失败 (%s): %w", mihomoPath, err)
	}

	// Release the process handle so it runs independently.
	go func() {
		cmd.Wait()
	}()

	return nil
}

// Stop kills any running mihomo.exe process.
func Stop() error {
	cmd := exec.Command("taskkill", "/F", "/IM", "mihomo.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
