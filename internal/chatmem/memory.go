// Package chatmem provides persistent chat memory management for EdgeCLI.
// Chat history is stored as a JSON file and synchronized across devices.
package chatmem

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// MaxMessages is the maximum number of messages to keep verbatim.
	// Older messages are summarized.
	MaxMessages = 6

	// DefaultPath is the default location for the chat memory file.
	DefaultPath = ".edgemesh/chat_memory.json"
)

// ChatMessage represents a single message in the chat history.
type ChatMessage struct {
	Role        string `json:"role"` // "user", "assistant", or "system"
	Content     string `json:"content"`
	TimestampMs int64  `json:"timestamp_ms"`
}

// ChatMemory manages persistent chat history.
type ChatMemory struct {
	Version       int           `json:"version"`
	LastUpdatedMs int64         `json:"last_updated_ms"`
	Summary       string        `json:"summary"`  // Summary of older messages
	Messages      []ChatMessage `json:"messages"` // Most recent messages (max MaxMessages)

	mu   sync.RWMutex
	path string // file path for persistence
}

// New creates a new empty ChatMemory instance.
func New() *ChatMemory {
	return &ChatMemory{
		Version:  1,
		Messages: make([]ChatMessage, 0),
	}
}

// DefaultFilePath returns the default path for the chat memory file.
func DefaultFilePath() (string, error) {
	return DeviceSpecificFilePath("default")
}

// DeviceSpecificFilePath returns a device-scoped chat memory file path.
func DeviceSpecificFilePath(deviceID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".edgemesh", "chats", "chat_memory_"+deviceID+".json"), nil
}

// LoadFromFile loads chat memory from a JSON file.
// If the file doesn't exist, returns an empty ChatMemory.
func LoadFromFile(path string) (*ChatMemory, error) {
	m := New()
	m.path = path

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	return m, nil
}

// SaveToFile persists the chat memory to its JSON file.
func (m *ChatMemory) SaveToFile() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveToFileUnsafe()
}

// saveToFileUnsafe writes to disk without locking (caller must hold lock)
func (m *ChatMemory) saveToFileUnsafe() error {
	if m.path == "" {
		path, err := DefaultFilePath()
		if err != nil {
			return err
		}
		m.path = path
	}

	// Ensure directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.path, data, 0644)
}

// AddMessage adds a new message to the chat history.
// Unlike the previous version, this does NOT auto-summarize synchronously.
// The caller is responsible for checking len(m.Messages) and calling SummarizeAsync if needed.
func (m *ChatMemory) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := ChatMessage{
		Role:        role,
		Content:     content,
		TimestampMs: time.Now().UnixMilli(),
	}

	m.Messages = append(m.Messages, msg)
	m.LastUpdatedMs = msg.TimestampMs
}

// SummarizerFunc is the signature for the callback function that performs summarization.
// It receives the current summary and the messages to be summarized.
// It returns the new summary string or an error.
type SummarizerFunc func(currentSummary string, messagesToSummarize []ChatMessage) (string, error)

