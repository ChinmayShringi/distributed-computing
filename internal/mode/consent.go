package mode

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/edgecli/edgecli/internal/ui"
)

// PromptDangerousConsent shows a full-screen warning and requires exact phrase confirmation.
// Returns true if user consented, false if aborted.
func PromptDangerousConsent() bool {
	// Clear screen for full-screen warning (ANSI escape sequence)
	fmt.Print("\033[2J\033[H")

	width := 80

	// Top warning banner
	fmt.Println(ui.Color(ui.Red+ui.Bold, strings.Repeat("!", width)))
	fmt.Println()
	fmt.Println(ui.Color(ui.Red+ui.Bold, centerText("DANGEROUS MODE WARNING", width)))
	fmt.Println()
	fmt.Println(ui.Color(ui.Red+ui.Bold, strings.Repeat("!", width)))
	fmt.Println()

	// Warning details
	fmt.Println(ui.Color(ui.Yellow+ui.Bold, "You are about to enable DANGEROUS MODE."))
	fmt.Println()
	fmt.Println(ui.Color(ui.White, "This mode:"))
	fmt.Println()
	fmt.Println(ui.Color(ui.White, "  * Disables the tool allowlist"))
	fmt.Println(ui.Color(ui.White, "  * Disables argument schema validation"))
	fmt.Println(ui.Color(ui.White, "  * Allows raw shell command execution"))
	fmt.Println(ui.Color(ui.White, "  * Enables the AI to execute ANY command on your system"))
	fmt.Println()

	// Consequences section
	fmt.Println(ui.Color(ui.Red+ui.Bold, "THIS MAY RESULT IN:"))
	fmt.Println()
	fmt.Println(ui.Color(ui.Red, "  * DATA LOSS"))
	fmt.Println(ui.Color(ui.Red, "  * SYSTEM CORRUPTION"))
	fmt.Println(ui.Color(ui.Red, "  * SECURITY VULNERABILITIES"))
	fmt.Println(ui.Color(ui.Red, "  * UNINTENDED SYSTEM MODIFICATIONS"))
	fmt.Println()

	// Safety note
	fmt.Println(ui.Color(ui.Yellow, "All actions will still be logged for audit purposes."))
	fmt.Println(ui.Color(ui.Yellow, "This bypasses ALL safety guarantees."))
	fmt.Println()

	// Divider
	fmt.Println(ui.Color(ui.Red+ui.Bold, strings.Repeat("-", width)))
	fmt.Println()

	// Confirmation prompt
	fmt.Println("To continue, type the following phrase EXACTLY:")
	fmt.Println()
	fmt.Printf("  %s\n", ui.Color(ui.Cyan+ui.Bold, RequiredPhrase))
	fmt.Println()
	fmt.Print("Your input: ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println()
		fmt.Println(ui.Color(ui.Red, "Error reading input. Aborting."))
		return false
	}

	input = strings.TrimSpace(input)

	// Validate exact match
	if input != RequiredPhrase {
		fmt.Println()
		fmt.Println(ui.Color(ui.Red+ui.Bold, "Phrase did not match. Aborting."))
		fmt.Println()
		fmt.Println(ui.Color(ui.Dim, fmt.Sprintf("Expected: %q", RequiredPhrase)))
		fmt.Println(ui.Color(ui.Dim, fmt.Sprintf("Got:      %q", input)))
		fmt.Println()
		return false
	}

	// Success
	fmt.Println()
	fmt.Println(ui.Color(ui.Yellow+ui.Bold, strings.Repeat("!", width)))
	fmt.Println(ui.Color(ui.Yellow+ui.Bold, centerText("DANGEROUS MODE ENABLED", width)))
	fmt.Println(ui.Color(ui.Yellow+ui.Bold, strings.Repeat("!", width)))
	fmt.Println()

	return true
}

// centerText centers text within the given width
func centerText(text string, width int) string {
	padding := (width - len(text)) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + text
}
