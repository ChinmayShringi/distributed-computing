//go:build darwin

package osdetect

import (
	"os"
	"os/exec"
	"strings"
)

// detectPlatform populates SystemInfo for the current platform
func detectPlatform(info *SystemInfo) error {
	return detectDarwin(info)
}

// detectDarwin populates SystemInfo for macOS
func detectDarwin(info *SystemInfo) error {
	info.Platform = PlatformDarwin
	info.Distro = DistroNone
	info.PackageManager = PMBrew

	// Get macOS version using sw_vers
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		info.DistroVersion = strings.TrimSpace(string(out))
	}

	// Get kernel version
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		info.KernelVersion = strings.TrimSpace(string(out))
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	return nil
}

// IsBrewInstalled checks if Homebrew is installed
func IsBrewInstalled() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// GetBrewPrefix returns the Homebrew prefix path
func GetBrewPrefix() string {
	if out, err := exec.Command("brew", "--prefix").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	// Default paths
	if _, err := os.Stat("/opt/homebrew"); err == nil {
		return "/opt/homebrew" // Apple Silicon
	}
	return "/usr/local" // Intel
}
