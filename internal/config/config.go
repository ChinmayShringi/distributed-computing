// Package config manages CLI configuration and state persistence
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// ConfigDirName is the name of the config directory
	ConfigDirName = ".edgecli"
	// ConfigFileName is the name of the config file
	ConfigFileName = "config.json"
)

// Config holds the CLI configuration
type Config struct {
	// Verbose enables verbose logging
	Verbose bool `json:"verbose"`
	// Auth holds authentication tokens (optional, for future use)
	Auth *AuthConfig `json:"auth,omitempty"`
}

// AuthConfig holds authentication state
type AuthConfig struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Username     string    `json:"username"`
}

// IsAuthenticated returns true if valid auth tokens exist
func (c *Config) IsAuthenticated() bool {
	if c.Auth == nil {
		return false
	}
	if c.Auth.AccessToken == "" || c.Auth.RefreshToken == "" {
		return false
	}
	// Check if expired (with 1 minute buffer)
	return time.Now().Add(time.Minute).Before(c.Auth.ExpiresAt)
}

// Paths holds commonly used paths
type Paths struct {
	// ConfigDir is ~/.edgecli
	ConfigDir string
	// ConfigFile is ~/.edgecli/config.json
	ConfigFile string
	// LogsDir is ~/.edgecli/logs
	LogsDir string
	// CacheDir is ~/.edgecli/cache
	CacheDir string
}

// GetPaths returns the standard paths
func GetPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ConfigDirName)
	return &Paths{
		ConfigDir:  configDir,
		ConfigFile: filepath.Join(configDir, ConfigFileName),
		LogsDir:    filepath.Join(configDir, "logs"),
		CacheDir:   filepath.Join(configDir, "cache"),
	}, nil
}

// EnsureDirectories creates all required directories
func (p *Paths) EnsureDirectories() error {
	dirs := []string{p.ConfigDir, p.LogsDir, p.CacheDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Default returns a new Config with default values
func Default() *Config {
	return &Config{
		Verbose: false,
	}
}

// Load loads configuration from disk
func Load() (*Config, error) {
	paths, err := GetPaths()
	if err != nil {
		return nil, err
	}

	// Ensure config directory exists
	if err := paths.EnsureDirectories(); err != nil {
		return nil, err
	}

	// If config file doesn't exist, return defaults
	if _, err := os.Stat(paths.ConfigFile); os.IsNotExist(err) {
		return Default(), nil
	}

	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := Default()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// Save saves configuration to disk
func (c *Config) Save() error {
	paths, err := GetPaths()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(paths.ConfigFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(paths.ConfigFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ClearAuth removes authentication tokens
func (c *Config) ClearAuth() error {
	c.Auth = nil
	return c.Save()
}

// SetAuth sets authentication tokens
func (c *Config) SetAuth(accessToken, refreshToken, username string, expiresIn int) error {
	c.Auth = &AuthConfig{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Username:     username,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
	return c.Save()
}
