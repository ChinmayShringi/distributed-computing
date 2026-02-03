// Package exec provides safe command execution with output capture
package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default command timeout
	DefaultTimeout = 2 * time.Minute
	// MaxOutputSize is the maximum output size to capture (1MB)
	MaxOutputSize = 1024 * 1024
	// TailLines is the default number of lines to include in tails
	TailLines = 100
)

// Result holds the result of a command execution
type Result struct {
	// Command is the command that was executed
	Command string
	// Args are the arguments passed to the command
	Args []string
	// ExitCode is the exit code (0 = success)
	ExitCode int
	// Stdout is the full stdout output
	Stdout string
	// Stderr is the full stderr output
	Stderr string
	// Duration is how long the command took
	Duration time.Duration
	// Error is any error that occurred
	Error error
	// TimedOut is true if the command timed out
	TimedOut bool
}

// OK returns true if the command succeeded (exit code 0)
func (r *Result) OK() bool {
	return r.ExitCode == 0 && r.Error == nil
}

// StdoutTail returns the last N lines of stdout
func (r *Result) StdoutTail(lines int) string {
	return tailString(r.Stdout, lines)
}

// StderrTail returns the last N lines of stderr
func (r *Result) StderrTail(lines int) string {
	return tailString(r.Stderr, lines)
}

// CombinedOutput returns stdout and stderr combined
func (r *Result) CombinedOutput() string {
	if r.Stderr == "" {
		return r.Stdout
	}
	if r.Stdout == "" {
		return r.Stderr
	}
	return r.Stdout + "\n" + r.Stderr
}

// String returns a human-readable summary
func (r *Result) String() string {
	status := "OK"
	if !r.OK() {
		status = fmt.Sprintf("FAILED (exit %d)", r.ExitCode)
	}
	if r.TimedOut {
		status = "TIMEOUT"
	}
	return fmt.Sprintf("%s %s [%s] (%s)", r.Command, strings.Join(r.Args, " "), status, r.Duration)
}

// Runner executes commands with safety features
type Runner struct {
	// Timeout is the default timeout for commands
	Timeout time.Duration
	// Env is additional environment variables
	Env []string
	// Dir is the working directory
	Dir string
	// Verbose logs commands before execution
	Verbose bool
}

// NewRunner creates a new Runner with defaults
func NewRunner() *Runner {
	return &Runner{
		Timeout: DefaultTimeout,
	}
}

// Run executes a command and returns the result
func (r *Runner) Run(ctx context.Context, name string, args ...string) *Result {
	return r.RunWithTimeout(ctx, r.Timeout, name, args...)
}

// RunWithTimeout executes a command with a specific timeout
func (r *Runner) RunWithTimeout(ctx context.Context, timeout time.Duration, name string, args ...string) *Result {
	start := time.Now()
	result := &Result{
		Command: name,
		Args:    args,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, name, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	if len(r.Env) > 0 {
		cmd.Env = append(cmd.Environ(), r.Env...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdout, limit: MaxOutputSize}
	cmd.Stderr = &limitedWriter{w: &stderr, limit: MaxOutputSize}

	// Run command
	err := cmd.Run()
	result.Duration = time.Since(start)
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	// Handle errors
	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.Error = fmt.Errorf("command timed out after %s", timeout)
		result.ExitCode = -1
	} else if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result
}

// RunSimple runs a command and returns stdout or error
func (r *Runner) RunSimple(ctx context.Context, name string, args ...string) (string, error) {
	result := r.Run(ctx, name, args...)
	if !result.OK() {
		return "", fmt.Errorf("%s: %s", result.Error, result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// CommandExists checks if a command is available in PATH
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// WhichCommand returns the full path to a command
func WhichCommand(name string) (string, error) {
	return exec.LookPath(name)
}

// tailString returns the last N lines of a string
func tailString(s string, lines int) string {
	if lines <= 0 {
		return ""
	}
	split := strings.Split(s, "\n")
	if len(split) <= lines {
		return s
	}
	return strings.Join(split[len(split)-lines:], "\n")
}

// limitedWriter limits the amount written to prevent memory issues
type limitedWriter struct {
	w       *bytes.Buffer
	limit   int
	written int
}

func (lw *limitedWriter) Write(p []byte) (n int, err error) {
	if lw.written >= lw.limit {
		return len(p), nil // Discard but pretend we wrote it
	}
	remaining := lw.limit - lw.written
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err = lw.w.Write(p)
	lw.written += n
	return len(p), err
}
