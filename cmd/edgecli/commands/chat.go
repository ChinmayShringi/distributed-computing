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
	"sync"
	"sync/atomic"
	"time"

	"github.com/edgecli/edgecli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	chatCmd = &cobra.Command{
		Use:   "chat [message]",
		Short: "Interactive LLM chat with tool calling",
		Long: `Start an interactive chat session with the LLM that can call tools
to interact with devices in the mesh.`,
		RunE: runChat,
	}
	
	chatWebAddr     string
	lastMessageCount int
	isChatting      atomic.Bool
	printMutex      sync.Mutex
)

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

// chatMemoryResponse matches the JSON structure from /api/chat/memory
type chatMemoryResponse struct {
	Version       int `json:"version"`
	LastUpdatedMs int64 `json:"last_updated_ms"`
	Summary       string `json:"summary"`
	Messages      []struct {
		Role        string `json:"role"`
		Content     string `json:"content"`
		TimestampMs int64  `json:"timestamp_ms"`
	} `json:"messages"`
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

	// Fetch and display history
	fetchAndDisplayHistory()

	// Start background polling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pollChatHistory(ctx)

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
			fmt.Print("\033[H\033[2J")
			fmt.Print(header)
			fmt.Print(ui.RenderHelpLines())
			continue
		}

		// Handle shell commands
		if strings.HasPrefix(input, "!") {
			shellCmd := strings.TrimPrefix(input, "!")
			fmt.Println(ui.RenderDim(fmt.Sprintf("Running: %s", shellCmd)))
			fmt.Println(ui.RenderDim("(Shell execution not implemented in chat mode)"))
			continue
		}

		isChatting.Store(true)
		reply, err := sendChatMessageWithReply(input, true)
		isChatting.Store(false)
		
		if err != nil {
			fmt.Println(ui.RenderError(err))
		} else {
			// Explicitly print the interaction to ensure it's visible (handling spinner line clearing)
			// and that the reply is shown (since syncAndPrintNew skips it)
			fmt.Println(ui.RenderUserPrompt() + input)
			fmt.Println(ui.RenderAssistantPrefix() + reply)

			// Sync and skip our own message and reply
			syncAndPrintNew(input, reply)
		}
		fmt.Println() 
	}

	return scanner.Err()
}

func fetchAndDisplayHistory() {
	// Re-use syncAndPrintNew logic for initial load, but maybe add summary support later
	// For now, just treating it as a sync from 0 works perfectly to display all messages.
	syncAndPrintNew("", "")
}

func sendChatMessage(message string, interactive bool) error {
	reply, err := sendChatMessageWithReply(message, interactive)
	if err != nil {
		return err
	}
	if interactive {
		// In interactive mode, the runChat loop handles printing via syncAndPrintNew
		// to ensure deduplication. We don't print here.
	} else {
		// In single-shot mode, we must print the reply.
		fmt.Println(reply)
	}
	return nil
}

func sendChatMessageWithReply(message string, interactive bool) (string, error) {
	url := fmt.Sprintf("http://%s/api/agent", chatWebAddr)

	reqBody := chatRequest{Message: message}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Start spinner
	spinner := ui.NewSpinner("Thinking...")
	spinner.Start()

	resp, err := http.DefaultClient.Do(req)
	spinner.Stop()

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("server error: %s", errResp.Error)
		}
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Returns error string as error
	if chatResp.Error != "" {
		return "", fmt.Errorf("%s", chatResp.Error)
	}

	// Show tool calls info if any (Side effect: prints to stdout)
	if len(chatResp.ToolCalls) > 0 {
		toolInfo := fmt.Sprintf("[%d tool call(s) in %d iteration(s)]", len(chatResp.ToolCalls), chatResp.Iterations)
		fmt.Println(ui.RenderDim(toolInfo))
		for _, tc := range chatResp.ToolCalls {
			fmt.Println(ui.RenderDim(fmt.Sprintf("  - %s (%d bytes)", tc.ToolName, tc.ResultLen)))
		}
		fmt.Println()
	}

	return chatResp.Reply, nil
}

func pollChatHistory(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !isChatting.Load() {
				syncAndPrintNew("", "")
			}
		}
	}
}

func syncAndPrintNew(skipUser, skipAgent string) {
	printMutex.Lock()
	defer printMutex.Unlock()

	url := fmt.Sprintf("http://%s/api/chat/memory", chatWebAddr)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var mem chatMemoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&mem); err != nil {
		return
	}

	if len(mem.Messages) > lastMessageCount {
		newMsgs := mem.Messages[lastMessageCount:]
		for _, msg := range newMsgs {
			// Dedup check
			if skipUser != "" && msg.Role == "user" && strings.TrimSpace(msg.Content) == strings.TrimSpace(skipUser) {
				continue
			}
			if skipAgent != "" && msg.Role == "assistant" && strings.TrimSpace(msg.Content) == strings.TrimSpace(skipAgent) {
				continue
			}

			// Print new message
			// We need to clear current line to avoid mess with prompt?
			// Ideally yes, but tricky. We'll just print newline.
			fmt.Println() 
			if msg.Role == "user" {
				fmt.Print(ui.RenderUserPrompt() + msg.Content + "\n")
			} else {
				fmt.Print(ui.RenderAssistantPrefix() + msg.Content + "\n")
			}
		}
		
		// If we printed something via background polling, prompt might be buried. 
		// Ideally we reprint prompt.
		if skipUser == "" && skipAgent == "" && len(newMsgs) > 0 {
			fmt.Print(ui.RenderUserPrompt()) 
		}

		lastMessageCount = len(mem.Messages)
	}
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
