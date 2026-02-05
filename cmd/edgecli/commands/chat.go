package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/edgecli/edgecli/internal/ui"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Interactive LLM chat with tool calling",
	Long: `Start an interactive chat session with the LLM that can call tools
to interact with devices in the mesh.

Available tools:
  - get_capabilities: List all devices and their capabilities
  - execute_shell_cmd: Run shell commands on devices
  - get_file: Read files from devices

Examples:
  # Interactive mode
  edgecli chat

  # Single message mode
  edgecli chat "show disk usage on all devices"

  # Custom web server address
  edgecli chat --web-addr localhost:8080 "list devices"
`,
	RunE: runChat,
}

var chatWebAddr string

func init() {
	chatCmd.Flags().StringVar(&chatWebAddr, "web-addr", "localhost:8080", "Web server address")
	rootCmd.AddCommand(chatCmd)
}

// chatRequest is the request body for /api/agent
type chatRequest struct {
	Message string `json:"message"`
}

// chatResponse is the response from /api/agent
type chatResponse struct {
	Reply      string `json:"reply"`
	Iterations int    `json:"iterations"`
	ToolCalls  []struct {
		Iteration int    `json:"iteration"`
		ToolName  string `json:"tool_name"`
		Arguments string `json:"arguments"`
		ResultLen int    `json:"result_len"`
	} `json:"tool_calls,omitempty"`
	Error string `json:"error,omitempty"`
}

func runChat(cmd *cobra.Command, args []string) error {
	// If a message is provided as argument, run in single-shot mode
	if len(args) > 0 {
		message := strings.Join(args, " ")
		return sendChatMessage(message, false)
	}

	// Interactive mode - show header
	currentUser := "user"
	if u, err := user.Current(); err == nil {
		currentUser = u.Username
	}

	cwd, _ := os.Getwd()
	header := ui.RenderHeader("EdgeMesh", "1.0", currentUser, chatWebAddr, cwd)
	fmt.Print(header)
	fmt.Print(ui.RenderHelpLines())

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(ui.RenderUserPrompt())
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println(ui.RenderDim("Goodbye!"))
			return nil
		case "help", "?":
			printChatHelp()
			continue
		case "tools":
			printChatTools()
			continue
		case "clear":
			// Clear screen
			fmt.Print("\033[H\033[2J")
			header := ui.RenderHeader("EdgeMesh", "1.0", currentUser, chatWebAddr, cwd)
			fmt.Print(header)
			fmt.Print(ui.RenderHelpLines())
			continue
		}

		// Handle shell commands with !
		if strings.HasPrefix(input, "!") {
			shellCmd := strings.TrimPrefix(input, "!")
			fmt.Println(ui.RenderDim(fmt.Sprintf("Running: %s", shellCmd)))
			// Note: actual shell execution would go here
			fmt.Println(ui.RenderDim("(Shell execution not implemented in chat mode)"))
			continue
		}

		if err := sendChatMessage(input, true); err != nil {
			fmt.Println(ui.RenderError(err))
		}
		fmt.Println() // Add spacing after response
	}

	return scanner.Err()
}

func sendChatMessage(message string, interactive bool) error {
	url := fmt.Sprintf("http://%s/api/agent", chatWebAddr)

	reqBody := chatRequest{Message: message}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Start spinner
	spinner := ui.NewSpinner("Thinking...")
	spinner.Start()

	resp, err := http.DefaultClient.Do(req)
	spinner.Stop()

	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Display results
	if chatResp.Error != "" {
		return fmt.Errorf("%s", chatResp.Error)
	}

	// Show tool calls info if any
	if len(chatResp.ToolCalls) > 0 {
		toolInfo := fmt.Sprintf("[%d tool call(s) in %d iteration(s)]", len(chatResp.ToolCalls), chatResp.Iterations)
		fmt.Println(ui.RenderDim(toolInfo))
		for _, tc := range chatResp.ToolCalls {
			fmt.Println(ui.RenderDim(fmt.Sprintf("  - %s (%d bytes)", tc.ToolName, tc.ResultLen)))
		}
		fmt.Println()
	}

	// Show reply with assistant prefix
	if interactive {
		fmt.Print(ui.RenderAssistantPrefix())
	}
	fmt.Println(chatResp.Reply)

	return nil
}

func printChatHelp() {
	fmt.Println()
	fmt.Println(ui.Color(ui.Bold, "Commands:"))
	fmt.Printf("  %s    - Show this help\n", ui.Color(ui.Cyan, "help, ?"))
	fmt.Printf("  %s     - List available tools\n", ui.Color(ui.Cyan, "tools"))
	fmt.Printf("  %s     - Clear the screen\n", ui.Color(ui.Cyan, "clear"))
	fmt.Printf("  %s - Exit the chat\n", ui.Color(ui.Cyan, "exit, quit"))
	fmt.Printf("  %s     - Run a bash command\n", ui.Color(ui.Cyan, "!cmd"))
	fmt.Println()
	fmt.Println(ui.Color(ui.Bold, "The assistant has access to these tools:"))
	fmt.Printf("  %s - Discover devices in the mesh\n", ui.Color(ui.Green, "get_capabilities"))
	fmt.Printf("  %s - Run shell commands on devices\n", ui.Color(ui.Green, "execute_shell_cmd"))
	fmt.Printf("  %s - Read files from devices\n", ui.Color(ui.Green, "get_file"))
	fmt.Println()
	fmt.Println(ui.Color(ui.Bold, "Examples:"))
	fmt.Println(ui.RenderDim("  \"list all devices\""))
	fmt.Println(ui.RenderDim("  \"show disk usage on any device\""))
	fmt.Println(ui.RenderDim("  \"read /etc/os-release from the laptop\""))
	fmt.Println(ui.RenderDim("  \"what's the memory usage across all devices?\""))
	fmt.Println()
}

func printChatTools() {
	fmt.Println()
	fmt.Println(ui.Color(ui.Bold+ui.Cyan, "Available Tools"))
	fmt.Println()

	// Tool 1: get_capabilities
	fmt.Println(ui.Color(ui.Bold, "1. get_capabilities"))
	fmt.Println("   List all registered devices with their capabilities")
	fmt.Println(ui.RenderDim("   Parameters:"))
	fmt.Println(ui.RenderDim("     - include_benchmarks (bool): Include LLM benchmark data"))
	fmt.Println()

	// Tool 2: execute_shell_cmd
	fmt.Println(ui.Color(ui.Bold, "2. execute_shell_cmd"))
	fmt.Println("   Execute a shell command on a device")
	fmt.Println(ui.RenderDim("   Parameters:"))
	fmt.Println(ui.RenderDim("     - device_id (string): Target device (from get_capabilities)"))
	fmt.Println(ui.RenderDim("     - command (string): The shell command to run"))
	fmt.Println(ui.RenderDim("     - timeout_ms (int): Timeout in milliseconds (default: 30000)"))
	fmt.Println()

	// Tool 3: get_file
	fmt.Println(ui.Color(ui.Bold, "3. get_file"))
	fmt.Println("   Read file contents from a device")
	fmt.Println(ui.RenderDim("   Parameters:"))
	fmt.Println(ui.RenderDim("     - device_id (string): Target device"))
	fmt.Println(ui.RenderDim("     - path (string): File path to read"))
	fmt.Println(ui.RenderDim("     - read_mode (string): \"full\", \"head\", \"tail\", or \"range\""))
	fmt.Println(ui.RenderDim("     - max_bytes (int): Maximum bytes to read (default: 65536)"))
	fmt.Println()
}
