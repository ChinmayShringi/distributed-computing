//go:build !windows

package brain

import (
	"fmt"

	pb "github.com/edgecli/edgecli/proto"
)

// isAvailable always returns false on non-Windows platforms.
func isAvailable(b *Brain) bool {
	return false
}

// generatePlan returns an error on non-Windows platforms.
func generatePlan(b *Brain, text string, devices []*pb.DeviceInfo, maxWorkers int) (*pb.Plan, *pb.ReduceSpec, error) {
	return nil, nil, fmt.Errorf("Windows AI CLI is only available on Windows")
}

// summarize returns an error on non-Windows platforms.
func summarize(b *Brain, text string) (string, bool, error) {
	return "", false, fmt.Errorf("Windows AI CLI is only available on Windows")
}
