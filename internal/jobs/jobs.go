// Package jobs provides in-memory job and task management for distributed execution
package jobs

import (
	"sync"
	"time"

	"github.com/google/uuid"

	pb "github.com/edgecli/edgecli/proto"
)

// TaskState represents the state of a task
type TaskState string

const (
	TaskQueued  TaskState = "QUEUED"
	TaskRunning TaskState = "RUNNING"
	TaskDone    TaskState = "DONE"
	TaskFailed  TaskState = "FAILED"
)

// JobState represents the state of a job
type JobState string

const (
	JobQueued  JobState = "QUEUED"
	JobRunning JobState = "RUNNING"
	JobDone    JobState = "DONE"
	JobFailed  JobState = "FAILED"
)

// Task represents a unit of work to be executed on a device
type Task struct {
	ID         string
	JobID      string
	Kind       string // "SYSINFO" or "ECHO"
	Input      string
	DeviceID   string
	DeviceName string
	DeviceAddr string
	State      TaskState
	Result     string
	Error      string
	GroupIndex int // which group this task belongs to
}

// ReduceSpec specifies how to combine results
type ReduceSpec struct {
	Kind string // "CONCAT" for now
}

// Job represents a distributed job with multiple tasks
type Job struct {
	ID           string
	CreatedAt    time.Time
	State        JobState
	Tasks        []*Task
	FinalResult  string
	CurrentGroup int         // which group is currently executing
	TotalGroups  int         // total number of groups
	ReduceSpec   *ReduceSpec // how to combine results
}

// Manager manages jobs and their tasks in-memory
type Manager struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewManager creates a new job manager
func NewManager() *Manager {
	return &Manager{
		jobs: make(map[string]*Job),
	}
}

// CreateJob creates a new job with tasks distributed across devices
// If no plan provided, auto-generates a default plan with one SYSINFO task per device
func (m *Manager) CreateJob(devices []*pb.DeviceInfo, maxWorkers int, plan *pb.Plan, reduce *pb.ReduceSpec) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobID := uuid.New().String()

	// Default reduce to CONCAT if not specified
	reduceSpec := &ReduceSpec{Kind: "CONCAT"}
	if reduce != nil && reduce.Kind != "" {
		reduceSpec.Kind = reduce.Kind
	}

	job := &Job{
		ID:           jobID,
		CreatedAt:    time.Now(),
		State:        JobQueued,
		Tasks:        make([]*Task, 0),
		CurrentGroup: 0,
		ReduceSpec:   reduceSpec,
	}

	// Select devices (limit by maxWorkers if specified)
	selectedDevices := devices
	if maxWorkers > 0 && maxWorkers < len(devices) {
		selectedDevices = devices[:maxWorkers]
	}

	// Build device lookup map
	deviceMap := make(map[string]*pb.DeviceInfo)
	for _, d := range devices {
		deviceMap[d.DeviceId] = d
	}

	// If no plan provided, generate default plan
	if plan == nil || len(plan.Groups) == 0 {
		plan = m.generateDefaultPlan(selectedDevices)
	}

	job.TotalGroups = len(plan.Groups)

	// Convert plan to tasks
	for _, group := range plan.Groups {
		for _, taskSpec := range group.Tasks {
			// Find target device
			var device *pb.DeviceInfo
			if taskSpec.TargetDeviceId != "" {
				device = deviceMap[taskSpec.TargetDeviceId]
			}
			if device == nil && len(selectedDevices) > 0 {
				// Assign to first available device if not specified
				device = selectedDevices[0]
			}

			deviceName := ""
			deviceAddr := ""
			deviceID := ""
			if device != nil {
				deviceName = device.DeviceName
				deviceAddr = device.GrpcAddr
				deviceID = device.DeviceId
			}

			taskID := taskSpec.TaskId
			if taskID == "" {
				taskID = uuid.New().String()
			}

			task := &Task{
				ID:         taskID,
				JobID:      jobID,
				Kind:       taskSpec.Kind,
				Input:      taskSpec.Input,
				DeviceID:   deviceID,
				DeviceName: deviceName,
				DeviceAddr: deviceAddr,
				State:      TaskQueued,
				GroupIndex: int(group.Index),
			}
			job.Tasks = append(job.Tasks, task)
		}
	}

	m.jobs[jobID] = job
	return job, nil
}

// generateDefaultPlan creates a default plan with one SYSINFO task per device
func (m *Manager) generateDefaultPlan(devices []*pb.DeviceInfo) *pb.Plan {
	tasks := make([]*pb.TaskSpec, len(devices))
	for i, d := range devices {
		tasks[i] = &pb.TaskSpec{
			TaskId:         uuid.New().String(),
			Kind:           "SYSINFO",
			Input:          "collect_status",
			TargetDeviceId: d.DeviceId,
		}
	}

	return &pb.Plan{
		Groups: []*pb.TaskGroup{
			{Index: 0, Tasks: tasks},
		},
	}
}

// Get retrieves a job by ID
func (m *Manager) Get(jobID string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	return job, ok
}

// SetJobRunning marks a job as running
func (m *Manager) SetJobRunning(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.State = JobRunning
	}
}

// UpdateTask updates the state and result of a task
func (m *Manager) UpdateTask(jobID, taskID string, state TaskState, result, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return
	}

	for _, task := range job.Tasks {
		if task.ID == taskID {
			task.State = state
			task.Result = result
			task.Error = errMsg
			break
		}
	}
}

// SetJobDone marks a job as done with the final result
func (m *Manager) SetJobDone(jobID, finalResult string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.State = JobDone
		job.FinalResult = finalResult
	}
}

// SetJobFailed marks a job as failed with an error message
func (m *Manager) SetJobFailed(jobID, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.State = JobFailed
		job.FinalResult = "Job failed: " + errMsg
	}
}

// SetCurrentGroup updates the current group being executed
func (m *Manager) SetCurrentGroup(jobID string, groupIndex int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.CurrentGroup = groupIndex
	}
}

// GetTasksForGroup returns all tasks in a specific group
func (m *Manager) GetTasksForGroup(jobID string, groupIndex int) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil
	}

	var tasks []*Task
	for _, task := range job.Tasks {
		if task.GroupIndex == groupIndex {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// IsGroupComplete checks if all tasks in a group are done or failed
func (m *Manager) IsGroupComplete(jobID string, groupIndex int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return false
	}

	for _, task := range job.Tasks {
		if task.GroupIndex == groupIndex {
			if task.State != TaskDone && task.State != TaskFailed {
				return false
			}
		}
	}
	return true
}

// IsGroupFailed checks if any task in a group failed
func (m *Manager) IsGroupFailed(jobID string, groupIndex int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return false
	}

	for _, task := range job.Tasks {
		if task.GroupIndex == groupIndex && task.State == TaskFailed {
			return true
		}
	}
	return false
}

// GetGroupResults returns the results of all completed tasks in a group
func (m *Manager) GetGroupResults(jobID string, groupIndex int) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil
	}

	var results []string
	for _, task := range job.Tasks {
		if task.GroupIndex == groupIndex && task.State == TaskDone && task.Result != "" {
			results = append(results, task.Result)
		}
	}
	return results
}
