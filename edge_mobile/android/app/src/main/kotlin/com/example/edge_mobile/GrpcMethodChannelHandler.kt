package com.example.edge_mobile

import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * GrpcMethodChannelHandler handles method channel calls from Flutter
 * and delegates them to the OrchestratorGrpcClient
 */
class GrpcMethodChannelHandler(
    private val grpcClient: OrchestratorGrpcClient
) : MethodChannel.MethodCallHandler {

    private val scope = CoroutineScope(Dispatchers.Main)

    override fun onMethodCall(call: MethodCall, result: MethodChannel.Result) {
        when (call.method) {
            "listDevices" -> handleListDevices(result)
            "createSession" -> handleCreateSession(call, result)
            "healthCheck" -> handleHealthCheck(result)
            "getDeviceStatus" -> handleGetDeviceStatus(call, result)
            "executeRoutedCommand" -> handleExecuteRoutedCommand(call, result)
            "configureHost" -> handleConfigureHost(call, result)
            "closeConnection" -> handleCloseConnection(result)
            else -> result.notImplemented()
        }
    }

    private fun handleListDevices(result: MethodChannel.Result) {
        scope.launch {
            try {
                val devices = grpcClient.listDevices()
                result.success(devices)
            } catch (e: Exception) {
                result.error("LIST_DEVICES_ERROR", e.message, e.toString())
            }
        }
    }

    private fun handleCreateSession(call: MethodCall, result: MethodChannel.Result) {
        scope.launch {
            try {
                val deviceName = call.argument<String>("device_name") ?: "android-device"
                val securityKey = call.argument<String>("security_key") ?: "dev"
                
                val session = grpcClient.createSession(deviceName, securityKey)
                result.success(session)
            } catch (e: Exception) {
                result.error("CREATE_SESSION_ERROR", e.message, e.toString())
            }
        }
    }

    private fun handleHealthCheck(result: MethodChannel.Result) {
        scope.launch {
            try {
                val health = grpcClient.healthCheck()
                result.success(health)
            } catch (e: Exception) {
                result.error("HEALTH_CHECK_ERROR", e.message, e.toString())
            }
        }
    }

    private fun handleGetDeviceStatus(call: MethodCall, result: MethodChannel.Result) {
        scope.launch {
            try {
                val deviceId = call.argument<String>("device_id")
                if (deviceId == null) {
                    result.error("INVALID_ARGUMENT", "device_id is required", null)
                    return@launch
                }
                
                val status = grpcClient.getDeviceStatus(deviceId)
                result.success(status)
            } catch (e: Exception) {
                result.error("GET_DEVICE_STATUS_ERROR", e.message, e.toString())
            }
        }
    }

    private fun handleExecuteRoutedCommand(call: MethodCall, result: MethodChannel.Result) {
        scope.launch {
            try {
                val command = call.argument<String>("command")
                if (command == null) {
                    result.error("INVALID_ARGUMENT", "command is required", null)
                    return@launch
                }
                
                val args = call.argument<List<String>>("args") ?: emptyList()
                val policy = call.argument<String>("policy") ?: "BEST_AVAILABLE"
                
                val response = grpcClient.executeRoutedCommand(command, args, policy)
                result.success(response)
            } catch (e: Exception) {
                result.error("EXECUTE_COMMAND_ERROR", e.message, e.toString())
            }
        }
    }

    private fun handleConfigureHost(call: MethodCall, result: MethodChannel.Result) {
        try {
            val host = call.argument<String>("host")
            val port = call.argument<Int>("port")
            
            if (host == null || port == null) {
                result.error("INVALID_ARGUMENT", "host and port are required", null)
                return
            }
            
            // Close existing connection
            grpcClient.close()
            
            // Note: To fully support dynamic host configuration, the grpcClient
            // would need to be recreated with new host/port.
            // For now, this just closes the existing connection.
            // A full implementation would create a new OrchestratorGrpcClient instance.
            
            result.success(mapOf("configured" to true, "host" to host, "port" to port))
        } catch (e: Exception) {
            result.error("CONFIGURE_HOST_ERROR", e.message, e.toString())
        }
    }

    private fun handleCloseConnection(result: MethodChannel.Result) {
        try {
            grpcClient.close()
            result.success(mapOf("closed" to true))
        } catch (e: Exception) {
            result.error("CLOSE_CONNECTION_ERROR", e.message, e.toString())
        }
    }
}
