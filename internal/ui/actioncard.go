package ui

import (
	"fmt"
	"strings"
)

// ActionCardOptions configures the action card display
type ActionCardOptions struct {
	ToolName  string
	Command   string
	Rationale string
	RiskLevel string // "HIGH", "MEDIUM", "LOW"
	WorkDir   string
	NeedsSudo bool
}

// RenderActionCard displays a styled action approval card
func RenderActionCard(opts ActionCardOptions) string {
	width := 70

	var sb strings.Builder

	// Top border with title
	title := fmt.Sprintf(" Action Required: %s ", opts.ToolName)
	topPadding := width - 4 - len(title)
	if topPadding < 0 {
		topPadding = 0
	}
	topBorder := BoxTopLeft + strings.Repeat(BoxHorizontal, 2) + title +
		strings.Repeat(BoxHorizontal, topPadding) + BoxTopRight
	sb.WriteString("\n")
	sb.WriteString(Color(Yellow, topBorder))
	sb.WriteString("\n")

	// Command section
	cmdDisplay := truncate(opts.Command, width-14)
	sb.WriteString(Color(Yellow, BoxVertical))
	sb.WriteString(fmt.Sprintf(" %s %s", Color(Bold, "Command:"), cmdDisplay))
	cmdPadding := width - 12 - len(cmdDisplay)
	if cmdPadding > 0 {
		sb.WriteString(strings.Repeat(" ", cmdPadding))
	}
	sb.WriteString(Color(Yellow, BoxVertical))
	sb.WriteString("\n")

	// Working directory
	if opts.WorkDir != "" {
		sb.WriteString(Color(Yellow, BoxVertical))
		dirDisplay := truncate(opts.WorkDir, width-16)
		sb.WriteString(fmt.Sprintf(" %s %s", Color(Dim, "Directory:"), dirDisplay))
		dirPadding := width - 14 - len(dirDisplay)
		if dirPadding > 0 {
			sb.WriteString(strings.Repeat(" ", dirPadding))
		}
		sb.WriteString(Color(Yellow, BoxVertical))
		sb.WriteString("\n")
	}

	// Sudo indicator
	if opts.NeedsSudo {
		sb.WriteString(Color(Yellow, BoxVertical))
		sb.WriteString(fmt.Sprintf(" %s %s", Color(Dim, "Sudo:"), Color(Red+Bold, "Required")))
		sudoPadding := width - 17
		if sudoPadding > 0 {
			sb.WriteString(strings.Repeat(" ", sudoPadding))
		}
		sb.WriteString(Color(Yellow, BoxVertical))
		sb.WriteString("\n")
	}

	// Risk level with color coding
	riskColor := Green
	riskText := "LOW"
	switch strings.ToUpper(opts.RiskLevel) {
	case "HIGH":
		riskColor = Red
		riskText = "HIGH"
	case "MEDIUM":
		riskColor = Yellow
		riskText = "MEDIUM"
	default:
		riskColor = Green
		riskText = "LOW"
	}
	sb.WriteString(Color(Yellow, BoxVertical))
	sb.WriteString(fmt.Sprintf(" %s %s", Color(Dim, "Risk:"), Color(Bold+riskColor, riskText)))
	riskPadding := width - 10 - len(riskText)
	if riskPadding > 0 {
		sb.WriteString(strings.Repeat(" ", riskPadding))
	}
	sb.WriteString(Color(Yellow, BoxVertical))
	sb.WriteString("\n")

	// Rationale section (if provided)
	if opts.Rationale != "" {
		// Separator
		sb.WriteString(Color(Yellow, BoxTeeRight+strings.Repeat(BoxHorizontal, width-2)+BoxTeeLeft))
		sb.WriteString("\n")

		// Wrap and display rationale
		wrapped := wrapText(opts.Rationale, width-4)
		for _, line := range wrapped {
			sb.WriteString(Color(Yellow, BoxVertical))
			sb.WriteString(" ")
			sb.WriteString(line)
			linePadding := width - 3 - len(line)
			if linePadding > 0 {
				sb.WriteString(strings.Repeat(" ", linePadding))
			}
			sb.WriteString(Color(Yellow, BoxVertical))
			sb.WriteString("\n")
		}
	}

	// Bottom border
	sb.WriteString(Color(Yellow, BoxBottomLeft+strings.Repeat(BoxHorizontal, width-2)+BoxBottomRight))
	sb.WriteString("\n")

	return sb.String()
}

// RenderActionOptions displays the action choice menu
func RenderActionOptions() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(Color(Bold, "Choose an option:\n"))
	sb.WriteString(fmt.Sprintf("  %s Yes, run this command\n", Color(Green, "[1]")))
	sb.WriteString(fmt.Sprintf("  %s Yes, always allow (configure scope)\n", Color(Cyan, "[2]")))
	sb.WriteString(fmt.Sprintf("  %s No, tell assistant what to do differently\n", Color(Yellow, "[3]")))
	sb.WriteString(fmt.Sprintf("  %s Cancel\n", Color(Red, "[4]")))
	sb.WriteString("\n")

	return sb.String()
}

// RenderChoicePrompt displays the choice input prompt
func RenderChoicePrompt() string {
	return "Choice [1-4]: "
}

// truncate shortens a string if it exceeds maxLen
func truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// wrapText wraps text to fit within the specified width
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}

	var lines []string
	words := strings.Fields(s)
	var current string

	for _, word := range words {
		if len(current)+len(word)+1 > width {
			if current != "" {
				lines = append(lines, current)
			}
			current = word
		} else {
			if current != "" {
				current += " "
			}
			current += word
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}
