package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	pb "github.com/edgecli/edgecli/proto"
)

// planWrapper is the top-level JSON structure expected from the LLM.
type planWrapper struct {
	Groups []taskGroupJSON `json:"groups"`
	Reduce *reduceJSON     `json:"reduce,omitempty"`
}

type taskGroupJSON struct {
	Index int            `json:"index"`
	Tasks []taskSpecJSON `json:"tasks"`
}

type taskSpecJSON struct {
	TaskID          string `json:"task_id"`
	Kind            string `json:"kind"`
	Input           string `json:"input"`
	TargetDeviceID  string `json:"target_device_id"`
	PromptTokens    int32  `json:"prompt_tokens,omitempty"`
	MaxOutputTokens int32  `json:"max_output_tokens,omitempty"`
}

type reduceJSON struct {
	Kind string `json:"kind"`
}

// validKinds are the allowed task kinds.
var validKinds = map[string]bool{
	"SYSINFO":        true,
	"ECHO":           true,
	"LLM_GENERATE":   true,
	"IMAGE_GENERATE": true,
}

// ParsePlanJSON parses and validates raw LLM output into proto Plan and ReduceSpec.
func ParsePlanJSON(raw string) (*pb.Plan, *pb.ReduceSpec, error) {
	// Strip markdown code fences if present
	cleaned := stripCodeFences(raw)

	var wrapper planWrapper
	if err := json.Unmarshal([]byte(cleaned), &wrapper); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON from LLM: %w", err)
	}

	// Validate groups
	if len(wrapper.Groups) == 0 {
		return nil, nil, fmt.Errorf("plan has no groups")
	}

	seenIDs := make(map[string]bool)
	for gi, g := range wrapper.Groups {
		if len(g.Tasks) == 0 {
			return nil, nil, fmt.Errorf("group %d has no tasks", gi)
		}
		for ti, t := range g.Tasks {
			if t.TaskID == "" {
				return nil, nil, fmt.Errorf("group %d task %d has empty task_id", gi, ti)
			}
			if seenIDs[t.TaskID] {
				return nil, nil, fmt.Errorf("duplicate task_id: %s", t.TaskID)
			}
			seenIDs[t.TaskID] = true

			kind := strings.ToUpper(t.Kind)
			if !validKinds[kind] {
				return nil, nil, fmt.Errorf("group %d task %d (%s): invalid kind %q (allowed: SYSINFO, ECHO, LLM_GENERATE, IMAGE_GENERATE)", gi, ti, t.TaskID, t.Kind)
			}

			// Validate input for path traversal
			if strings.Contains(t.Input, "..") {
				return nil, nil, fmt.Errorf("group %d task %d (%s): input contains path traversal (..)", gi, ti, t.TaskID)
			}
			if len(t.Input) > 0 && (t.Input[0] == '/' || t.Input[0] == '\\') {
				return nil, nil, fmt.Errorf("group %d task %d (%s): input contains absolute path", gi, ti, t.TaskID)
			}
			if len(t.Input) > 1 && t.Input[1] == ':' {
				return nil, nil, fmt.Errorf("group %d task %d (%s): input contains Windows absolute path", gi, ti, t.TaskID)
			}
		}
	}

	// Convert to proto
	protoGroups := make([]*pb.TaskGroup, len(wrapper.Groups))
	for i, g := range wrapper.Groups {
		tasks := make([]*pb.TaskSpec, len(g.Tasks))
		for j, t := range g.Tasks {
			tasks[j] = &pb.TaskSpec{
				TaskId:          t.TaskID,
				Kind:            strings.ToUpper(t.Kind),
				Input:           t.Input,
				TargetDeviceId:  t.TargetDeviceID,
				PromptTokens:    t.PromptTokens,
				MaxOutputTokens: t.MaxOutputTokens,
			}
		}
		protoGroups[i] = &pb.TaskGroup{
			Index: int32(g.Index),
			Tasks: tasks,
		}
	}

	plan := &pb.Plan{Groups: protoGroups}

	// Default reduce to CONCAT
	reduceKind := "CONCAT"
	if wrapper.Reduce != nil && wrapper.Reduce.Kind != "" {
		reduceKind = strings.ToUpper(wrapper.Reduce.Kind)
	}
	reduce := &pb.ReduceSpec{Kind: reduceKind}

	return plan, reduce, nil
}

// stripCodeFences removes surrounding markdown code fences from LLM output.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)

	// Strip ```json ... ``` or ``` ... ```
	if strings.HasPrefix(s, "```") {
		// Find end of first line (the opening fence)
		idx := strings.Index(s, "\n")
		if idx >= 0 {
			s = s[idx+1:]
		}
		// Strip trailing fence
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}

	return s
}
