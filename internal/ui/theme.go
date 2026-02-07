// Package ui provides terminal UI components for CLI output styling
package ui

import (
	"os"

	"golang.org/x/term"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Cyan    = "\033[36m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	White   = "\033[37m"
)

// Box drawing characters
const (
	BoxTopLeft     = "╭"
	BoxTopRight    = "╮"
	BoxBottomLeft  = "╰"
	BoxBottomRight = "╯"
	BoxHorizontal  = "─"
	BoxVertical    = "│"
	BoxTeeRight    = "├"
	BoxTeeLeft     = "┤"
)

var (
	colorEnabled = true
	isTTY        = true
)

func init() {
	// Check NO_COLOR env var (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
	}
	// Check if stdout is a TTY
	isTTY = term.IsTerminal(int(os.Stdout.Fd()))
	if !isTTY {
		colorEnabled = false
	}
}

// SetNoColor disables color output
func SetNoColor(disable bool) {
	if disable {
		colorEnabled = false
	}
}

// IsColorEnabled returns whether color output is enabled
func IsColorEnabled() bool {
	return colorEnabled
}

// IsTTY returns whether stdout is a terminal
func IsTTY() bool {
	return isTTY
}

// Color wraps text with an ANSI color code
func Color(code, text string) string {
	if !colorEnabled {
		return text
	}
	return code + text + Reset
}
