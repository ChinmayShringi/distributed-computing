package llm

import (
	"encoding/json"
	"testing"
)

func TestToolDefinitions_Valid(t *testing.T) {
	// Verify all tool definitions are valid JSON
	for _, tool := range ToolDefinitions {
		if tool.Type != "function" {
			t.Errorf("Tool %s: expected type 'function', got %s", tool.Function.Name, tool.Type)
		}

		if tool.Function.Name == "" {
			t.Error("Tool has empty name")
		}

		if tool.Function.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Function.Name)
		}

		// Verify parameters is valid JSON
		var params map[string]interface{}
		if err := json.Unmarshal(tool.Function.Parameters, &params); err != nil {
			t.Errorf("Tool %s: invalid parameters JSON: %v", tool.Function.Name, err)
		}

		// Verify required fields in parameters schema
		if params["type"] != "object" {
			t.Errorf("Tool %s: parameters.type should be 'object'", tool.Function.Name)
		}
	}
}

func TestToolDefinitions_ExpectedTools(t *testing.T) {
	expected := []string{"get_capabilities", "execute_shell_cmd", "get_file"}

	for _, name := range expected {
		found := false
		for _, tool := range ToolDefinitions {
			if tool.Function.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %q not found in ToolDefinitions", name)
		}
	}
}

func TestGetToolDefinitions(t *testing.T) {
	defs := GetToolDefinitions()
	if len(defs) != len(ToolDefinitions) {
		t.Errorf("GetToolDefinitions() returned %d tools, expected %d", len(defs), len(ToolDefinitions))
	}

	// Verify it returns a copy
	defs[0].Function.Name = "modified"
	if ToolDefinitions[0].Function.Name == "modified" {
		t.Error("GetToolDefinitions() should return a copy")
	}
}

func TestGetToolByName(t *testing.T) {
	// Test existing tool
	tool := GetToolByName("get_capabilities")
	if tool == nil {
		t.Error("GetToolByName('get_capabilities') should return a tool")
	}
	if tool.Function.Name != "get_capabilities" {
		t.Errorf("GetToolByName('get_capabilities') returned wrong tool: %s", tool.Function.Name)
	}

	// Test non-existing tool
	tool = GetToolByName("non_existent_tool")
	if tool != nil {
		t.Error("GetToolByName('non_existent_tool') should return nil")
	}
}

func TestListToolNames(t *testing.T) {
	names := ListToolNames()
	if len(names) != len(ToolDefinitions) {
		t.Errorf("ListToolNames() returned %d names, expected %d", len(names), len(ToolDefinitions))
	}

	expected := map[string]bool{
		"get_capabilities":  true,
		"execute_shell_cmd": true,
		"get_file":          true,
	}

	for _, name := range names {
		if !expected[name] {
			t.Errorf("Unexpected tool name: %s", name)
		}
	}
}

func TestGetCapabilitiesSchema(t *testing.T) {
	tool := GetToolByName("get_capabilities")
	if tool == nil {
		t.Fatal("get_capabilities tool not found")
	}

	var params map[string]interface{}
	if err := json.Unmarshal(tool.Function.Parameters, &params); err != nil {
		t.Fatalf("Failed to parse get_capabilities parameters: %v", err)
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("get_capabilities should have properties")
	}

	// Check include_benchmarks property exists
	if _, ok := props["include_benchmarks"]; !ok {
		t.Error("get_capabilities should have include_benchmarks property")
	}
}

func TestExecuteShellCmdSchema(t *testing.T) {
	tool := GetToolByName("execute_shell_cmd")
	if tool == nil {
		t.Fatal("execute_shell_cmd tool not found")
	}

	var params map[string]interface{}
	if err := json.Unmarshal(tool.Function.Parameters, &params); err != nil {
		t.Fatalf("Failed to parse execute_shell_cmd parameters: %v", err)
	}

	// Check required fields
	required, ok := params["required"].([]interface{})
	if !ok {
		t.Fatal("execute_shell_cmd should have required array")
	}

	hasCommand := false
	for _, r := range required {
		if r == "command" {
			hasCommand = true
			break
		}
	}
	if !hasCommand {
		t.Error("execute_shell_cmd should require 'command' parameter")
	}
}

func TestGetFileSchema(t *testing.T) {
	tool := GetToolByName("get_file")
	if tool == nil {
		t.Fatal("get_file tool not found")
	}

	var params map[string]interface{}
	if err := json.Unmarshal(tool.Function.Parameters, &params); err != nil {
		t.Fatalf("Failed to parse get_file parameters: %v", err)
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("get_file should have properties")
	}

	// Check read_mode property has enum
	readMode, ok := props["read_mode"].(map[string]interface{})
	if !ok {
		t.Fatal("get_file should have read_mode property")
	}

	enum, ok := readMode["enum"].([]interface{})
	if !ok {
		t.Fatal("read_mode should have enum")
	}

	expectedModes := map[string]bool{"full": false, "head": false, "tail": false, "range": false}
	for _, mode := range enum {
		if s, ok := mode.(string); ok {
			expectedModes[s] = true
		}
	}

	for mode, found := range expectedModes {
		if !found {
			t.Errorf("read_mode enum should include %q", mode)
		}
	}
}
