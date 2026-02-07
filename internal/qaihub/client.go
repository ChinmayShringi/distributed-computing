// Package qaihub provides integration with the Qualcomm AI Hub CLI (qai-hub).
package qaihub

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Environment variable for the QAI Hub API token.
const EnvQAIHubAPIToken = "QAI_HUB_API_TOKEN"

// Default timeout for CLI operations.
const defaultTimeout = 120 * time.Second

// Client wraps the qai-hub CLI.
type Client struct {
	Bin     string            // path to qai-hub binary
	Env     map[string]string // additional environment variables
	Timeout time.Duration     // timeout for CLI operations
}

// DoctorResult contains the results of a qai-hub health check.
type DoctorResult struct {
	QaiHubFound     bool     `json:"qai_hub_found"`
	QaiHubVersion   string   `json:"qai_hub_version,omitempty"`
	TokenEnvPresent bool     `json:"token_env_present"`
	Notes           []string `json:"notes"`
}

// CompileResult contains the results of a model compilation.
type CompileResult struct {
	Submitted     bool     `json:"submitted"`
	JobID         string   `json:"job_id,omitempty"`
	OutDir        string   `json:"out_dir"`
	RawOutputPath string   `json:"raw_output_path"`
	Notes         []string `json:"notes"`
}

// New creates a new Client, auto-detecting the qai-hub binary.
func New() *Client {
	return &Client{
		Bin:     findQaiHubBinary(),
		Timeout: defaultTimeout,
	}
}

// NewWithBin creates a new Client with a specific binary path.
func NewWithBin(bin string) *Client {
	return &Client{
		Bin:     bin,
		Timeout: defaultTimeout,
	}
}

// findQaiHubBinary looks for the qai-hub binary in common locations.
func findQaiHubBinary() string {
	// Check common venv locations first
	venvPaths := []string{
		filepath.Join(".venv-qaihub", "Scripts", "qai-hub.exe"), // Windows venv
		filepath.Join(".venv-qaihub", "bin", "qai-hub"),         // Unix venv
		filepath.Join(".venv", "Scripts", "qai-hub.exe"),        // Generic Windows venv
		filepath.Join(".venv", "bin", "qai-hub"),                // Generic Unix venv
	}

	for _, path := range venvPaths {
		if _, err := os.Stat(path); err == nil {
			if absPath, err := filepath.Abs(path); err == nil {
				return absPath
			}
			return path
		}
	}

	// Fall back to PATH
	if path, err := exec.LookPath("qai-hub"); err == nil {
		return path
	}

	// Return "qai-hub" and let execution fail with helpful error
	return "qai-hub"
}

// IsAvailable returns true if qai-hub is available.
func (c *Client) IsAvailable() bool {
	if c.Bin == "" {
		return false
	}
	_, err := exec.LookPath(c.Bin)
	if err != nil {
		// Check if it's an absolute path that exists
		if _, statErr := os.Stat(c.Bin); statErr == nil {
			return true
		}
		return false
	}
	return true
}

// Version returns the qai-hub version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Bin, "--version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.applyEnv(cmd)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("qai-hub --version timed out")
		}
		return "", fmt.Errorf("qai-hub --version failed: %w (stderr: %s)", err, stderr.String())
	}

	version := strings.TrimSpace(stdout.String())
	if version == "" {
		version = strings.TrimSpace(stderr.String()) // Some CLIs output version to stderr
	}
	return version, nil
}

// Doctor performs a health check on the qai-hub installation.
func (c *Client) Doctor(ctx context.Context) (*DoctorResult, error) {
	result := &DoctorResult{
		Notes: make([]string, 0),
	}

	// Check if binary is available
	if !c.IsAvailable() {
		result.QaiHubFound = false
		result.Notes = append(result.Notes, "qai-hub binary not found")
		result.Notes = append(result.Notes, "Install with: pip install qai-hub")
		return result, nil
	}

	result.QaiHubFound = true
	result.Notes = append(result.Notes, fmt.Sprintf("qai-hub binary found at: %s", c.Bin))

	// Get version
	version, err := c.Version(ctx)
	if err != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("Failed to get version: %v", err))
	} else {
		result.QaiHubVersion = version
		result.Notes = append(result.Notes, fmt.Sprintf("Version: %s", version))
	}

	// Check for API token
	token := os.Getenv(EnvQAIHubAPIToken)
	if token != "" {
		result.TokenEnvPresent = true
		result.Notes = append(result.Notes, fmt.Sprintf("%s is set", EnvQAIHubAPIToken))
	} else {
		result.TokenEnvPresent = false
		result.Notes = append(result.Notes, fmt.Sprintf("%s is NOT set", EnvQAIHubAPIToken))
		result.Notes = append(result.Notes, "Configure with: qai-hub configure --api_token YOUR_TOKEN")
	}

	// Try to verify connectivity by running a simple command
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(listCtx, c.Bin, "list-devices", "--help")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	c.applyEnv(cmd)

	if err := cmd.Run(); err != nil {
		if listCtx.Err() == context.DeadlineExceeded {
			result.Notes = append(result.Notes, "list-devices --help timed out")
		} else {
			result.Notes = append(result.Notes, fmt.Sprintf("list-devices --help failed: %v", err))
		}
	} else {
		result.Notes = append(result.Notes, "CLI commands appear functional")
	}

	return result, nil
}

