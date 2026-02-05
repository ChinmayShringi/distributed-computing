package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/edgecli/edgecli/internal/qaihub"
	"github.com/spf13/cobra"
)

var qaihubCmd = &cobra.Command{
	Use:   "qaihub",
	Short: "Qualcomm AI Hub CLI commands",
	Long: `Commands for interacting with the Qualcomm AI Hub CLI (qai-hub).

These commands wrap the qai-hub CLI for model compilation and
pipeline verification. Note that compiled models target Qualcomm
Snapdragon devices and cannot run locally on Mac or non-Qualcomm
hardware.`,
}

var qaihubDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check qai-hub installation and configuration",
	Long: `Run diagnostics on the qai-hub CLI installation.

Checks:
- qai-hub binary availability
- Version information
- QAI_HUB_API_TOKEN environment variable
- Basic CLI functionality`,
	RunE: runQaihubDoctor,
}

var qaihubCompileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile a model using qai-hub",
	Long: `Compile an ONNX model for Qualcomm devices using qai-hub.

This submits a compilation job to Qualcomm AI Hub cloud service.
The compiled artifacts are optimized for Snapdragon devices and
cannot run on non-Qualcomm hardware.

Example:
  edgecli qaihub compile --onnx ./model.onnx --target "Samsung Galaxy S24"`,
	RunE: runQaihubCompile,
}

// Flags for qaihub compile
var (
	compileONNXPath string
	compileTarget   string
	compileRuntime  string
	compileOutDir   string
	compileJSON     bool
)

func init() {
	qaihubCmd.AddCommand(qaihubDoctorCmd)
	qaihubCmd.AddCommand(qaihubCompileCmd)

	// Doctor flags
	qaihubDoctorCmd.Flags().Bool("json", false, "Output as JSON")

	// Compile flags
	qaihubCompileCmd.Flags().StringVar(&compileONNXPath, "onnx", "", "Path to ONNX model file (required)")
	qaihubCompileCmd.Flags().StringVar(&compileTarget, "target", "", "Target device (required, e.g., 'Samsung Galaxy S24')")
	qaihubCompileCmd.Flags().StringVar(&compileRuntime, "runtime", "precompiled_qnn_onnx", "Target runtime")
	qaihubCompileCmd.Flags().StringVar(&compileOutDir, "out", "", "Output directory (default: ./artifacts/qaihub/<timestamp>)")
	qaihubCompileCmd.Flags().BoolVar(&compileJSON, "json", false, "Output as JSON")

	qaihubCompileCmd.MarkFlagRequired("onnx")
	qaihubCompileCmd.MarkFlagRequired("target")
}

func runQaihubDoctor(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client := qaihub.New()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.Doctor(ctx)
	if err != nil {
		return fmt.Errorf("doctor failed: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	fmt.Println("Qualcomm AI Hub Doctor")
	fmt.Println("======================")
	fmt.Println()

	if result.QaiHubFound {
		fmt.Println("  qai-hub CLI:    FOUND")
	} else {
		fmt.Println("  qai-hub CLI:    NOT FOUND")
	}

	if result.QaiHubVersion != "" {
		fmt.Printf("  Version:        %s\n", result.QaiHubVersion)
	}

	if result.TokenEnvPresent {
		fmt.Println("  API Token:      SET")
	} else {
		fmt.Println("  API Token:      NOT SET")
	}

	fmt.Println()
	fmt.Println("Notes:")
	for _, note := range result.Notes {
		fmt.Printf("  - %s\n", note)
	}

	if !result.QaiHubFound {
		fmt.Println()
		fmt.Println("To install qai-hub:")
		fmt.Println("  pip install qai-hub")
		fmt.Println()
		fmt.Println("To configure authentication:")
		fmt.Println("  qai-hub configure --api_token YOUR_TOKEN")
		return nil
	}

	return nil
}

func runQaihubCompile(cmd *cobra.Command, args []string) error {
	// Resolve ONNX path
	onnxPath, err := filepath.Abs(compileONNXPath)
	if err != nil {
		return fmt.Errorf("invalid ONNX path: %w", err)
	}

	// Check ONNX file exists
	if _, err := os.Stat(onnxPath); os.IsNotExist(err) {
		return fmt.Errorf("ONNX file not found: %s", onnxPath)
	}

	// Resolve output directory
	outDir := compileOutDir
	if outDir == "" {
		outDir = filepath.Join("artifacts", "qaihub", time.Now().Format("20060102-150405"))
	}
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("invalid output directory: %w", err)
	}

	client := qaihub.New()

	// Check if qai-hub is available
	if !client.IsAvailable() {
		return fmt.Errorf("qai-hub is not installed. Run 'edgecli qaihub doctor' for setup instructions")
	}

	if !compileJSON {
		fmt.Println("Compiling model with Qualcomm AI Hub...")
		fmt.Printf("  ONNX:    %s\n", onnxPath)
		fmt.Printf("  Target:  %s\n", compileTarget)
		fmt.Printf("  Runtime: %s\n", compileRuntime)
		fmt.Printf("  Output:  %s\n", outDir)
		fmt.Println()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := client.Compile(ctx, onnxPath, compileTarget, compileRuntime, outDir)
	if err != nil {
		return fmt.Errorf("compile failed: %w", err)
	}

	if compileJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	if result.Submitted {
		fmt.Println("Compilation submitted successfully!")
	} else {
		fmt.Println("Compilation failed.")
	}

	if result.JobID != "" {
		fmt.Printf("  Job ID:  %s\n", result.JobID)
	}

	fmt.Printf("  Output:  %s\n", result.OutDir)
	fmt.Printf("  Log:     %s\n", result.RawOutputPath)

	if len(result.Notes) > 0 {
		fmt.Println()
		fmt.Println("Notes:")
		for _, note := range result.Notes {
			fmt.Printf("  - %s\n", note)
		}
	}

	if !result.Submitted {
		fmt.Println()
		fmt.Printf("Check %s for details.\n", result.RawOutputPath)
	}

	return nil
}
