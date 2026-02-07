package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/edgecli/edgecli/internal/llm"
	pb "github.com/edgecli/edgecli/proto"
)

// generatePlanWithLLM calls the LLM provider to generate an execution plan.
func (s *OrchestratorServer) generatePlanWithLLM(userText string, devices []*pb.DeviceInfo, maxWorkers int) (*pb.Plan, *pb.ReduceSpec, error) {
	// Serialize devices to JSON for LLM
	devicesJSON, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal devices: %w", err)
	}

	// Call LLM provider with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	planJSON, err := s.llmProvider.Plan(ctx, userText, string(devicesJSON))
	if err != nil {
		return nil, nil, fmt.Errorf("LLM provider plan call: %w", err)
	}

	// Parse the plan JSON
	plan, reduce, err := llm.ParsePlanJSON(planJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("parse LLM plan: %w", err)
	}

	return plan, reduce, nil
}
