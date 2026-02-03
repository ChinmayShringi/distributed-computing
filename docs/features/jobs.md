# Distributed Job System

The job system enables parallel task execution across multiple devices with sequential group ordering and result aggregation.

## Concepts

### Job
A job is a collection of tasks distributed across devices. Jobs have states: `QUEUED`, `RUNNING`, `DONE`, `FAILED`.

### Task Group
Tasks are organized into groups. Groups execute **sequentially** (group 0 completes before group 1 starts). This enables multi-step workflows.

### Task
A task is a unit of work executed on a single device. Tasks within a group execute **in parallel**.

### Reduce
After all groups complete, results are combined using a reduce operation. Currently supports `CONCAT` (concatenate all results).

## Execution Model

```
Job
├── Group 0 (executes first)
│   ├── Task A → Device 1 ─┐
│   └── Task B → Device 2 ─┴── parallel
│
├── Group 1 (waits for Group 0)
│   ├── Task C → Device 1 ─┐
│   └── Task D → Device 2 ─┴── parallel
│
└── Reduce (CONCAT) → Final Result
```

## Task Types

### SYSINFO
Collects system information from the device.

**Output:**
```
Device: hostname
Device ID: uuid
Platform: darwin/arm64
Memory: 16384 MB total, 8192 MB used
Time: 2026-02-03T15:00:00-05:00
```

### ECHO
Returns the input string (for testing).

**Input:** `"hello"`
**Output:** `"echo: hello"`

## API

### Submit Job

```bash
curl -X POST http://localhost:8080/api/submit-job \
  -H "Content-Type: application/json" \
  -d '{"text":"collect status","max_workers":0}'
```

**Parameters:**
- `text` - Job description
- `max_workers` - Maximum devices to use (0 = all)

**Response:**
```json
{
  "job_id": "8155c378-f55a-44ff-9b78-6dc60f129190",
  "created_at": 1770149334,
  "summary": "distributed to 2 devices in 1 group(s)"
}
```

### Get Job Status

```bash
curl "http://localhost:8080/api/job?id=8155c378-f55a-44ff-9b78-6dc60f129190"
```

**Response:**
```json
{
  "job_id": "8155c378-...",
  "state": "DONE",
  "tasks": [
    {
      "task_id": "3f0b6a31-...",
      "assigned_device_id": "e452458d-...",
      "assigned_device_name": "macbook-pro",
      "state": "DONE",
      "result": "Device: macbook-pro\n...",
      "error": ""
    },
    {
      "task_id": "2fad5fee-...",
      "assigned_device_id": "f66a8dc8-...",
      "assigned_device_name": "windows-pc",
      "state": "DONE",
      "result": "Device: windows-pc\n...",
      "error": ""
    }
  ],
  "final_result": "=== macbook-pro ===\n...\n\n=== windows-pc ===\n...",
  "current_group": 0,
  "total_groups": 1
}
```

## Job States

| State | Description |
|-------|-------------|
| `QUEUED` | Job created, not yet started |
| `RUNNING` | Tasks are being executed |
| `DONE` | All tasks completed successfully |
| `FAILED` | One or more tasks failed |

## Task States

| State | Description |
|-------|-------------|
| `QUEUED` | Task waiting to execute |
| `RUNNING` | Task currently executing |
| `DONE` | Task completed successfully |
| `FAILED` | Task failed with error |

## Internal Implementation

### Job Manager (`internal/jobs/jobs.go`)

```go
type Job struct {
    ID           string
    CreatedAt    time.Time
    State        JobState
    Tasks        []*Task
    FinalResult  string
    CurrentGroup int
    TotalGroups  int
    ReduceSpec   *ReduceSpec
}

type Task struct {
    ID         string
    JobID      string
    Kind       string      // "SYSINFO", "ECHO"
    Input      string
    DeviceID   string
    DeviceName string
    DeviceAddr string
    State      TaskState
    Result     string
    Error      string
    GroupIndex int
}
```

### Key Methods

- `CreateJob(devices, maxWorkers, plan, reduce)` - Create job with auto-generated or explicit plan
- `SetJobRunning(jobID)` - Mark job as running
- `UpdateTask(jobID, taskID, state, result, error)` - Update task state
- `SetCurrentGroup(jobID, groupIndex)` - Track current group
- `IsGroupComplete(jobID, groupIndex)` - Check if group finished
- `IsGroupFailed(jobID, groupIndex)` - Check if any task in group failed
- `GetTasksForGroup(jobID, groupIndex)` - Get tasks for a group
- `SetJobDone(jobID, finalResult)` - Mark job complete

### Server Execution (`cmd/server/main.go`)

```go
func (s *OrchestratorServer) executeJobGroups(job *jobs.Job) {
    for groupIdx := 0; groupIdx < job.TotalGroups; groupIdx++ {
        // Execute all tasks in group in parallel
        groupResults := s.executeTaskGroup(job, groupTasks)

        // Stop if group failed
        if s.jobManager.IsGroupFailed(job.ID, groupIdx) {
            break
        }
    }

    // Apply reduce
    finalResult := s.applyReduce(job.ReduceSpec, allResults)
    s.jobManager.SetJobDone(job.ID, finalResult)
}
```

## Custom Plans (Advanced)

Jobs can include explicit execution plans via the gRPC API:

```protobuf
message JobRequest {
  string session_id = 1;
  string text = 2;
  int32 max_workers = 3;
  Plan plan = 4;           // Custom plan
  ReduceSpec reduce = 5;   // Custom reduce
}
```

If no plan is provided, the system auto-generates a default plan with one SYSINFO task per device in a single group.

## Error Handling

- If a task fails, its error is recorded but other tasks in the group continue
- Job state shows `DONE` even if some tasks failed (check individual task states)
- `final_result` includes warning if tasks failed: `"Warning: N task(s) failed\n\n..."`
