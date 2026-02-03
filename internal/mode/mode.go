// Package mode defines execution mode types for safe and dangerous operation
package mode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/user"
	"runtime"
	"time"
)

// ExecutionMode represents the safety level of command execution
type ExecutionMode int

const (
	// ModeSafe is the default mode with allowlist enforcement and schema validation
	ModeSafe ExecutionMode = iota
	// ModeDangerous bypasses safety controls and allows raw shell execution
	ModeDangerous
)

// String returns the string representation of the execution mode
func (m ExecutionMode) String() string {
	switch m {
	case ModeDangerous:
		return "DANGEROUS"
	default:
		return "SAFE"
	}
}

// IsDangerous returns true if the mode is dangerous
func (m ExecutionMode) IsDangerous() bool {
	return m == ModeDangerous
}

// ModeContext contains the execution mode and associated metadata
type ModeContext struct {
	Mode             ExecutionMode `json:"mode"`
	ConsentTimestamp time.Time     `json:"consent_timestamp,omitempty"`
	ConsentHash      string        `json:"consent_hash,omitempty"`
	UserUID          string        `json:"user_uid,omitempty"`
	CLIVersion       string        `json:"cli_version"`
	OS               string        `json:"os"`
	Arch             string        `json:"arch"`
}

// RequiredPhrase is the exact phrase users must type to enable dangerous mode
const RequiredPhrase = "I UNDERSTAND AND ACCEPT THE RISK"

// NewSafeContext creates a new ModeContext in safe mode
func NewSafeContext(cliVersion string) *ModeContext {
	return &ModeContext{
		Mode:       ModeSafe,
		CLIVersion: cliVersion,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}
}

// NewDangerousContext creates a new ModeContext in dangerous mode with consent metadata
func NewDangerousContext(cliVersion string) (*ModeContext, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	timestamp := time.Now().UTC()

	ctx := &ModeContext{
		Mode:             ModeDangerous,
		ConsentTimestamp: timestamp,
		UserUID:          u.Uid,
		CLIVersion:       cliVersion,
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
	}

	// Compute consent hash for audit verification
	ctx.ConsentHash = ctx.computeConsentHash()

	return ctx, nil
}

// computeConsentHash generates a SHA-256 hash of consent data for tamper-proof audit trail
func (m *ModeContext) computeConsentHash() string {
	// Hash: phrase + timestamp + uid + version
	data := fmt.Sprintf("%s|%s|%s|%s",
		RequiredPhrase,
		m.ConsentTimestamp.Format(time.RFC3339Nano),
		m.UserUID,
		m.CLIVersion,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// DangerousModeMetadata contains metadata for dangerous mode audit logging
type DangerousModeMetadata struct {
	Enabled          bool      `json:"enabled"`
	ConsentTimestamp time.Time `json:"consent_timestamp,omitempty"`
	ConsentHash      string    `json:"consent_hash,omitempty"`
	UserUID          string    `json:"user_uid,omitempty"`
	CLIVersion       string    `json:"cli_version,omitempty"`
	OS               string    `json:"os,omitempty"`
	Arch             string    `json:"arch,omitempty"`
}

// ToMetadata converts ModeContext to DangerousModeMetadata for incident bundles
func (m *ModeContext) ToMetadata() *DangerousModeMetadata {
	return &DangerousModeMetadata{
		Enabled:          m.Mode.IsDangerous(),
		ConsentTimestamp: m.ConsentTimestamp,
		ConsentHash:      m.ConsentHash,
		UserUID:          m.UserUID,
		CLIVersion:       m.CLIVersion,
		OS:               m.OS,
		Arch:             m.Arch,
	}
}
