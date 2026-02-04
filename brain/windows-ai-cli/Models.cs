using System.Text.Json.Serialization;

namespace WindowsAiCli;

// ============================================================================
// Request Models
// ============================================================================

/// <summary>
/// Input for the 'plan' command. Contains job context and available devices.
/// </summary>
public record PlanRequest
{
    [JsonPropertyName("text")]
    public string Text { get; init; } = "";

    [JsonPropertyName("max_workers")]
    public int MaxWorkers { get; init; }

    [JsonPropertyName("devices")]
    public List<DeviceInfo> Devices { get; init; } = new();
}

/// <summary>
/// Device information from the orchestrator registry.
/// </summary>
public record DeviceInfo
{
    [JsonPropertyName("device_id")]
    public string DeviceId { get; init; } = "";

    [JsonPropertyName("device_name")]
    public string DeviceName { get; init; } = "";

    [JsonPropertyName("has_npu")]
    public bool HasNpu { get; init; }

    [JsonPropertyName("has_gpu")]
    public bool HasGpu { get; init; }

    [JsonPropertyName("has_cpu")]
    public bool HasCpu { get; init; } = true;

    [JsonPropertyName("grpc_addr")]
    public string GrpcAddr { get; init; } = "";
}

// ============================================================================
// Response Models (matching proto/orchestrator.proto)
// ============================================================================

/// <summary>
/// Execution plan containing sequential task groups.
/// </summary>
public record Plan
{
    [JsonPropertyName("groups")]
    public List<TaskGroup> Groups { get; init; } = new();
}

/// <summary>
/// A group of tasks that execute in parallel. Groups execute sequentially.
/// </summary>
public record TaskGroup
{
    [JsonPropertyName("index")]
    public int Index { get; init; }

    [JsonPropertyName("tasks")]
    public List<TaskSpec> Tasks { get; init; } = new();
}

/// <summary>
/// Specification for a single task to execute on a device.
/// </summary>
public record TaskSpec
{
    [JsonPropertyName("task_id")]
    public string TaskId { get; init; } = "";

    [JsonPropertyName("kind")]
    public string Kind { get; init; } = "";

    [JsonPropertyName("input")]
    public string Input { get; init; } = "";

    [JsonPropertyName("target_device_id")]
    public string TargetDeviceId { get; init; } = "";
}

/// <summary>
/// Specification for reducing task results.
/// </summary>
public record ReduceSpec
{
    [JsonPropertyName("kind")]
    public string Kind { get; init; } = "CONCAT";
}

// ============================================================================
// CLI Response Models
// ============================================================================

/// <summary>
/// Response for the 'capabilities' command.
/// </summary>
public record CapabilitiesResponse
{
    [JsonPropertyName("ok")]
    public bool Ok { get; init; }

    [JsonPropertyName("features")]
    public List<string> Features { get; init; } = new();

    [JsonPropertyName("notes")]
    public string Notes { get; init; } = "";

    [JsonPropertyName("error")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Error { get; init; }
}

/// <summary>
/// Response for the 'plan' command.
/// </summary>
public record PlanResponse
{
    [JsonPropertyName("ok")]
    public bool Ok { get; init; }

    [JsonPropertyName("plan")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public Plan? Plan { get; init; }

    [JsonPropertyName("reduce")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public ReduceSpec? Reduce { get; init; }

    [JsonPropertyName("error")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Error { get; init; }
}

/// <summary>
/// Response for the 'summarize' command.
/// </summary>
public record SummarizeResponse
{
    [JsonPropertyName("ok")]
    public bool Ok { get; init; }

    [JsonPropertyName("summary")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Summary { get; init; }

    [JsonPropertyName("used_ai")]
    public bool UsedAi { get; init; }

    [JsonPropertyName("error")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Error { get; init; }
}

/// <summary>
/// Generic error response.
/// </summary>
public record ErrorResponse
{
    [JsonPropertyName("ok")]
    public bool Ok { get; init; } = false;

    [JsonPropertyName("error")]
    public string Error { get; init; } = "";
}
