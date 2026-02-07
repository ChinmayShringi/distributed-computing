using System.CommandLine;
using System.Text.Json;

namespace WindowsAiCli;

class Program
{
    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        WriteIndented = false,
        PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower
    };

    static async Task<int> Main(string[] args)
    {
        var rootCommand = new RootCommand("Windows AI CLI - Plan generation and text processing");

        // capabilities command
        var capabilitiesCommand = new Command("capabilities", "Check Windows AI availability and supported features");
        capabilitiesCommand.SetHandler(HandleCapabilities);
        rootCommand.AddCommand(capabilitiesCommand);

        // plan command
        var planCommand = new Command("plan", "Generate an execution plan from JSON input");
        var inOption = new Option<FileInfo?>("--in", "Path to JSON file containing PlanRequest");
        var formatOption = new Option<string>("--format", () => "json", "Output format (json)");
        planCommand.AddOption(inOption);
        planCommand.AddOption(formatOption);
        planCommand.SetHandler(HandlePlan, inOption, formatOption);
        rootCommand.AddCommand(planCommand);

        // summarize command
        var summarizeCommand = new Command("summarize", "Summarize input text");
        var textOption = new Option<string>("--text", "Text to summarize") { IsRequired = true };
        var summaryFormatOption = new Option<string>("--format", () => "json", "Output format (json)");
        summarizeCommand.AddOption(textOption);
        summarizeCommand.AddOption(summaryFormatOption);
        summarizeCommand.SetHandler(HandleSummarize, textOption, summaryFormatOption);
        rootCommand.AddCommand(summarizeCommand);

        return await rootCommand.InvokeAsync(args);
    }

    /// <summary>
    /// Handle the 'capabilities' command - detect Windows AI availability.
    /// </summary>
    private static void HandleCapabilities()
    {
        try
        {
            var (available, notes) = CheckWindowsAiAvailability();

            var response = new CapabilitiesResponse
            {
                Ok = true,
                Features = new List<string> { "plan", "summarize" },
                Notes = available
                    ? $"Windows AI available. {notes}"
                    : $"Windows AI not available (fallback mode). {notes}"
            };

            Console.WriteLine(JsonSerializer.Serialize(response, JsonOptions));
        }
        catch (Exception ex)
        {
            WriteError($"Failed to check capabilities: {ex.Message}");
        }
    }

    /// <summary>
    /// Handle the 'plan' command - generate execution plan from input.
    /// </summary>
    private static void HandlePlan(FileInfo? inputFile, string format)
    {
        try
        {
            if (inputFile == null || !inputFile.Exists)
            {
                WriteError("Input file not specified or does not exist. Use --in <path>");
                return;
            }

            var json = File.ReadAllText(inputFile.FullName);
            var request = JsonSerializer.Deserialize<PlanRequest>(json, JsonOptions);

            if (request == null)
            {
                WriteError("Failed to parse input JSON");
                return;
            }

            // Generate plan (fallback-first design)
            var plan = GenerateDeterministicPlan(request);
            var reduce = new ReduceSpec { Kind = "CONCAT" };

            // Check AI availability for metadata
            var (aiAvailable, aiNotes) = CheckWindowsAiAvailability();
            bool usedAi = false;

            if (aiAvailable)
            {
                // Future: AI could validate/optimize the plan here
                // For now, we just use the deterministic plan
            }

            // Build rationale based on what the plan does
            var totalDevices = request.Devices.Count;
            var selectedDevices = request.MaxWorkers > 0 && request.MaxWorkers < totalDevices
                ? request.MaxWorkers : totalDevices;
            var rationale = $"Deterministic: 1 SYSINFO task per device ({selectedDevices} of {totalDevices} devices selected"
                + (request.MaxWorkers > 0 ? $", max_workers={request.MaxWorkers}" : "") + ")";

            var response = new PlanResponse
            {
                Ok = true,
                UsedAi = usedAi,
                Notes = aiAvailable ? $"AI available. {aiNotes}" : $"Fallback mode. {aiNotes}",
                Rationale = rationale,
                Plan = plan,
                Reduce = reduce
            };

            Console.WriteLine(JsonSerializer.Serialize(response, JsonOptions));
        }
        catch (JsonException ex)
        {
            WriteError($"Invalid JSON input: {ex.Message}");
        }
        catch (Exception ex)
        {
            WriteError($"Plan generation failed: {ex.Message}");
        }
    }

    /// <summary>
    /// Handle the 'summarize' command - summarize input text.
    /// </summary>
    private static void HandleSummarize(string text, string format)
    {
        try
        {
            if (string.IsNullOrWhiteSpace(text))
            {
                WriteError("Text cannot be empty");
                return;
            }

            var (aiAvailable, aiNotes) = CheckWindowsAiAvailability();
            string summary;
            bool usedAi = false;

            if (aiAvailable)
            {
                // Future: Use Windows AI for summarization
                // For now, fall through to fallback
                summary = GenerateFallbackSummary(text);
                usedAi = false;
            }
            else
            {
                summary = GenerateFallbackSummary(text);
                usedAi = false;
            }

            var response = new SummarizeResponse
            {
                Ok = true,
                Summary = summary,
                UsedAi = usedAi,
                Notes = aiAvailable ? $"AI available. {aiNotes}" : $"Fallback mode. {aiNotes}"
            };

            Console.WriteLine(JsonSerializer.Serialize(response, JsonOptions));
        }
        catch (Exception ex)
        {
            WriteError($"Summarization failed: {ex.Message}");
        }
    }

    /// <summary>
    /// Check if Windows AI APIs are available at runtime.
    /// </summary>
    private static (bool Available, string Notes) CheckWindowsAiAvailability()
    {
        try
        {
            // Check Windows version - Windows AI features require Windows 11 24H2+
            var osVersion = Environment.OSVersion.Version;

            // Windows 11 is version 10.0.22000+
            // Windows 11 24H2 is version 10.0.26100+
            if (osVersion.Major < 10 || (osVersion.Major == 10 && osVersion.Build < 22000))
            {
                return (false, "Requires Windows 11 or later");
            }

            if (osVersion.Build < 26100)
            {
                return (false, $"Windows build {osVersion.Build} detected. Windows AI requires build 26100+ (24H2)");
            }

            // Try to check for Windows AI namespace availability
            // This is a runtime check that gracefully fails on older Windows
            try
            {
                // Check if Windows.AI.MachineLearning is available
                var mlType = Type.GetType("Windows.AI.MachineLearning.LearningModel, Windows.AI.MachineLearning, Version=1.0.0.0, Culture=neutral, PublicKeyToken=null");
                if (mlType != null)
                {
                    return (true, "Windows.AI.MachineLearning available");
                }
            }
            catch
            {
                // Type not available
            }

            return (false, $"Windows build {osVersion.Build} detected but AI APIs not found");
        }
        catch (Exception ex)
        {
            return (false, $"Detection error: {ex.Message}");
        }
    }

    /// <summary>
    /// Generate a deterministic execution plan based on available devices.
    /// This is the primary (fallback-first) plan generation logic.
    /// </summary>
    private static Plan GenerateDeterministicPlan(PlanRequest request)
    {
        var devices = request.Devices;
        var maxWorkers = request.MaxWorkers;

        // Respect max_workers limit
        if (maxWorkers > 0 && maxWorkers < devices.Count)
        {
            devices = devices.Take(maxWorkers).ToList();
        }

        // Create one SYSINFO task per device
        var tasks = new List<TaskSpec>();
        for (int i = 0; i < devices.Count; i++)
        {
            var device = devices[i];
            tasks.Add(new TaskSpec
            {
                TaskId = Guid.NewGuid().ToString(),
                Kind = "SYSINFO",
                Input = "collect_status",
                TargetDeviceId = device.DeviceId
            });
        }

        // Single group with all tasks (parallel execution)
        var group = new TaskGroup
        {
            Index = 0,
            Tasks = tasks
        };

        return new Plan
        {
            Groups = new List<TaskGroup> { group }
        };
    }

    /// <summary>
    /// Generate a fallback summary (first N characters).
    /// </summary>
    private static string GenerateFallbackSummary(string text)
    {
        const int maxLength = 200;

        if (string.IsNullOrWhiteSpace(text))
        {
            return "";
        }

        text = text.Trim();

        if (text.Length <= maxLength)
        {
            return text;
        }

        // Truncate at word boundary if possible
        var truncated = text.Substring(0, maxLength);
        var lastSpace = truncated.LastIndexOf(' ');

        if (lastSpace > maxLength / 2)
        {
            truncated = truncated.Substring(0, lastSpace);
        }

        return truncated + "...";
    }

    /// <summary>
    /// Write an error response as JSON to stdout.
    /// </summary>
    private static void WriteError(string message)
    {
        var response = new ErrorResponse
        {
            Ok = false,
            Error = message
        };
        Console.WriteLine(JsonSerializer.Serialize(response, JsonOptions));
    }
}