// Compile runs model compilation using qai-hub.
// onnxPath: path to the ONNX model file
// target: target device (e.g., "Samsung Galaxy S24")
// runtime: compilation runtime (e.g., "precompiled_qnn_onnx")
// outDir: output directory for artifacts
func (c *Client) Compile(ctx context.Context, onnxPath, target, runtime, outDir string) (*CompileResult, error) {
	result := &CompileResult{
		Notes: make([]string, 0),
	}

	// Validate inputs
	if onnxPath == "" {
		return nil, fmt.Errorf("onnx_path is required")
	}
	if target == "" {
		return nil, fmt.Errorf("target device is required")
	}

	// Check if ONNX file exists
	if _, err := os.Stat(onnxPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("ONNX file not found: %s", onnxPath)
	}

	// Set default runtime
	if runtime == "" {
		runtime = "precompiled_qnn_onnx"
	}

	// Set default output directory
	if outDir == "" {
		outDir = filepath.Join("artifacts", "qaihub", time.Now().Format("20060102-150405"))
	}

	// Create output directory
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	result.OutDir = outDir

	// Create log file path
	logPath := filepath.Join(outDir, "compile.log")
	result.RawOutputPath = logPath

	// Build compile command
	// Note: The exact qai-hub compile syntax may vary; this is a best-effort implementation
	// based on typical CLI patterns. Adjust as needed based on actual CLI help output.
	args := []string{
		"compile",
		"--model", onnxPath,
		"--device", target,
	}

	// Add runtime if the CLI supports it
	if runtime != "" {
		args = append(args, "--target-runtime", runtime)
	}

	result.Notes = append(result.Notes, fmt.Sprintf("Running: %s %s", c.Bin, strings.Join(args, " ")))

	// Set timeout
	timeout := c.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Run compile
	cmd := exec.CommandContext(ctx, c.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = outDir
	c.applyEnv(cmd)

	runErr := cmd.Run()

	// Save raw output to log file
	logContent := fmt.Sprintf("Command: %s %s\n\nStdout:\n%s\n\nStderr:\n%s\n",
		c.Bin, strings.Join(args, " "), stdout.String(), stderr.String())

	if writeErr := os.WriteFile(logPath, []byte(logContent), 0644); writeErr != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("Warning: failed to write log file: %v", writeErr))
	}

	if runErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Notes = append(result.Notes, fmt.Sprintf("Compile timed out after %s", timeout))
			return result, fmt.Errorf("compile timed out after %s", timeout)
		}
		result.Notes = append(result.Notes, fmt.Sprintf("Compile failed: %v", runErr))
		result.Notes = append(result.Notes, "Check compile.log for details")
		return result, nil // Return result with notes, not error, so caller can see details
	}

	result.Submitted = true
	result.Notes = append(result.Notes, "Compile command completed")

	// Try to parse job ID from output
	// Look for patterns like "Job ID: abc123" or just alphanumeric strings that look like job IDs
	combinedOutput := stdout.String() + "\n" + stderr.String()
	jobID := parseJobID(combinedOutput)
	if jobID != "" {
		result.JobID = jobID
		result.Notes = append(result.Notes, fmt.Sprintf("Parsed job ID: %s", jobID))
	}

	return result, nil
}

// parseJobID attempts to extract a job ID from CLI output.
func parseJobID(output string) string {
	// Try common patterns
	patterns := []string{
		`[Jj]ob\s+[Ii][Dd]:\s*([a-z0-9]{6,})`,    // "Job ID: abc123"
		`[Jj]ob:\s*([a-z0-9]{6,})`,               // "Job: abc123"
		`(?:submitted|created).*?([a-z0-9]{6,})`, // "submitted job abc123"
		`([a-z][a-z0-9]{7,}[a-z0-9])`,            // Generic alphanumeric job ID
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(output); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// applyEnv sets environment variables for the command.
func (c *Client) applyEnv(cmd *exec.Cmd) {
	if len(c.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range c.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
}
