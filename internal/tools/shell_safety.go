// Package tools provides shell command safety validation for LLM tool calling
package tools

import (
	"fmt"
	"strings"
)

// DangerousPatterns contains shell patterns that are blocked for safety
var DangerousPatterns = []string{
	// Destructive file operations
	"rm -rf /",
	"rm -rf ~",
	"rm -rf .",
	"rm -r /",
	"rm -fr",

	// Disk operations
	"dd if=",
	"mkfs",
	"format c:",
	"fdisk",
	"parted",

	// Fork bomb patterns
	":(){ :|:& };:",
	":(){",
	"fork()",

	// Permission disasters
	"chmod 777 /",
	"chmod -R 777",
	"chown -R",

	// Write to device nodes
	"> /dev/sd",
	"> /dev/nvme",
	"> /dev/null",
	">/dev/sd",
	">/dev/nvme",

	// System control
	"shutdown",
	"reboot",
	"poweroff",
	"halt",
	"init 0",
	"init 6",

	// Remote code execution - catch any piping to shell
	"| sh",
	"| bash",
	"| sh -",
	"| bash -",
	"|sh",
	"|bash",

	// Environment manipulation
	"export PATH=",
	"unset PATH",

	// History manipulation
	"history -c",
	"rm ~/.bash_history",

	// Cron manipulation
	"crontab -r",
}

// ValidationResult contains the result of command validation
type ValidationResult struct {
	Allowed bool
	Reason  string
}

// ValidateShellCommand checks if a shell command is safe to execute
// Returns nil if the command is allowed, error with reason if blocked
func ValidateShellCommand(command string) error {
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Normalize for pattern matching
	lower := strings.ToLower(command)
	normalized := strings.ReplaceAll(lower, "  ", " ")

	// Check against dangerous patterns
	for _, pattern := range DangerousPatterns {
		patternLower := strings.ToLower(pattern)
		if strings.Contains(normalized, patternLower) {
			return fmt.Errorf("blocked: command contains dangerous pattern %q", pattern)
		}
	}

	return nil
}

// ValidateShellCommandWithResult returns a detailed validation result
func ValidateShellCommandWithResult(command string) ValidationResult {
	err := ValidateShellCommand(command)
	if err != nil {
		return ValidationResult{
			Allowed: false,
			Reason:  err.Error(),
		}
	}
	return ValidationResult{
		Allowed: true,
		Reason:  "command allowed",
	}
}

// IsDangerousPattern checks if a specific pattern is in the blocklist
func IsDangerousPattern(pattern string) bool {
	lower := strings.ToLower(pattern)
	for _, p := range DangerousPatterns {
		if strings.ToLower(p) == lower {
			return true
		}
	}
	return false
}

// ListDangerousPatterns returns all blocked patterns (for documentation/UI)
func ListDangerousPatterns() []string {
	result := make([]string, len(DangerousPatterns))
	copy(result, DangerousPatterns)
	return result
}
