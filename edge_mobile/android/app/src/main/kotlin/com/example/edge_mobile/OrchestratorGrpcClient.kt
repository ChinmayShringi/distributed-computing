package com.example.edge_mobile

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Stub OrchestratorGrpcClient — allows the app to build and run without gRPC/protobuf.
 * Returns empty/safe data so the UI renders. Replace with real gRPC client when
 * proto codegen and dependencies are added.
 */
class OrchestratorGrpcClient(
    private val host: String = "10.0.2.2",
    private val port: Int = 50051
) {
    private var sessionId: String? = null

    companion object {
        private const val DEFAULT_SECURITY_KEY = "dev"
        private const val DEFAULT_DEVICE_NAME = "android-device"
    }

    fun init() {
        // No-op for stub
    }

    suspend fun createSession(
        deviceName: String = DEFAULT_DEVICE_NAME,
        securityKey: String = DEFAULT_SECURITY_KEY
    ): Map<String, Any> = withContext(Dispatchers.IO) {
        sessionId = "stub-session"
        mapOf(
            "session_id" to sessionId!!,
            "host_name" to "stub",
            "connected_at" to System.currentTimeMillis().toString()
        )
    }

    suspend fun listDevices(): List<Map<String, Any>> = withContext(Dispatchers.IO) {
        emptyList()
    }

    suspend fun healthCheck(): Map<String, Any> = withContext(Dispatchers.IO) {
        mapOf(
            "device_id" to "stub",
            "server_time" to System.currentTimeMillis().toString(),
            "message" to "Stub client — configure gRPC for real server"
        )
    }

    suspend fun getDeviceStatus(deviceId: String): Map<String, Any> = withContext(Dispatchers.IO) {
        mapOf(
            "device_id" to deviceId,
            "last_seen" to "",
            "cpu_load" to 0.0,
            "mem_used_mb" to 0,
            "mem_total_mb" to 0
        )
    }

    suspend fun executeRoutedCommand(
        command: String,
        args: List<String> = emptyList(),
        policy: String = "BEST_AVAILABLE"
    ): Map<String, Any> = withContext(Dispatchers.IO) {
        if (sessionId == null) createSession()
        mapOf(
            "selected_device_id" to "",
            "selected_device_name" to "",
            "selected_device_addr" to "",
            "executed_locally" to false,
            "total_time_ms" to 0L,
            "exit_code" to -1,
            "stdout" to "Stub client — configure gRPC for real server",
            "stderr" to ""
        )
    }

    fun close() {
        sessionId = null
    }
}
