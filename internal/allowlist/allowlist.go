// Package allowlist provides command allowlisting for remote execution
package allowlist

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// CommandSpec represents a validated command ready for execution
type CommandSpec struct {
	Executable string
	Args       []string
}

// AllowedCommands is the set of commands permitted for remote execution
var AllowedCommands = map[string]bool{
	"pwd": true,
	"ls":  true,
	"cat": true,
}

// ValidateCommand checks if a command is allowed and returns the OS-specific
// command spec for execution. Returns an error if the command is not allowed
// or if validation fails.
func ValidateCommand(command string, args []string) (*CommandSpec, error) {
	if !AllowedCommands[command] {
		return nil, fmt.Errorf("command %q is not in the allowlist", command)
	}

	switch command {
	case "pwd":
		return mapPwd()
	case "ls":
		return mapLs(args)
	case "cat":
		return mapCat(args)
	default:
		return nil, fmt.Errorf("command %q is not implemented", command)
	}
}

// mapPwd returns the OS-specific command for pwd
func mapPwd() (*CommandSpec, error) {
	if runtime.GOOS == "windows" {
		return &CommandSpec{
			Executable: "cmd",
			Args:       []string{"/c", "cd"},
		}, nil
	}
	return &CommandSpec{
		Executable: "pwd",
		Args:       nil,
	}, nil
}

// mapLs returns the OS-specific command for ls
func mapLs(args []string) (*CommandSpec, error) {
	if runtime.GOOS == "windows" {
		cmdArgs := []string{"/c", "dir"}
		cmdArgs = append(cmdArgs, args...)
		return &CommandSpec{
			Executable: "cmd",
			Args:       cmdArgs,
		}, nil
	}
	lsArgs := []string{"-la"}
	lsArgs = append(lsArgs, args...)
	return &CommandSpec{
		Executable: "ls",
		Args:       lsArgs,
	}, nil
}

// mapCat returns the OS-specific command for cat with path validation
func mapCat(args []string) (*CommandSpec, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("cat requires a file path argument")
	}

	// Validate each path argument
	for _, arg := range args {
		if err := validateCatPath(arg); err != nil {
			return nil, err
		}
	}

	if runtime.GOOS == "windows" {
		cmdArgs := []string{"/c", "type"}
		cmdArgs = append(cmdArgs, args...)
		return &CommandSpec{
			Executable: "cmd",
			Args:       cmdArgs,
		}, nil
	}

	return &CommandSpec{
		Executable: "cat",
		Args:       args,
	}, nil
}

// validateCatPath validates that a path is safe for cat command
// - Must be under ./shared/ directory
// - Must not contain path traversal sequences
// - Must not be an absolute path
func validateCatPath(path string) error {
	// Reject absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths are not allowed: %s", path)
	}

	// Reject path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal is not allowed: %s", path)
	}

	// Clean the path and check if it's under ./shared/
	cleaned := filepath.Clean(path)

	// Must start with "shared/" or "shared" (for the directory itself)
	if !strings.HasPrefix(cleaned, "shared/") && cleaned != "shared" {
		return fmt.Errorf("cat is only allowed to read files under ./shared/: %s", path)
	}

	return nil
}

// IsAllowed returns true if the command is in the allowlist
func IsAllowed(command string) bool {
	return AllowedCommands[command]
}

// ListAllowed returns all allowed commands
func ListAllowed() []string {
	commands := make([]string, 0, len(AllowedCommands))
	for cmd := range AllowedCommands {
		commands = append(commands, cmd)
	}
	return commands
}
