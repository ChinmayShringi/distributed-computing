//go:build windows

package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	pb "github.com/edgecli/edgecli/proto"
)

const (
	// cliTimeout is the maximum time to wait for CLI commands.
	cliTimeout = 30 * time.Second
)

// isAvailable returns true if the brain is configured and available on Windows.
func isAvailable(b *Brain) bool {
	if !b.enabled || b.cliPath == "" {
		return false
	}

	// Check if the CLI executable exists
	if _, err := os.Stat(b.cliPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// generatePlan calls the Windows AI CLI to generate an execution plan.
func generatePlan(b *Brain, text string, devices []*pb.DeviceInfo, maxWorkers int) (*PlanResult, error) {
	if !isAvailable(b) {
		return nil, fmt.Errorf("brain not available")
	}

	// Create request
	request := PlanRequest{
		Text:       text,
		MaxWorkers: maxWorkers,
		Devices:    convertDevicesToJSON(devices),
	}

	// Write request to temp file
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "brain-plan-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(requestJSON); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Execute CLI
	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, b.cliPath, "plan", "--in", tmpFile.Name(), "--format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("CLI timed out after %s", cliTimeout)
		}
		return nil, fmt.Errorf("CLI execution failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse response
	var response PlanResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to parse CLI response: %w (output: %s)", err, stdout.String())
	}

	if !response.Ok {
		return nil, fmt.Errorf("CLI returned error: %s", response.Error)
	}

	return &PlanResult{
		Plan:      convertPlanToProto(response.Plan),
		Reduce:    convertReduceToProto(response.Reduce),
		UsedAi:    response.UsedAi,
		Notes:     response.Notes,
		Rationale: response.Rationale,
	}, nil
}

// summarize calls the Windows AI CLI to summarize text.
func summarize(b *Brain, text string) (string, bool, error) {
	if !isAvailable(b) {
		return "", false, fmt.Errorf("brain not available")
	}

	// Execute CLI
	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, b.cliPath, "summarize", "--text", text, "--format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", false, fmt.Errorf("CLI timed out after %s", cliTimeout)
		}
		return "", false, fmt.Errorf("CLI execution failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse response
	var response SummarizeResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return "", false, fmt.Errorf("failed to parse CLI response: %w (output: %s)", err, stdout.String())
	}

	if !response.Ok {
		return "", false, fmt.Errorf("CLI returned error: %s", response.Error)
	}

	return response.Summary, response.UsedAi, nil
}
