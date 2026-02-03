package commands

import (
	"fmt"

	"github.com/edgecli/edgecli/internal/mode"
	"github.com/edgecli/edgecli/internal/osdetect"
	"github.com/edgecli/edgecli/internal/tools"
	"github.com/edgecli/edgecli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time
	Commit = "none"
)

var rootCmd = &cobra.Command{
	Use:   "edgecli",
	Short: "EdgeCLI - Edge-first tool execution framework",
	Long: `EdgeCLI is a framework for building edge-first CLI tools with safe mode,
dangerous mode, and approval workflows for remote tool execution.

Use "edgecli [command] --help" for more information about a command.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("config", "", "Config file (default: ~/.edgecli/config.json)")

	// Dangerous mode flags (global)
	rootCmd.PersistentFlags().Bool("allow-dangerous", false, "Enable dangerous mode (bypasses safety controls)")
	rootCmd.PersistentFlags().Bool("ad", false, "Alias for --allow-dangerous")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "Auto-confirm dangerous mode consent (use with --allow-dangerous)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(debugCmd)
}

// versionCmd shows version info
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		info, err := osdetect.Detect()

		fmt.Printf("EdgeCLI\n")
		fmt.Printf("  Version:  %s\n", Version)
		fmt.Printf("  Commit:   %s\n", Commit)
		if err == nil {
			fmt.Printf("  Platform: %s\n", info)
		}
	},
}

// toolsCmd lists registered tools
var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List registered tools",
	Long:  `List all tools registered in the tool registry.`,
	Run: func(cmd *cobra.Command, args []string) {
		registry := tools.DefaultRegistry()
		toolList := registry.ListTools()

		if len(toolList) == 0 {
			fmt.Println("No tools registered.")
			fmt.Println("\nTo register tools, implement the tools.Tool interface and call registry.Register().")
			return
		}

		fmt.Printf("Registered tools (%d):\n\n", len(toolList))
		for _, t := range toolList {
			dangerous := ""
			if t.IsDangerous() {
				dangerous = ui.Color(ui.Red, " [DANGEROUS]")
			}
			fmt.Printf("  %s%s\n", t.Name(), dangerous)
			fmt.Printf("    %s\n\n", t.Description())
		}
	},
}

// GetModeContext returns the execution mode context based on flags
func GetModeContext(cmd *cobra.Command) (*mode.ModeContext, error) {
	allowDangerous, _ := cmd.Flags().GetBool("allow-dangerous")
	adAlias, _ := cmd.Flags().GetBool("ad")
	autoConfirm, _ := cmd.Flags().GetBool("yes")

	if allowDangerous || adAlias {
		// Prompt for consent unless auto-confirm is set
		if !autoConfirm {
			if !mode.PromptDangerousConsent() {
				return nil, fmt.Errorf("dangerous mode consent denied")
			}
		}
		// Create dangerous mode context
		modeCtx, err := mode.NewDangerousContext(Version)
		if err != nil {
			return nil, err
		}
		return modeCtx, nil
	}

	// Default to safe mode
	return mode.NewSafeContext(Version), nil
}

// GetToolRegistry returns the default tool registry
func GetToolRegistry() *tools.Registry {
	return tools.DefaultRegistry()
}
