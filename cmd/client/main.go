// Package main implements the gRPC orchestrator CLI client
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/edgecli/edgecli/internal/deviceid"
	pb "github.com/edgecli/edgecli/proto"
)

type arrayFlags []string

func (a *arrayFlags) String() string {
	return fmt.Sprintf("%v", *a)
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: client [global flags] [command] [command flags]

Global flags:
  --addr string    Server address (default "localhost:50051")
  --key string     Security key (required for some commands)

Commands:
  register         Register this device to the server registry
  list-devices     List all registered devices
  status           Get device status
  route-task       Route an AI task to the best device
  routed-cmd       Execute command on best available device (routed)

Legacy mode (without subcommand):
  --cmd string     Command to execute locally (requires --key)
  --arg string     Command arguments (repeatable)

Examples:
  # Register this device
  client register --name my-laptop --self-addr 192.168.1.10:50052

  # List all devices
  client list-devices

  # Get device status
  client status --id <device-id>

  # Route an AI task
  client --key dev route-task --task summarize --input "hello world"

  # Execute routed command (auto-selects best device)
  client --key dev routed-cmd --cmd ls

  # Execute routed command with policy
  client --key dev routed-cmd --cmd pwd --prefer-remote
  client --key dev routed-cmd --cmd ls --force-device <device-id>
  client --key dev routed-cmd --cmd pwd --require-npu

  # Execute a command locally (legacy mode)
  client --key dev --cmd pwd
`)
}

func main() {
	// Global flags
	addr := flag.String("addr", "localhost:50051", "Server address")
	key := flag.String("key", "", "Security key")
	device := flag.String("device", "", "Device name (defaults to hostname)")
	cmd := flag.String("cmd", "", "Command to execute (legacy mode)")
	var args arrayFlags
	flag.Var(&args, "arg", "Command arguments (repeatable)")

	flag.Usage = usage
	flag.Parse()

	// Check for subcommand
	subcommand := flag.Arg(0)

	// Connect to server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, *addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewOrchestratorServiceClient(conn)

	// Route to appropriate handler
	switch subcommand {
	case "register":
		handleRegister(ctx, client, flag.Args()[1:])
	case "list-devices":
		handleListDevices(ctx, client)
	case "status":
		handleStatus(ctx, client, flag.Args()[1:])
	case "route-task":
		handleRouteTask(ctx, client, *key, flag.Args()[1:])
	case "routed-cmd":
		handleRoutedCmd(ctx, client, *key, flag.Args()[1:])
	case "":
		// Legacy mode: execute command
		if *cmd == "" {
			fmt.Fprintln(os.Stderr, "Error: either --cmd or a subcommand is required")
			flag.Usage()
			os.Exit(1)
		}
		handleExecuteCommand(ctx, client, *key, *device, *cmd, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		flag.Usage()
		os.Exit(1)
	}
}

func handleRegister(ctx context.Context, client pb.OrchestratorServiceClient, args []string) {
	// Parse register-specific flags
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	name := fs.String("name", "", "Device name (required)")
	selfAddr := fs.String("self-addr", "", "This device's gRPC address (required)")
	deviceID := fs.String("id", "", "Device ID (auto-generated if not provided)")
	platform := fs.String("platform", "", "Platform (auto-detected if not provided)")
	arch := fs.String("arch", "", "Architecture (auto-detected if not provided)")
	hasGPU := fs.Bool("gpu", false, "Device has GPU")
	hasNPU := fs.Bool("npu", false, "Device has NPU")
	fs.Parse(args)

	if *name == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required")
		os.Exit(1)
	}
	if *selfAddr == "" {
		fmt.Fprintln(os.Stderr, "Error: --self-addr is required")
		os.Exit(1)
	}

	// Determine device ID
	var devID string
	if *deviceID != "" {
		// Use provided ID
		devID = *deviceID
	} else if isLocalAddress(*selfAddr) {
		// Local device: use persistent ID
		var err error
		devID, err = deviceid.GetOrCreate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting device ID: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Remote device: generate new UUID
		devID = generateUUID()
		fmt.Printf("Generated new device ID for remote device: %s\n", devID)
	}

	// Determine platform and arch
	plat := *platform
	if plat == "" {
		plat = runtime.GOOS
	}
	archStr := *arch
	if archStr == "" {
		archStr = runtime.GOARCH
	}

	// Build device info
	info := &pb.DeviceInfo{
		DeviceId:   devID,
		DeviceName: *name,
		Platform:   plat,
		Arch:       archStr,
		HasCpu:     true,
		HasGpu:     *hasGPU,
		HasNpu:     *hasNPU,
		GrpcAddr:   *selfAddr,
	}

	// Register device
	resp, err := client.RegisterDevice(ctx, info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error registering device: %v\n", err)
		os.Exit(1)
	}

	if resp.Ok {
		fmt.Printf("Device registered successfully!\n")
		fmt.Printf("  Device ID: %s\n", devID)
		fmt.Printf("  Name: %s\n", *name)
		fmt.Printf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  Address: %s\n", *selfAddr)
		fmt.Printf("  Registered at: %s\n", time.Unix(resp.RegisteredAt, 0).Format(time.RFC3339))
	} else {
		fmt.Fprintln(os.Stderr, "Registration failed")
		os.Exit(1)
	}
}

func handleListDevices(ctx context.Context, client pb.OrchestratorServiceClient) {
	resp, err := client.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing devices: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Devices) == 0 {
		fmt.Println("No devices registered")
		return
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEVICE ID\tNAME\tPLATFORM\tARCH\tCAPABILITIES\tADDRESS")
	fmt.Fprintln(w, "---------\t----\t--------\t----\t------------\t-------")

	for _, d := range resp.Devices {
		caps := "cpu"
		if d.HasGpu {
			caps += ",gpu"
		}
		if d.HasNpu {
			caps += ",npu"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncateID(d.DeviceId), d.DeviceName, d.Platform, d.Arch, caps, d.GrpcAddr)
	}
	w.Flush()
}

func handleStatus(ctx context.Context, client pb.OrchestratorServiceClient, args []string) {
	// Parse status-specific flags
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	id := fs.String("id", "", "Device ID (required)")
	fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "Error: --id is required")
		os.Exit(1)
	}

	resp, err := client.GetDeviceStatus(ctx, &pb.DeviceId{DeviceId: *id})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting device status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Device Status:\n")
	fmt.Printf("  Device ID: %s\n", resp.DeviceId)
	if resp.LastSeen > 0 {
		fmt.Printf("  Last Seen: %s\n", time.Unix(resp.LastSeen, 0).Format(time.RFC3339))
	} else {
		fmt.Printf("  Last Seen: never\n")
	}
	if resp.CpuLoad >= 0 {
		fmt.Printf("  CPU Load: %.1f%%\n", resp.CpuLoad*100)
	} else {
		fmt.Printf("  CPU Load: unavailable\n")
	}
	fmt.Printf("  Memory: %d MB used / %d MB total\n", resp.MemUsedMb, resp.MemTotalMb)
}

func handleRouteTask(ctx context.Context, client pb.OrchestratorServiceClient, key string, args []string) {
	// Parse route-task specific flags
	fs := flag.NewFlagSet("route-task", flag.ExitOnError)
	task := fs.String("task", "", "Task type (required): summarize, transcribe, analyze_screen")
	input := fs.String("input", "", "Task input")
	fs.Parse(args)

	if *task == "" {
		fmt.Fprintln(os.Stderr, "Error: --task is required")
		os.Exit(1)
	}

	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: --key is required for route-task")
		os.Exit(1)
	}

	// Create session first
	hostname, _ := os.Hostname()
	sessionResp, err := client.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  hostname,
		SecurityKey: key,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	// Route task
	resp, err := client.RunAITask(ctx, &pb.AITaskRequest{
		SessionId: sessionResp.SessionId,
		Task:      *task,
		Input:     *input,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error routing task: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task Routing Decision:\n")
	fmt.Printf("  Selected Device ID: %s\n", resp.SelectedDeviceId)
	fmt.Printf("  Selected Device Address: %s\n", resp.SelectedDeviceAddr)
	fmt.Printf("  Would Use NPU: %v\n", resp.WouldUseNpu)
	fmt.Printf("  Result: %s\n", resp.Result)
}

func handleRoutedCmd(ctx context.Context, client pb.OrchestratorServiceClient, key string, args []string) {
	// Parse routed-cmd specific flags
	fs := flag.NewFlagSet("routed-cmd", flag.ExitOnError)
	cmd := fs.String("cmd", "", "Command to execute (required)")
	preferRemote := fs.Bool("prefer-remote", false, "Prefer remote device if available")
	requireNPU := fs.Bool("require-npu", false, "Require device with NPU")
	forceDevice := fs.String("force-device", "", "Force execution on specific device ID")
	var cmdArgs arrayFlags
	fs.Var(&cmdArgs, "arg", "Command arguments (repeatable)")
	fs.Parse(args)

	if *cmd == "" {
		fmt.Fprintln(os.Stderr, "Error: --cmd is required for routed-cmd")
		os.Exit(1)
	}

	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: --key is required for routed-cmd")
		os.Exit(1)
	}

	// Build routing policy
	policy := &pb.RoutingPolicy{
		Mode: pb.RoutingPolicy_BEST_AVAILABLE,
	}

	if *forceDevice != "" {
		policy.Mode = pb.RoutingPolicy_FORCE_DEVICE_ID
		policy.DeviceId = *forceDevice
	} else if *requireNPU {
		policy.Mode = pb.RoutingPolicy_REQUIRE_NPU
	} else if *preferRemote {
		policy.Mode = pb.RoutingPolicy_PREFER_REMOTE
	}

	// Create session first
	hostname, _ := os.Hostname()
	sessionResp, err := client.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  hostname,
		SecurityKey: key,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	// Execute routed command
	resp, err := client.ExecuteRoutedCommand(ctx, &pb.RoutedCommandRequest{
		SessionId: sessionResp.SessionId,
		Policy:    policy,
		Command:   *cmd,
		Args:      cmdArgs,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing routed command: %v\n", err)
		os.Exit(1)
	}

	// Print routing info
	fmt.Printf("Routed Execution:\n")
	fmt.Printf("  Selected Device: %s (%s)\n", resp.SelectedDeviceName, truncateID(resp.SelectedDeviceId))
	fmt.Printf("  Device Address: %s\n", resp.SelectedDeviceAddr)
	fmt.Printf("  Executed Locally: %v\n", resp.ExecutedLocally)
	fmt.Printf("  Total Time: %.2f ms\n", resp.TotalTimeMs)
	fmt.Printf("  Exit Code: %d\n", resp.Output.ExitCode)
	fmt.Println("---")

	// Print command output
	if resp.Output.Stdout != "" {
		fmt.Print(resp.Output.Stdout)
	}
	if resp.Output.Stderr != "" {
		fmt.Fprint(os.Stderr, resp.Output.Stderr)
	}

	// Exit with command's exit code
	if resp.Output.ExitCode != 0 {
		os.Exit(int(resp.Output.ExitCode))
	}
}

func handleExecuteCommand(ctx context.Context, client pb.OrchestratorServiceClient, key, device, cmd string, args []string) {
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: --key is required")
		os.Exit(1)
	}

	// Default device name to hostname
	deviceName := device
	if deviceName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			deviceName = "unknown"
		} else {
			deviceName = hostname
		}
	}

	// Create session
	sessionResp, err := client.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  deviceName,
		SecurityKey: key,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	// Execute command
	cmdResp, err := client.ExecuteCommand(ctx, &pb.CommandRequest{
		SessionId: sessionResp.SessionId,
		Command:   cmd,
		Args:      args,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}

	// Print output
	if cmdResp.Stdout != "" {
		fmt.Print(cmdResp.Stdout)
	}
	if cmdResp.Stderr != "" {
		fmt.Fprint(os.Stderr, cmdResp.Stderr)
	}

	// Exit with command's exit code
	if cmdResp.ExitCode != 0 {
		os.Exit(int(cmdResp.ExitCode))
	}
}

// truncateID shortens a UUID for display
func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8] + "..."
	}
	return id
}

// isLocalAddress checks if an address refers to the local machine
func isLocalAddress(addr string) bool {
	// Extract host from addr (host:port format)
	host := addr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		host = addr[:idx]
	}

	// Check for common local addresses
	localHosts := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
	}

	for _, local := range localHosts {
		if host == local {
			return true
		}
	}

	return false
}

// generateUUID generates a new UUID v4
func generateUUID() string {
	// Simple UUID v4 generation
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("dev-%d", time.Now().UnixNano())
	}
	// Set version (4) and variant (2)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
