// Package deviceid provides persistent device ID management
package deviceid

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	// ConfigDir is the directory for edgemesh configuration
	ConfigDir = ".edgemesh"
	// DeviceIDFile is the filename for the device ID
	DeviceIDFile = "device_id"
)

// GetOrCreate returns the device ID, creating one if it doesn't exist
// The device ID is persisted in ~/.edgemesh/device_id
func GetOrCreate() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}

	// Try to read existing device ID
	data, err := os.ReadFile(configPath)
	if err == nil {
		deviceID := strings.TrimSpace(string(data))
		if deviceID != "" {
			return deviceID, nil
		}
	}

	// Generate new device ID
	deviceID := uuid.New().String()

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	// Write device ID to file
	if err := os.WriteFile(configPath, []byte(deviceID), 0600); err != nil {
		return "", err
	}

	return deviceID, nil
}

// Get returns the device ID if it exists, or empty string if not
func Get() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// getConfigPath returns the path to the device ID file
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDir, DeviceIDFile), nil
}
