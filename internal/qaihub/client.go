// Package qaihub provides integration with the Qualcomm AI Hub CLI (qai-hub).
package qaihub

import (
	"bytes"
	"context"
	"encoding/json"
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

// ListDevicesResult contains the parsed output of qai-hub list-devices.
type ListDevicesResult struct {
	Devices []TargetDevice `json:"devices"`
	RawText string         `json:"raw_text,omitempty"`
}

// TargetDevice represents a Qualcomm target device from the cloud catalog.
type TargetDevice struct {
	Name    string `json:"name"`
	OS      string `json:"os"`
	Vendor  string `json:"vendor"`
	Type    string `json:"type"`
	Chipset string `json:"chipset"`
}

// ListDevices runs "qai-hub list-devices" and returns parsed results.
func (c *Client) ListDevices(ctx context.Context) (*ListDevicesResult, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("qai-hub binary not found")
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Bin, "list-devices")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.applyEnv(cmd)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("list-devices timed out")
		}
		return nil, fmt.Errorf("list-devices failed: %w (stderr: %s)", err, stderr.String())
	}

	raw := stdout.String()
	devices := parseDeviceTable(raw)

	return &ListDevicesResult{
		Devices: devices,
		RawText: raw,
	}, nil
}

// parseDeviceTable parses the prettytable output from qai-hub list-devices.
// Each row has: | Device | OS | Vendor | Type | Chipset | CLI Invocation |
func parseDeviceTable(output string) []TargetDevice {
	var devices []TargetDevice
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "+") || strings.HasPrefix(line, "|") && strings.Contains(line, "Device") && strings.Contains(line, "Chipset") {
			continue // skip borders and header
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			continue
		}
		// parts[0] is empty (before first |), parts[1..6] are the columns
		name := strings.TrimSpace(parts[1])
		osVer := strings.TrimSpace(parts[2])
		vendor := strings.TrimSpace(parts[3])
		devType := strings.TrimSpace(parts[4])
		chipset := strings.TrimSpace(parts[5])

		if name == "" || name == "Device" {
			continue
		}
		devices = append(devices, TargetDevice{
			Name:    name,
			OS:      osVer,
			Vendor:  vendor,
			Type:    devType,
			Chipset: chipset,
		})
	}
	return devices
}

// JobStatusResult contains the status of a QAI Hub job.
type JobStatusResult struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// GetJobStatus checks the status of a QAI Hub job using the Python SDK.
// It shells out to a small Python script because the CLI doesn't have a
// "get-job" command.
func (c *Client) GetJobStatus(ctx context.Context, jobID string) (*JobStatusResult, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job_id is required")
	}

	// Find Python from the venv
	pythonBin := c.findVenvPython()
	if pythonBin == "" {
		return nil, fmt.Errorf("python not found in venv — cannot check job status")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Inline Python script to check job status
	script := fmt.Sprintf(`
import json, sys
try:
    import qai_hub as hub
    job = hub.get_job("%s")
    st = job.get_status()
    print(json.dumps({"job_id": "%s", "status": st.message, "success": st.success}))
except Exception as e:
    print(json.dumps({"job_id": "%s", "status": "error", "success": False, "error": str(e)}))
`, jobID, jobID, jobID)

	cmd := exec.CommandContext(ctx, pythonBin, "-c", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.applyEnv(cmd)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("job status check timed out")
		}
		return nil, fmt.Errorf("job status check failed: %w (stderr: %s)", err, stderr.String())
	}

	var result JobStatusResult
	if err := parseJSON(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse job status: %w (raw: %s)", err, stdout.String())
	}
	return &result, nil
}

// SubmitCompileResult contains the results of a compile job submission via the Python SDK.
type SubmitCompileResult struct {
	OK     bool   `json:"ok"`
	JobID  string `json:"job_id,omitempty"`
	JobURL string `json:"job_url,omitempty"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SubmitCompile submits a compile job using the Python SDK (richer than CLI).
// modelPath can be a local ONNX file or a QAI Hub model ID.
func (c *Client) SubmitCompile(ctx context.Context, modelPath, deviceName, options string) (*SubmitCompileResult, error) {
	pythonBin := c.findVenvPython()
	if pythonBin == "" {
		return nil, fmt.Errorf("python not found in venv")
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Inline Python to submit compile via SDK
	script := fmt.Sprintf(`
import json, sys
try:
    import qai_hub as hub
    devices = hub.get_devices(name=%q)
    if not devices:
        print(json.dumps({"ok": False, "error": "No device found matching %s"}))
        sys.exit(0)
    device = devices[0]
    model = %q
    job = hub.submit_compile_job(model=model, device=device, options=%q)
    job_id = job.job_id if hasattr(job, "job_id") else str(job)
    print(json.dumps({
        "ok": True,
        "job_id": job_id,
        "job_url": "https://aihub.qualcomm.com/jobs/" + job_id,
        "status": "submitted"
    }))
except Exception as e:
    print(json.dumps({"ok": False, "error": str(e)}))
`, deviceName, deviceName, modelPath, options)

	cmd := exec.CommandContext(ctx, pythonBin, "-c", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.applyEnv(cmd)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("compile submission timed out")
		}
		// Don't fail hard — the script might have printed JSON before erroring
	}

	var result SubmitCompileResult
	if err := parseJSON(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse compile result: %w (raw: %s, stderr: %s)", err, stdout.String(), stderr.String())
	}
	return &result, nil
}

// findVenvPython returns the path to the Python binary in the qai-hub venv.
func (c *Client) findVenvPython() string {
	candidates := []string{
		filepath.Join(".venv-qaihub", "bin", "python3"),        // Unix
		filepath.Join(".venv-qaihub", "bin", "python"),         // Unix alt
		filepath.Join(".venv-qaihub", "Scripts", "python.exe"), // Windows
		filepath.Join(".venv", "bin", "python3"),               // Generic Unix
		filepath.Join(".venv", "Scripts", "python.exe"),        // Generic Windows
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}
	// Fall back to system python
	if p, err := exec.LookPath("python3"); err == nil {
		return p
	}
	if p, err := exec.LookPath("python"); err == nil {
		return p
	}
	return ""
}

// parseJSON is a small helper to unmarshal JSON.
func parseJSON(data []byte, v interface{}) error {
	// Trim any leading/trailing whitespace or non-JSON characters
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty response")
	}
	return json.Unmarshal(trimmed, v)
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
