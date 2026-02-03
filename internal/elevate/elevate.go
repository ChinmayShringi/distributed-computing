// Package elevate handles privilege elevation (sudo/admin)
package elevate

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// IsRoot returns true if running with root/admin privileges
func IsRoot() bool {
	if runtime.GOOS == "windows" {
		return isWindowsAdmin()
	}
	return os.Geteuid() == 0
}

// RequireRoot checks if running as root and returns an error if not
func RequireRoot() error {
	if !IsRoot() {
		return fmt.Errorf("this command requires root privileges. Please run with sudo")
	}
	return nil
}

// ReexecWithSudo re-executes the current program with sudo
// This replaces the current process
func ReexecWithSudo() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("sudo not available on Windows, please run as Administrator")
	}

	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("sudo not found: %w", err)
	}

	// Get current executable
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build args: sudo <executable> <original args...>
	args := append([]string{sudoPath, executable}, os.Args[1:]...)

	// Replace current process with sudo
	return syscall.Exec(sudoPath, args, os.Environ())
}

// RunWithSudo runs a command with sudo if not already root
func RunWithSudo(name string, args ...string) *exec.Cmd {
	if IsRoot() {
		return exec.Command(name, args...)
	}

	// Prepend sudo
	sudoArgs := append([]string{name}, args...)
	return exec.Command("sudo", sudoArgs...)
}

// EnsureRoot checks if running as root, and if not, prompts to re-execute with sudo
func EnsureRoot(allowReexec bool) error {
	if IsRoot() {
		return nil
	}

	if !allowReexec {
		return fmt.Errorf("this command requires root privileges. Please run with sudo")
	}

	fmt.Println("This command requires root privileges. Requesting sudo access...")
	return ReexecWithSudo()
}

// isWindowsAdmin checks if running as Windows admin (stub for now)
func isWindowsAdmin() bool {
	// On Windows, we'd check for admin privileges
	// This is a simplified check
	return false
}
