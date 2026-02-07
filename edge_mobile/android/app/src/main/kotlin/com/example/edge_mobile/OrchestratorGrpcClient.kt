package com.example.edge_mobile

import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import com.example.edge_mobile.grpc.*
import java.util.concurrent.TimeUnit

/**
 * OrchestratorGrpcClient manages the connection to the EdgeCLI gRPC server
 * and provides methods to interact with the orchestrator service.
 */
class OrchestratorGrpcClient(
    private val host: String = "10.0.2.2",
    private val port: Int = 50051
) {
    private var channel: ManagedChannel? = null
    private var stub: OrchestratorServiceGrpc.OrchestratorServiceBlockingStub? = null
    private var sessionId: String? = null

    companion object {
        private const val DEFAULT_SECURITY_KEY = "dev"
        private const val DEFAULT_DEVICE_NAME = "android-device"
        private const val CONNECTION_TIMEOUT_SECONDS = 10L
    }

    /**
     * Initialize the gRPC channel and stub
     */
    fun init() {
        if (channel == null) {
            channel = ManagedChannelBuilder
                .forAddress(host, port)
                .usePlaintext()
                .build()

            stub = OrchestratorServiceGrpc.newBlockingStub(channel)
        }
    }

    /**
     * Create a session with the orchestrator
     */
    suspend fun createSession(
        deviceName: String = DEFAULT_DEVICE_NAME,
        securityKey: String = DEFAULT_SECURITY_KEY
    ): Map<String, Any> = withContext(Dispatchers.IO) {
        init()

        try {
            val request = AuthRequest.newBuilder()
                .setDeviceName(deviceName)
                .setSecurityKey(securityKey)
                .build()

            val response = stub!!.createSession(request)
            sessionId = response.sessionId

            mapOf(
                "session_id" to response.sessionId,
                "host_name" to response.hostName,
                "connected_at" to response.connectedAt
            )
        } catch (e: Exception) {
            throw Exception("Failed to create session: ${e.message}", e)
        }
    }

    /**
     * List all registered devices
     */
    suspend fun listDevices(): List<Map<String, Any>> = withContext(Dispatchers.IO) {
        init()

        try {
            val request = ListDevicesRequest.newBuilder().build()
            val response = stub!!.listDevices(request)

            response.devicesList.map { device ->
                val capabilities = mutableListOf("cpu")
                if (device.hasGpu) capabilities.add("gpu")
                if (device.hasNpu) capabilities.add("npu")

                mapOf(
                    "device_id" to device.deviceId,
                    "device_name" to device.deviceName,
                    "platform" to device.platform,
                    "arch" to device.arch,
                    "capabilities" to capabilities,
                    "grpc_addr" to device.grpcAddr,
                    "can_screen_capture" to device.canScreenCapture,
                    "http_addr" to device.httpAddr
                )
            }
        } catch (e: Exception) {
            throw Exception("Failed to list devices: ${e.message}", e)
        }
    }

    /**
     * Health check to verify server connectivity
     */
    suspend fun healthCheck(): Map<String, Any> = withContext(Dispatchers.IO) {
        init()

        try {
            val request = Empty.newBuilder().build()
            val response = stub!!.healthCheck(request)

            mapOf(
                "device_id" to response.deviceId,
                "server_time" to response.serverTime,
                "message" to response.message
            )
        } catch (e: Exception) {
            throw Exception("Failed to health check: ${e.message}", e)
        }
    }

    /**
     * Get status of a specific device
     */
    suspend fun getDeviceStatus(deviceId: String): Map<String, Any> = withContext(Dispatchers.IO) {
        init()

        try {
            val request = DeviceId.newBuilder()
                .setDeviceId(deviceId)
                .build()

            val response = stub!!.getDeviceStatus(request)

            mapOf(
                "device_id" to response.deviceId,
                "last_seen" to response.lastSeen,
                "cpu_load" to response.cpuLoad,
                "mem_used_mb" to response.memUsedMb,
                "mem_total_mb" to response.memTotalMb
            )
        } catch (e: Exception) {
            throw Exception("Failed to get device status: ${e.message}", e)
        }
    }

    /**
     * Get job status by ID
     */
    suspend fun getJob(jobId: String): Map<String, Any> = withContext(Dispatchers.IO) {
        init()

        try {
            val request = JobId.newBuilder()
                .setJobId(jobId)
                .build()

            val response = stub!!.getJob(request)

            mapOf(
                "job_id" to response.jobId,
                "state" to response.state,
                "final_result" to response.finalResult,
                "current_group" to response.currentGroup,
                "total_groups" to response.totalGroups,
                "tasks" to response.tasksList.map { task ->
                    mapOf(
                        "task_id" to task.taskId,
                        "assigned_device_id" to task.assignedDeviceId,
                        "assigned_device_name" to task.assignedDeviceName,
                        "state" to task.state,
                        "result" to task.result,
                        "error" to task.error
                    )
                }
            )
        } catch (e: Exception) {
            throw Exception("Failed to get job: ${e.message}", e)
        }
    }

    /**
     * Execute a routed command (with automatic device selection)
     */
    suspend fun executeRoutedCommand(
        command: String,
        args: List<String> = emptyList(),
        policy: String = "BEST_AVAILABLE"
    ): Map<String, Any> = withContext(Dispatchers.IO) {
        init()

        if (sessionId == null) {
            createSession()
        }

        try {
            val policyMode = when (policy.uppercase()) {
                "REQUIRE_NPU" -> RoutingPolicy.Mode.REQUIRE_NPU
                "PREFER_REMOTE" -> RoutingPolicy.Mode.PREFER_REMOTE
                "FORCE_DEVICE_ID" -> RoutingPolicy.Mode.FORCE_DEVICE_ID
                else -> RoutingPolicy.Mode.BEST_AVAILABLE
            }

            val routingPolicy = RoutingPolicy.newBuilder()
                .setMode(policyMode)
                .build()

            val request = RoutedCommandRequest.newBuilder()
                .setSessionId(sessionId!!)
                .setPolicy(routingPolicy)
                .setCommand(command)
                .addAllArgs(args)
                .build()

            val response = stub!!.executeRoutedCommand(request)

            mapOf(
                "selected_device_id" to response.selectedDeviceId,
                "selected_device_name" to response.selectedDeviceName,
                "selected_device_addr" to response.selectedDeviceAddr,
                "executed_locally" to response.executedLocally,
                "total_time_ms" to response.totalTimeMs,
                "exit_code" to response.output.exitCode,
                "stdout" to response.output.stdout,
                "stderr" to response.output.stderr
            )
        } catch (e: Exception) {
            throw Exception("Failed to execute routed command: ${e.message}", e)
        }
    }

    /**
     * Get activity data (running tasks, device activities, optional metrics history)
     */
    suspend fun getActivity(
        includeMetrics: Boolean = false,
        metricsSinceMs: Long = 0
    ): Map<String, Any> = withContext(Dispatchers.IO) {
        init()

        try {
            val request = GetActivityRequest.newBuilder()
                .setIncludeMetricsHistory(includeMetrics)
                .setMetricsSinceMs(metricsSinceMs)
                .build()

            val response = stub!!.getActivity(request)
            val activity = response.activity

            val result = mutableMapOf<String, Any>(
                "running_tasks" to activity.runningTasksList.map { task ->
                    mapOf(
                        "task_id" to task.taskId,
                        "job_id" to task.jobId,
                        "kind" to task.kind,
                        "input" to task.input,
                        "device_id" to task.deviceId,
                        "device_name" to task.deviceName,
                        "started_at_ms" to task.startedAtMs,
                        "elapsed_ms" to task.elapsedMs
                    )
                },
                "device_activities" to activity.deviceActivitiesList.map { deviceActivity ->
                    mapOf(
                        "device_id" to deviceActivity.deviceId,
                        "device_name" to deviceActivity.deviceName,
                        "running_task_count" to deviceActivity.runningTaskCount,
                        "current_status" to if (deviceActivity.hasCurrentStatus()) {
                            mapOf(
                                "cpu_load" to deviceActivity.currentStatus.cpuLoad,
                                "mem_used_mb" to deviceActivity.currentStatus.memUsedMb,
                                "mem_total_mb" to deviceActivity.currentStatus.memTotalMb
                            )
                        } else null
                    )
                }
            )

            // Include metrics history if requested
            if (includeMetrics && response.deviceMetricsCount > 0) {
                result["device_metrics"] = response.deviceMetricsMap.mapValues { (_, metricsHistory) ->
                    mapOf(
                        "device_id" to metricsHistory.deviceId,
                        "device_name" to metricsHistory.deviceName,
                        "samples" to metricsHistory.samplesList.map { sample ->
                            mapOf(
                                "timestamp_ms" to sample.timestampMs,
                                "cpu_load" to sample.cpuLoad,
                                "mem_used_mb" to sample.memUsedMb,
                                "mem_total_mb" to sample.memTotalMb
                            )
                        }
                    )
                }
            }

            result
        } catch (e: Exception) {
            throw Exception("Failed to get activity: ${e.message}", e)
        }
    }

    /**
     * Close the gRPC channel and clean up resources
     */
    fun close() {
        channel?.shutdown()
        try {
            channel?.awaitTermination(CONNECTION_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        } catch (e: InterruptedException) {
            channel?.shutdownNow()
        }
        channel = null
        stub = null
        sessionId = null
    }
}
