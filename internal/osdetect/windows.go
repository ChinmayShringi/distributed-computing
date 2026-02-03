//go:build windows

package osdetect

import (
	"os"
	"os/exec"
	"strings"
)

// detectPlatform populates SystemInfo for the current platform
func detectPlatform(info *SystemInfo) error {
	return detectWindows(info)
}

// detectWindows populates SystemInfo for Windows
func detectWindows(info *SystemInfo) error {
	info.Platform = PlatformWindows
	info.Distro = DistroNone

	// Detect package manager
	if _, err := exec.LookPath("winget"); err == nil {
		info.PackageManager = PMWinget
	} else if _, err := exec.LookPath("choco"); err == nil {
		info.PackageManager = PMChoco
	} else {
		info.PackageManager = PMUnknown
	}

	// Get Windows version using wmic
	if out, err := exec.Command("wmic", "os", "get", "Version", "/value").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "Version=") {
				info.DistroVersion = strings.TrimPrefix(line, "Version=")
				info.DistroVersion = strings.TrimSpace(info.DistroVersion)
				break
			}
		}
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	return nil
}

// IsWingetAvailable checks if winget is available
func IsWingetAvailable() bool {
	_, err := exec.LookPath("winget")
	return err == nil
}

// IsChocoAvailable checks if Chocolatey is available
func IsChocoAvailable() bool {
	_, err := exec.LookPath("choco")
	return err == nil
}