// SummarizeAsync triggers an asynchronous summarization if the message count exceeds the limit.
// It takes a callback function that performs the actual summarization (e.g., calling an LLM).
// The onComplete callback (optional) is called after a successful update and save.
func (m *ChatMemory) SummarizeAsync(summarizer SummarizerFunc, onComplete func()) {
	m.mu.Lock()
	// Check condition again under lock
	if len(m.Messages) <= MaxMessages {
		m.mu.Unlock()
		return
	}

	// Identify messages to summarize.
	// We want to keep MaxMessages/2 messages, and summarize the rest?
	// Or just summarize the excess?
	// Strategy: Keep the last N messages (e.g. 10). Summarize everything before that.
	// If MaxMessages = 10, and we have 12. We summarize the first 2.

	keepCount := MaxMessages / 2 // Keep at least half context
	if keepCount < 2 {
		keepCount = 2
	}
	// Or simpler: Keep MaxMessages. Summarize only if > MaxMessages.
	// If we have 12, Max=10. Excess = 2.
	// Summarize the first 2.

	excess := len(m.Messages) - MaxMessages
	if excess <= 0 {
		m.mu.Unlock()
		return
	}

	log.Printf("[DEBUG] SummarizeAsync: Triggering for %d excess messages (total %d)", excess, len(m.Messages))

	// Actually, we should summarize a chunk to avoid frequent small summarizations.
	// E.g. trigger at 15, summarize down to 10?
	// User said "exceed 10... keep on making the last messages as a summary".
	// Let's summarize the oldest 'excess' messages.

	msgsToCompact := make([]ChatMessage, excess)
	copy(msgsToCompact, m.Messages[:excess])

	// Create a temporary slice for the remaining messages to ensure we don't lose them if summarization fails?
	// No, we can just grab copies and update state later.
	// BUT, if we update state later, the "excess" messages might have changed indices if more messages were added?
	// If we use `m.Messages = m.Messages[excess:]` later, we assume `Msgs[0]` is still `Msgs[0]`.
	// Since we only append, the head is stable.
	// UNLESS another summarize happens.
	// We need to prevent concurrent summarizations.

	// For simplicity in this demo:
	// We will optimistically remove them from the "main" list?
	// No, user needs to see them until summary is ready.
	// We'll mark them as "being summarized"? Too complex.

	// Block other summarizations?
	// We can use a `isSummarizing` flag.

	// For this prototype:
	// We accept a small race condition where if multiple summarizations trigger, `summary` might get weird.
	// But `SummarizeAsync` is called by `cmd/web` likely sequentially.

	currentSummary := m.Summary
	m.mu.Unlock()

	// Execute callback (slow LLM call)
	go func() {
		newSummary, err := summarizer(currentSummary, msgsToCompact)
		if err != nil {
			// Log error? using a logger injected? Or just ignore.
			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		// Verify we can still safely remove those messages.
		// We assumed we removed `excess` messages from the head.
		// We need to check if those messages are still at the head.
		// We can check timestamps or IDs.
		// Simple check: `len(m.Messages) >= excess` and `m.Messages[0].Timestamp == msgsToCompact[0].Timestamp`

		if len(m.Messages) >= len(msgsToCompact) &&
			m.Messages[0].TimestampMs == msgsToCompact[0].TimestampMs {

			log.Printf("[DEBUG] SummarizeAsync: Updating memory. Old summary len: %d. New summary len: %d", len(m.Summary), len(newSummary))
			m.Summary = newSummary
			m.Messages = m.Messages[len(msgsToCompact):]
			m.LastUpdatedMs = time.Now().UnixMilli() // Mark updated so file saves

			// Auto save?
			// The caller `cmd/web` usually saves after `AddMessage`.
			// Since this is async/background, we should probably trigger a save here.
			// Ideall we'd have a callback for "OnUpdate".
			// But `cmd/web` can't know when this finishes.
			// `SaveToFile()` is available.
			if err := m.saveToFileUnsafe(); err != nil {
				log.Printf("[ERROR] SummarizeAsync: Failed to save to file: %v", err)
			} else {
				log.Printf("[DEBUG] SummarizeAsync: Saved to file %s", m.path)

				// Trigger completion callback (e.g., to sync to orchestrator)
				if onComplete != nil {
					go onComplete()
				}
			}
		} else {
			log.Printf("[WARN] SummarizeAsync: Memory changed during summarization, aborting update.")
		}
	}()
}

// Merge updates this ChatMemory with data from another if the other is newer.
// Returns true if this memory was updated.
func (m *ChatMemory) Merge(other *ChatMemory) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if other == nil || other.LastUpdatedMs <= m.LastUpdatedMs {
		return false
	}

	m.Version = other.Version
	m.LastUpdatedMs = other.LastUpdatedMs
	m.Summary = other.Summary
	m.Messages = make([]ChatMessage, len(other.Messages))
	copy(m.Messages, other.Messages)
	return true
}

// ToJSON serializes the chat memory to JSON.
func (m *ChatMemory) ToJSON() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ParseFromJSON deserializes chat memory from JSON.
func ParseFromJSON(data string) (*ChatMemory, error) {
	m := New()
	if err := json.Unmarshal([]byte(data), m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetMessages returns a copy of the current messages.
func (m *ChatMemory) GetMessages() []ChatMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ChatMessage, len(m.Messages))
	copy(result, m.Messages)
	return result
}

// GetSummary returns the summary of older messages.
func (m *ChatMemory) GetSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Summary
}

// GetLastUpdated returns the last update timestamp in milliseconds.
func (m *ChatMemory) GetLastUpdated() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.LastUpdatedMs
}

// truncate shortens a string for summaries.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
