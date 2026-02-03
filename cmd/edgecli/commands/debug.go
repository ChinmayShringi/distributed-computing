package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// debugCmd is the parent command for debug subcommands
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug and diagnostic commands",
	Long:  `Commands for debugging and diagnosing issues with EdgeCLI.`,
}

// debugFlagsCmd prints resolved flag values for debugging
var debugFlagsCmd = &cobra.Command{
	Use:   "flags",
	Short: "Print resolved flag values for debugging",
	Long: `Print the resolved values of global flags for debugging purposes.

This is useful to verify that flags like --allow-dangerous are being
correctly parsed and inherited from the command line.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		configPath, _ := cmd.Flags().GetString("config")
		allowDangerous, _ := cmd.Flags().GetBool("allow-dangerous")
		adAlias, _ := cmd.Flags().GetBool("ad")
		autoConfirm, _ := cmd.Flags().GetBool("yes")

		fmt.Println("Resolved Flag Values:")
		fmt.Printf("  --verbose:         %v\n", verbose)
		fmt.Printf("  --config:          %q\n", configPath)
		fmt.Printf("  --allow-dangerous: %v\n", allowDangerous)
		fmt.Printf("  --ad:              %v\n", adAlias)
		fmt.Printf("  --yes:             %v\n", autoConfirm)
		fmt.Printf("  Dangerous Mode:    %v\n", allowDangerous || adAlias)
		return nil
	},
}

func init() {
	debugCmd.AddCommand(debugFlagsCmd)
}
