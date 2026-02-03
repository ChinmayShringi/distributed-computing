package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// RenderHeader displays a styled welcome panel with ASCII art (Claude Code style)
func RenderHeader(appName, version, user, serverURL, workspace string) string {
	width := 78

	var sb strings.Builder

	// Top border with version: ╭─── Omniforge v0.1 ───────────────────────────╮
	titleText := fmt.Sprintf(" EdgeCLI v%s ", version)
	titleLen := utf8.RuneCountInString(titleText)
	leftDashes := 3
	rightDashes := width - 2 - leftDashes - titleLen
	if rightDashes < 0 {
		rightDashes = 0
	}

	sb.WriteString(Color(Cyan, BoxTopLeft))
	sb.WriteString(Color(Cyan, strings.Repeat(BoxHorizontal, leftDashes)))
	sb.WriteString(Color(Cyan+Bold, titleText))
	sb.WriteString(Color(Cyan, strings.Repeat(BoxHorizontal, rightDashes)))
	sb.WriteString(Color(Cyan, BoxTopRight))
	sb.WriteString("\n")

	// Empty line
	sb.WriteString(formatCenteredLine("", width))

	// Welcome message
	welcome := fmt.Sprintf("Welcome back %s!", user)
	sb.WriteString(formatCenteredLine(Color(Bold, welcome), width))

	// Empty line
	sb.WriteString(formatCenteredLine("", width))

	// ASCII art logo
	sb.WriteString(formatCenteredLine(Color(Magenta, "* ▐▛███▜▌ *"), width))
	sb.WriteString(formatCenteredLine(Color(Magenta, "* ▝▜█████▛▘ *"), width))
	sb.WriteString(formatCenteredLine(Color(Magenta, "*  ▘▘ ▝▝  *"), width))

	// Empty line
	sb.WriteString(formatCenteredLine("", width))

	// App name
	sb.WriteString(formatCenteredLine(Color(Bold+Cyan, "EdgeCLI"), width))

	// Empty line
	sb.WriteString(formatCenteredLine("", width))

	// CWD
	sb.WriteString(formatCenteredLine(Color(Dim, workspace), width))

	// Empty line
	sb.WriteString(formatCenteredLine("", width))

	// Bottom border
	sb.WriteString(Color(Cyan, BoxBottomLeft+strings.Repeat(BoxHorizontal, width-2)+BoxBottomRight))
	sb.WriteString("\n")

	return sb.String()
}

// formatCenteredLine creates a centered line within the box
func formatCenteredLine(text string, width int) string {
	var sb strings.Builder

	// Calculate visible length (strip ANSI codes for width calculation)
	visibleLen := visibleLength(text)
	padding := (width - 2 - visibleLen) / 2
	rightPadding := width - 2 - padding - visibleLen
	if padding < 0 {
		padding = 0
	}
	if rightPadding < 0 {
		rightPadding = 0
	}

	sb.WriteString(Color(Cyan, BoxVertical))
	sb.WriteString(strings.Repeat(" ", padding))
	sb.WriteString(text)
	sb.WriteString(strings.Repeat(" ", rightPadding))
	sb.WriteString(Color(Cyan, BoxVertical))
	sb.WriteString("\n")

	return sb.String()
}

// visibleLength returns the visible length of a string, ignoring ANSI codes
func visibleLength(s string) int {
	// Strip ANSI escape codes
	inEscape := false
	visible := 0
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		visible++
	}
	return visible
}

func formatInfoLine(label, value string, width int) string {
	var sb strings.Builder

	// Calculate visible length (without color codes)
	visibleLen := len(label) + len(value) + 4 // " label: value"
	padding := width - 2 - visibleLen
	if padding < 0 {
		padding = 0
	}

	sb.WriteString(Color(Cyan, BoxVertical))
	sb.WriteString(" ")
	sb.WriteString(Color(Dim, label+":"))
	sb.WriteString(" ")
	sb.WriteString(value)
	sb.WriteString(strings.Repeat(" ", padding))
	sb.WriteString(Color(Cyan, BoxVertical))
	sb.WriteString("\n")

	return sb.String()
}

// RenderMessage formats a chat message with role styling
func RenderMessage(role, text string) string {
	switch role {
	case "user":
		return fmt.Sprintf("%s %s", Color(Bold+Green, "You:"), text)
	case "assistant":
		return fmt.Sprintf("%s %s", Color(Bold+Blue, "EdgeCLI:"), text)
	case "system":
		return Color(Dim, text)
	default:
		return text
	}
}

// RenderHelpLines displays command hints
func RenderHelpLines() string {
	var sb strings.Builder

	sb.WriteString(Color(Dim, "  Commands: "))
	sb.WriteString("exit")
	sb.WriteString(Color(Dim, " | "))
	sb.WriteString("clear")
	sb.WriteString(Color(Dim, " | "))
	sb.WriteString("!cmd")
	sb.WriteString(Color(Dim, " (run bash)"))
	sb.WriteString("\n")
	sb.WriteString(Color(Dim, "  Press Ctrl+C to interrupt"))
	sb.WriteString("\n\n")

	return sb.String()
}

// RenderUserPrompt returns the styled "You: " prompt
func RenderUserPrompt() string {
	return Color(Bold+Green, "You: ")
}

// RenderAssistantPrefix returns the styled "EdgeCLI: " prefix
func RenderAssistantPrefix() string {
	return Color(Bold+Blue, "EdgeCLI: ")
}

// RenderError formats an error message
func RenderError(err error) string {
	return Color(Red, fmt.Sprintf("Error: %v", err))
}

// RenderSuccess formats a success message
func RenderSuccess(msg string) string {
	return Color(Green, msg)
}

// RenderDim formats text in dim style
func RenderDim(msg string) string {
	return Color(Dim, msg)
}

// RenderDangerousModeIndicator returns the dangerous mode warning indicator
func RenderDangerousModeIndicator() string {
	return Color(Red+Bold, "[DANGEROUS MODE ENABLED]")
}
