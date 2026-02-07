package approval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Rule represents an allow rule for auto-approval
type Rule struct {
	Type      string    `json:"type"`       // "prefix", "exact", "path"
	Pattern   string    `json:"pattern"`    // The pattern to match
	Tool      string    `json:"tool"`       // Tool name (e.g., "bash")
	CreatedAt time.Time `json:"created_at"` // When the rule was created
	UsedCount int       `json:"used_count"` // How many times this rule was used
}

// Rules manages the collection of allow rules
type Rules struct {
	Rules    []Rule `json:"rules"`
	filePath string
}

// LoadRules loads rules from the config file
func LoadRules() (*Rules, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(configDir, "approvals.json")
	rules := &Rules{
		Rules:    []Rule{},
		filePath: filePath,
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return rules, nil // No rules file yet, return empty
		}
		return nil, err
	}

	if err := json.Unmarshal(data, rules); err != nil {
		return nil, err
	}

	rules.filePath = filePath
	return rules, nil
}

// Save persists the rules to the config file
func (r *Rules) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0600)
}

// AddRule adds a new rule from a scope string (e.g., "prefix:ls", "path:/tmp")
func (r *Rules) AddRule(tool, scope string) error {
	parts := strings.SplitN(scope, ":", 2)
	if len(parts) != 2 {
		return nil // Invalid scope format
	}

	ruleType := parts[0]
	pattern := parts[1]

	// Check if rule already exists
	for _, existing := range r.Rules {
		if existing.Type == ruleType && existing.Pattern == pattern && existing.Tool == tool {
			return nil // Already exists
		}
	}

	rule := Rule{
		Type:      ruleType,
		Pattern:   pattern,
		Tool:      tool,
		CreatedAt: time.Now(),
		UsedCount: 0,
	}

	r.Rules = append(r.Rules, rule)
	return r.Save()
}

// IsAllowed checks if an action is allowed by any rule
func (r *Rules) IsAllowed(action *Action) bool {
	if r == nil || len(r.Rules) == 0 {
		return false
	}

	for i, rule := range r.Rules {
		if rule.Tool != action.Tool {
			continue
		}

		if r.matchRule(&rule, action) {
			// Update usage count
			r.Rules[i].UsedCount++
			r.Save() // Best effort save
			return true
		}
	}

	return false
}

// matchRule checks if a specific rule matches an action
func (r *Rules) matchRule(rule *Rule, action *Action) bool {
	switch rule.Type {
	case "prefix":
		return matchPrefix(rule.Pattern, action)
	case "exact":
		return matchExact(rule.Pattern, action)
	case "path":
		return matchPath(rule.Pattern, action)
	default:
		return false
	}
}

// matchPrefix checks if the command starts with the given prefix
func matchPrefix(pattern string, action *Action) bool {
	if action.Command == "" {
		return false
	}

	// Get the first word of the command (excluding sudo)
	args := parseCommand(action.Command)
	if len(args) == 0 {
		return false
	}

	cmdStart := args[0]
	if cmdStart == "sudo" && len(args) > 1 {
		cmdStart = args[1]
	}

	return cmdStart == pattern || strings.HasPrefix(action.Command, pattern+" ")
}

// matchExact checks if the command exactly matches the pattern
func matchExact(pattern string, action *Action) bool {
	return action.Command == pattern
}

// matchPath checks if the command operates within the given path
func matchPath(pattern string, action *Action) bool {
	if action.Command == "" {
		return false
	}

	// Expand ~ to home directory
	expandedPattern := expandPath(pattern)

	// Check if any argument in the command is within the allowed path
	args := parseCommand(action.Command)
	for _, arg := range args {
		expandedArg := expandPath(arg)
		if strings.HasPrefix(expandedArg, expandedPattern) {
			return true
		}
	}

	return false
}

// expandPath expands ~ to the home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ListRules returns a formatted list of rules for display
func (r *Rules) ListRules() []string {
	var result []string
	for _, rule := range r.Rules {
		desc := ""
		switch rule.Type {
		case "prefix":
			desc = "Commands starting with '" + rule.Pattern + "'"
		case "exact":
			desc = "Exact command: '" + rule.Pattern + "'"
		case "path":
			desc = "Commands in path: '" + rule.Pattern + "'"
		}
		result = append(result, desc)
	}
	return result
}

// RemoveRule removes a rule by index
func (r *Rules) RemoveRule(index int) error {
	if index < 0 || index >= len(r.Rules) {
		return nil
	}
	r.Rules = append(r.Rules[:index], r.Rules[index+1:]...)
	return r.Save()
}

// ClearRules removes all rules
func (r *Rules) ClearRules() error {
	r.Rules = []Rule{}
	return r.Save()
}

// getConfigDir returns the config directory path
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".omniforge"), nil
}
