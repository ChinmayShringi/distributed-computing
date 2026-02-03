//go:build linux

package osdetect

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

// detectPlatform populates SystemInfo for the current platform
func detectPlatform(info *SystemInfo) error {
	return detectLinux(info)
}

// detectLinux populates SystemInfo for Linux
func detectLinux(info *SystemInfo) error {
	info.Platform = PlatformLinux

	// Parse /etc/os-release for distribution info
	if err := parseOSRelease(info); err != nil {
		// Fallback: try to detect from other sources
		detectDistroFallback(info)
	}

	// Get kernel version
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		info.KernelVersion = strings.TrimSpace(string(out))
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	// Detect package manager based on distro
	info.PackageManager = detectPackageManager(info.Distro)

	return nil
}

// parseOSRelease parses /etc/os-release to get distro information
func parseOSRelease(info *SystemInfo) error {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return err
	}
	defer file.Close()

	data := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := parts[0]
			value := strings.Trim(parts[1], `"`)
			data[key] = value
		}
	}

	// Determine distro from ID
	id := strings.ToLower(data["ID"])
	idLike := strings.ToLower(data["ID_LIKE"])

	switch {
	case id == "ubuntu" || strings.Contains(idLike, "ubuntu"):
		info.Distro = DistroUbuntu
	case id == "debian" || strings.Contains(idLike, "debian"):
		info.Distro = DistroDebian
	case id == "rhel" || id == "redhat":
		info.Distro = DistroRHEL
	case id == "fedora":
		info.Distro = DistroFedora
	case id == "centos":
		info.Distro = DistroCentOS
	case id == "arch" || strings.Contains(idLike, "arch"):
		info.Distro = DistroArch
	default:
		info.Distro = DistroUnknown
	}

	// Get version
	if version, ok := data["VERSION_ID"]; ok {
		info.DistroVersion = version
	} else if version, ok := data["VERSION"]; ok {
		info.DistroVersion = version
	}

	return nil
}

// detectDistroFallback tries to detect distro from other sources
func detectDistroFallback(info *SystemInfo) {
	// Try lsb_release
	if out, err := exec.Command("lsb_release", "-is").Output(); err == nil {
		distro := strings.TrimSpace(strings.ToLower(string(out)))
		switch {
		case strings.Contains(distro, "ubuntu"):
			info.Distro = DistroUbuntu
		case strings.Contains(distro, "debian"):
			info.Distro = DistroDebian
		case strings.Contains(distro, "fedora"):
			info.Distro = DistroFedora
		case strings.Contains(distro, "centos"):
			info.Distro = DistroCentOS
		case strings.Contains(distro, "arch"):
			info.Distro = DistroArch
		default:
			info.Distro = DistroUnknown
		}

		if out, err := exec.Command("lsb_release", "-rs").Output(); err == nil {
			info.DistroVersion = strings.TrimSpace(string(out))
		}
		return
	}

	// Check for specific release files
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		info.Distro = DistroDebian
		if content, err := os.ReadFile("/etc/debian_version"); err == nil {
			info.DistroVersion = strings.TrimSpace(string(content))
		}
		return
	}

	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		info.Distro = DistroRHEL
		return
	}

	if _, err := os.Stat("/etc/arch-release"); err == nil {
		info.Distro = DistroArch
		return
	}

	info.Distro = DistroUnknown
}

// detectPackageManager determines the package manager for a distro
func detectPackageManager(distro Distro) PackageManager {
	switch distro {
	case DistroUbuntu, DistroDebian:
		return PMAPT
	case DistroRHEL, DistroFedora, DistroCentOS:
		return PMDNF
	case DistroArch:
		return PMPacman
	default:
		// Try to detect from available commands
		if _, err := exec.LookPath("apt-get"); err == nil {
			return PMAPT
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return PMDNF
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			return PMPacman
		}
		return PMUnknown
	}
}

// IsSystemdAvailable checks if systemd is available
func IsSystemdAvailable() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}
