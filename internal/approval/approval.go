// Package approval provides interactive tool approval UI for chat commands
package approval

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/edgecli/edgecli/internal/ui"
)

// Decision represents the user's approval decision
type Decision int

const (
	DecisionYes         Decision = iota // Run once
	DecisionAlwaysAllow                 // Always allow (with scope)
	DecisionNo                          // No, provide feedback
	DecisionCancel                      // Cancel/exit
)

// RiskLevel represents the risk level of a command
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Action represents a proposed tool action
type Action struct {
	Tool        string // Tool name (e.g., "bash", "get_system_time")
	Command     string // The command or action details
	WorkDir     string // Working directory (if applicable)
	Rationale   string // Why the assistant wants to run this
	NeedsSudo   bool   // Whether the command needs sudo
	RiskLevel   RiskLevel
	CommandArgs []string // Parsed command arguments
}

// Result contains the approval decision and any feedback
type Result struct {
	Decision Decision
	Feedback string // User feedback if Decision is DecisionNo
	Scope    string // Scope for always-allow (e.g., "prefix:ls", "path:/tmp")
}

// AnalyzeCommand analyzes a command and returns an Action with risk assessment
func AnalyzeCommand(tool, command, rationale string) *Action {
	action := &Action{
		Tool:      tool,
		Command:   command,
		Rationale: rationale,
		WorkDir:   getCurrentDir(),
	}

	// Parse command
	action.CommandArgs = parseCommand(command)

	// Check for sudo
	if len(action.CommandArgs) > 0 && action.CommandArgs[0] == "sudo" {
		action.NeedsSudo = true
	}

	// Assess risk
	action.RiskLevel = assessRisk(command)

	return action
}

// PromptApproval shows the interactive approval UI and returns the user's decision
func PromptApproval(action *Action, rules *Rules) *Result {
	// Check if already allowed by rules
	if rules != nil && rules.IsAllowed(action) {
		fmt.Printf("\n[Auto-approved by saved rule]\n")
		return &Result{Decision: DecisionYes}
	}

	reader := bufio.NewReader(os.Stdin)

	// Print styled action card
	card := ui.RenderActionCard(ui.ActionCardOptions{
		ToolName:  action.Tool,
		Command:   action.Command,
		Rationale: action.Rationale,
		RiskLevel: string(action.RiskLevel),
		WorkDir:   action.WorkDir,
		NeedsSudo: action.NeedsSudo,
	})
	fmt.Print(card)

	// Print options menu
	fmt.Print(ui.RenderActionOptions())
	fmt.Print(ui.RenderChoicePrompt())

	input, err := reader.ReadString('\n')
	if err != nil {
		return &Result{Decision: DecisionCancel}
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		// Default to cancel on invalid input
		fmt.Println("Invalid choice, cancelling.")
		return &Result{Decision: DecisionCancel}
	}

	switch choice {
	case 1:
		return &Result{Decision: DecisionYes}

	case 2:
		// Ask for scope
		scope := promptScope(action, reader)
		return &Result{
			Decision: DecisionAlwaysAllow,
			Scope:    scope,
		}

	case 3:
		// Get feedback
		fmt.Print("\nWhat should the assistant do differently? ")
		feedback, err := reader.ReadString('\n')
		if err != nil {
			return &Result{Decision: DecisionCancel}
		}
		return &Result{
			Decision: DecisionNo,
			Feedback: strings.TrimSpace(feedback),
		}

	case 4:
		return &Result{Decision: DecisionCancel}

	default:
		fmt.Println("Invalid choice, cancelling.")
		return &Result{Decision: DecisionCancel}
	}
}

// promptScope asks the user to choose a scope for the always-allow rule
func promptScope(action *Action, reader *bufio.Reader) string {
	fmt.Println("\nChoose what to allow:")

	// Build options based on command
	options := []struct {
		label string
		scope string
	}{}

	// Option 1: Command prefix (first word or first two words)
	if len(action.CommandArgs) > 0 {
		prefix := action.CommandArgs[0]
		if action.NeedsSudo && len(action.CommandArgs) > 1 {
			prefix = action.CommandArgs[1] // Use command after sudo
		}
		options = append(options, struct {
			label string
			scope string
		}{
			label: fmt.Sprintf("Commands starting with '%s'", prefix),
			scope: fmt.Sprintf("prefix:%s", prefix),
		})
	}

	// Option 2: Exact command
	options = append(options, struct {
		label string
		scope string
	}{
		label: fmt.Sprintf("This exact command"),
		scope: fmt.Sprintf("exact:%s", action.Command),
	})

	// Option 3: Path-based (if command contains a path)
	for _, arg := range action.CommandArgs {
		if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "~/") {
			dir := filepath.Dir(arg)
			if dir != "." && dir != "/" {
				options = append(options, struct {
					label string
					scope string
				}{
					label: fmt.Sprintf("Commands operating in '%s'", dir),
					scope: fmt.Sprintf("path:%s", dir),
				})
				break
			}
		}
	}

	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt.label)
	}

	fmt.Print("\nScope [1]: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return options[0].scope
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return options[0].scope
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		return options[0].scope
	}

	return options[choice-1].scope
}

// assessRisk determines the risk level of a command
func assessRisk(command string) RiskLevel {
	cmdLower := strings.ToLower(command)

	// Special check for piped curl/wget execution (high risk)
	if (strings.Contains(cmdLower, "curl") || strings.Contains(cmdLower, "wget")) &&
		(strings.Contains(cmdLower, "| sh") || strings.Contains(cmdLower, "| bash") ||
			strings.Contains(cmdLower, "|sh") || strings.Contains(cmdLower, "|bash")) {
		return RiskHigh
	}

	// High risk patterns
	highRisk := []string{
		"rm -rf", "rm -fr", "mkfs", "dd if=",
		"> /dev/sd", "chmod 777", "chmod -R 777",
		"> /etc/passwd", "> /etc/shadow",
		"shutdown", "reboot", "init 0", "init 6",
		"kill -9 -1", ":(){:|:&};:",
		"format", "fdisk", "parted",
	}
	for _, pattern := range highRisk {
		if strings.Contains(cmdLower, pattern) {
			return RiskHigh
		}
	}

	// Medium risk patterns
	mediumRisk := []string{
		"sudo", "rm ", "mv ", "cp -r",
		"chmod", "chown", "chgrp",
		"apt ", "yum ", "dnf ", "brew ",
		"pip install", "npm install -g",
		"docker rm", "docker rmi",
		"git push", "git reset --hard",
		"kill", "pkill", "killall",
	}
	for _, pattern := range mediumRisk {
		if strings.Contains(cmdLower, pattern) {
			return RiskMedium
		}
	}

	return RiskLow
}

// parseCommand splits a command into arguments
func parseCommand(command string) []string {
	// Simple splitting - could be improved with proper shell parsing
	return strings.Fields(command)
}

// getCurrentDir returns the current working directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}
