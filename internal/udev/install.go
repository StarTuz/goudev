package udev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// InstallResult is the result of an install attempt.
type InstallResult struct {
	Installed  bool   // true if file was written and udev reloaded
	RulePath   string // installed rule file path
	BackupPath string // if backup was made
	Err        error
}

// Install writes rules to /etc/udev/rules.d/, backs up existing file, and runs udevadm.
// If udev reload fails, the new file is removed and backup is restored.
func Install(fileName, rulesContent string) InstallResult {
	dir := RulesDir
	fullPath := filepath.Join(dir, fileName)

	if err := ValidateRules(rulesContent); err != nil {
		return InstallResult{Err: fmt.Errorf("validation failed: %w", err)}
	}
	if _, err := exec.LookPath("udevadm"); err != nil {
		return InstallResult{Err: fmt.Errorf("udevadm not found in PATH (required to reload udev rules). Install udev or systemd.")}
	}

	// Check we can write (effectively root check for /etc)
	info, err := os.Stat(dir)
	if err != nil {
		return InstallResult{Err: fmt.Errorf("cannot access %s: %w", dir, err)}
	}
	if !info.IsDir() {
		return InstallResult{Err: fmt.Errorf("%s is not a directory", dir)}
	}
	// Try to open for write to detect permission
	testPath := filepath.Join(dir, ".goudev-write-test")
	if err := os.WriteFile(testPath, []byte{}, 0644); err != nil {
		_ = os.Remove(testPath)
		return InstallResult{
			Err: fmt.Errorf("cannot write to %s (need root?). Run with: sudo goudev install", dir),
		}
	}
	_ = os.Remove(testPath)

	var backupPath string
	if _, err := os.Stat(fullPath); err == nil {
		backupPath = fullPath + "." + time.Now().Format("20060102-150405.bak")
		if err := copyFile(fullPath, backupPath); err != nil {
			return InstallResult{Err: fmt.Errorf("backup failed: %w", err)}
		}
	}

	if err := os.WriteFile(fullPath, []byte(rulesContent+"\n"), 0644); err != nil {
		return InstallResult{Err: fmt.Errorf("write rules: %w", err)}
	}

	if err := reloadUdev(); err != nil {
		// Restore previous state: remove new file, restore backup
		_ = os.Remove(fullPath)
		if backupPath != "" {
			_ = copyFile(backupPath, fullPath)
		}
		return InstallResult{
			Err: fmt.Errorf("udev reload failed (rules were reverted): %w", err),
		}
	}

	return InstallResult{
		Installed:  true,
		RulePath:   fullPath,
		BackupPath: backupPath,
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func reloadUdev() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "udevadm", "control", "--reload-rules")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("udevadm control: %w: %s", err, string(out))
	}
	cmd2 := exec.CommandContext(ctx, "udevadm", "trigger")
	if out, err := cmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("udevadm trigger: %w: %s", err, string(out))
	}
	return nil
}
