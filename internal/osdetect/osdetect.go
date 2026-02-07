// Package osdetect provides operating system and platform detection
package osdetect

import (
	"fmt"
	"runtime"
)

// Platform represents the operating system type
type Platform string

const (
	PlatformDarwin  Platform = "darwin"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
	PlatformUnknown Platform = "unknown"
)

// Distro represents the Linux distribution
type Distro string

const (
	DistroUbuntu  Distro = "ubuntu"
	DistroDebian  Distro = "debian"
	DistroRHEL    Distro = "rhel"
	DistroFedora  Distro = "fedora"
	DistroCentOS  Distro = "centos"
	DistroArch    Distro = "arch"
	DistroUnknown Distro = "unknown"
	DistroNone    Distro = "" // For non-Linux platforms
)

// Arch represents the CPU architecture
type Arch string

const (
	ArchAMD64   Arch = "amd64"
	ArchARM64   Arch = "arm64"
	ArchUnknown Arch = "unknown"
)

// PackageManager represents the system package manager
type PackageManager string

const (
	PMBrew    PackageManager = "brew"   // macOS
	PMAPT     PackageManager = "apt"    // Debian/Ubuntu
	PMDNF     PackageManager = "dnf"    // RHEL/Fedora
	PMPacman  PackageManager = "pacman" // Arch
	PMWinget  PackageManager = "winget" // Windows
	PMChoco   PackageManager = "choco"  // Windows (fallback)
	PMUnknown PackageManager = "unknown"
)

// SystemInfo contains detected system information
type SystemInfo struct {
	Platform       Platform
	Distro         Distro
	DistroVersion  string
	Arch           Arch
	PackageManager PackageManager
	Hostname       string
	KernelVersion  string
}

// String returns a human-readable representation of SystemInfo
func (s SystemInfo) String() string {
	if s.Platform == PlatformLinux {
		return fmt.Sprintf("%s/%s %s (%s)", s.Platform, s.Distro, s.DistroVersion, s.Arch)
	}
	return fmt.Sprintf("%s (%s)", s.Platform, s.Arch)
}

// IsSupported returns true if this platform is supported for installation
func (s SystemInfo) IsSupported() bool {
	switch s.Platform {
	case PlatformDarwin:
		return true
	case PlatformLinux:
		return s.Distro == DistroUbuntu || s.Distro == DistroDebian
	case PlatformWindows:
		return false // Not yet supported
	default:
		return false
	}
}

// SupportedMessage returns a message about platform support status
func (s SystemInfo) SupportedMessage() string {
	if s.IsSupported() {
		return fmt.Sprintf("Platform %s is supported", s)
	}
	switch s.Platform {
	case PlatformWindows:
		return "Windows support is coming soon"
	case PlatformLinux:
		if s.Distro != DistroUbuntu && s.Distro != DistroDebian {
			return fmt.Sprintf("Linux distribution %s is not yet supported. Currently supported: Ubuntu, Debian", s.Distro)
		}
	}
	return fmt.Sprintf("Platform %s is not supported", s)
}

// Detect detects the current system information
func Detect() (*SystemInfo, error) {
	info := &SystemInfo{
		Arch: detectArch(),
	}

	if err := detectPlatform(info); err != nil {
		return nil, err
	}

	return info, nil
}

// detectArch detects the CPU architecture
func detectArch() Arch {
	switch runtime.GOARCH {
	case "amd64":
		return ArchAMD64
	case "arm64":
		return ArchARM64
	default:
		return ArchUnknown
	}
}

// HostagentBinaryName returns the correct hostagent binary name for this platform
func (s SystemInfo) HostagentBinaryName() string {
	switch s.Platform {
	case PlatformDarwin:
		return "hostagent.darwin"
	case PlatformLinux:
		return "hostagent.linux"
	case PlatformWindows:
		return "hostagent.exe"
	default:
		return "hostagent.linux"
	}
}
