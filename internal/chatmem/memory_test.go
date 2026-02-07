package chatmem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewMemory(t *testing.T) {
	m := New()
	if m.Version != 1 {
		t.Errorf("expected version 1, got %d", m.Version)
	}
	if len(m.Messages) != 0 {
		t.Errorf("expected empty messages, got %d", len(m.Messages))
	}
}

func TestAddMessage(t *testing.T) {
	m := New()

	m.AddMessage("user", "Hello", "test-device")
	m.AddMessage("assistant", "Hi there!", "test-device")

	if len(m.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(m.Messages))
	}
	if m.Messages[0].Role != "user" {
		t.Errorf("expected user role, got %s", m.Messages[0].Role)
	}
	if m.Messages[1].Role != "assistant" {
		t.Errorf("expected assistant role, got %s", m.Messages[1].Role)
	}
	if m.LastUpdatedMs == 0 {
		t.Error("expected LastUpdatedMs to be set")
	}
}

func TestSummarization(t *testing.T) {
	m := New()

	// Add more than MaxMessages
	for i := 0; i < MaxMessages+4; i++ {
		if i%2 == 0 {
			m.AddMessage("user", "Question "+string(rune('A'+i/2)), "test-device")
		} else {
			m.AddMessage("assistant", "Answer "+string(rune('A'+i/2)), "test-device")
		}
	}

	// In the new async version, AddMessage doesn't summarize.
	// We must trigger it manually.
	done := make(chan bool)
	m.SummarizeAsync(func(curr string, msgs []ChatMessage) (string, error) {
		return "Mock summary", nil
	}, func() {
		done <- true
	})

	<-done

	if len(m.Messages) > MaxMessages {
		t.Errorf("expected at most %d messages, got %d", MaxMessages, len(m.Messages))
	}
	if m.Summary == "" {
		t.Error("expected summary to be set after overflow")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_memory.json")

	// Create and save
	m := New()
	m.path = path
	m.AddMessage("user", "Test question", "test-device")
	m.AddMessage("assistant", "Test answer", "test-device")

	if err := m.SaveToFile(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load and verify
	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if len(loaded.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Test question" {
		t.Errorf("expected 'Test question', got %s", loaded.Messages[0].Content)
	}
}

func TestLoadNonExistent(t *testing.T) {
	m, err := LoadFromFile("/nonexistent/path/memory.json")
	if err != nil {
		t.Fatalf("expected no error for nonexistent file, got: %v", err)
	}
	if len(m.Messages) != 0 {
		t.Errorf("expected empty messages, got %d", len(m.Messages))
	}
}

func TestToJSONAndParse(t *testing.T) {
	m := New()
	m.AddMessage("user", "Hello", "test-device")
	m.AddMessage("assistant", "World", "test-device")

	jsonStr, err := m.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	parsed, err := ParseFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("ParseFromJSON failed: %v", err)
	}

	if len(parsed.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(parsed.Messages))
	}
}

func TestMerge(t *testing.T) {
	m1 := New()
	m1.AddMessage("user", "Old message", "device1")

	m2 := New()
	m2.AddMessage("user", "New message 1", "device2")
	m2.AddMessage("assistant", "New message 2", "device2")
	// Ensure m2 is newer
	m2.LastUpdatedMs = m1.LastUpdatedMs + 1000

	updated := m1.Merge(m2)
	if !updated {
		t.Error("expected merge to return true")
	}
	if len(m1.Messages) != 2 {
		t.Errorf("expected 2 messages after merge, got %d", len(m1.Messages))
	}
	if m1.Messages[0].Content != "New message 1" {
		t.Errorf("expected 'New message 1', got %s", m1.Messages[0].Content)
	}
}

func TestMergeOlder(t *testing.T) {
	m1 := New()
	m1.AddMessage("user", "New message", "device1")

	m2 := New()
	m2.LastUpdatedMs = m1.LastUpdatedMs - 1000 // Older

	updated := m1.Merge(m2)
	if updated {
		t.Error("expected merge to return false for older data")
	}
}

func TestCreateDirectoryOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "dir", "memory.json")

	m := New()
	m.path = path
	m.AddMessage("user", "Test", "test-device")

	if err := m.SaveToFile(); err != nil {
		t.Fatalf("failed to save with nested dir: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to exist after save")
	}
}
