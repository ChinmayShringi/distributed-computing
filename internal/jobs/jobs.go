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
}

// Job represents a distributed job with multiple tasks
type Job struct {
	ID          string
	CreatedAt   time.Time
	State       JobState
	Tasks       []*Task
	FinalResult string
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
// For v0, creates one SYSINFO task per device
func (m *Manager) CreateJob(devices []*pb.DeviceInfo, maxWorkers int) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobID := uuid.New().String()
	job := &Job{
		ID:        jobID,
		CreatedAt: time.Now(),
		State:     JobQueued,
		Tasks:     make([]*Task, 0),
	}

	// Select devices (limit by maxWorkers if specified)
	selectedDevices := devices
	if maxWorkers > 0 && maxWorkers < len(devices) {
		selectedDevices = devices[:maxWorkers]
	}

	// Create one SYSINFO task per device
	for _, device := range selectedDevices {
		task := &Task{
			ID:         uuid.New().String(),
			JobID:      jobID,
			Kind:       "SYSINFO",
			Input:      "collect_status",
			DeviceID:   device.DeviceId,
			DeviceName: device.DeviceName,
			DeviceAddr: device.GrpcAddr,
			State:      TaskQueued,
		}
		job.Tasks = append(job.Tasks, task)
	}

	m.jobs[jobID] = job
	return job, nil
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
