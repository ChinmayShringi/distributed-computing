package com.example.edge_mobile

import android.content.Intent
import android.os.Build
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * GrpcMethodChannelHandler handles method channel calls from Flutter
 * and delegates them to the OrchestratorGrpcClient and AssistantClient
 */
class GrpcMethodChannelHandler(
    private val grpcClient: OrchestratorGrpcClient,
    private val assistantClient: AssistantClient,
    private val mainActivity: MainActivity
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
            "startWorker" -> handleStartWorker(result)
            "stopWorker" -> handleStopWorker(result)
            "isWorkerRunning" -> handleIsWorkerRunning(result)
            "getJob" -> handleGetJob(call, result)
            "requestScreenCapture" -> handleRequestScreenCapture(result)
            "sendAssistantMessage" -> handleSendAssistantMessage(call, result)
            "getActivity" -> handleGetActivity(call, result)
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

    private fun handleStartWorker(result: MethodChannel.Result) {
        try {
            val intent = Intent(mainActivity, WorkerService::class.java)
            
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                mainActivity.startForegroundService(intent)
            } else {
                mainActivity.startService(intent)
            }
            
            result.success(true)
        } catch (e: Exception) {
            result.error("START_WORKER_ERROR", e.message, e.toString())
        }
    }

    private fun handleStopWorker(result: MethodChannel.Result) {
        try {
            val intent = Intent(mainActivity, WorkerService::class.java)
            mainActivity.stopService(intent)
            result.success(true)
        } catch (e: Exception) {
            result.error("STOP_WORKER_ERROR", e.message, e.toString())
        }
    }

    private fun handleIsWorkerRunning(result: MethodChannel.Result) {
        try {
            // Check if WorkerService is running
            val manager = mainActivity.getSystemService(android.app.ActivityManager::class.java)
            val isRunning = manager.getRunningServices(Integer.MAX_VALUE)
                .any { it.service.className == WorkerService::class.java.name }
            
            result.success(isRunning)
        } catch (e: Exception) {
            result.error("IS_WORKER_RUNNING_ERROR", e.message, e.toString())
        }
    }

    private fun handleGetJob(call: MethodCall, result: MethodChannel.Result) {
        scope.launch {
            try {
                val jobId = call.argument<String>("job_id")
                if (jobId == null) {
                    result.error("INVALID_ARGUMENT", "job_id is required", null)
                    return@launch
                }
                
                val jobStatus = grpcClient.getJob(jobId)
                result.success(jobStatus)
            } catch (e: Exception) {
                result.error("GET_JOB_ERROR", e.message, e.toString())
            }
        }
    }

    private fun handleRequestScreenCapture(result: MethodChannel.Result) {
        try {
            // Request screen capture permission from MainActivity
            mainActivity.requestScreenCapture(result)
        } catch (e: Exception) {
            result.error("SCREEN_CAPTURE_ERROR", e.message, e.toString())
        }
    }

    /**
     * Handle sending a message to the assistant via REST API
     */
    private fun handleSendAssistantMessage(call: MethodCall, result: MethodChannel.Result) {
        scope.launch {
            try {
                val text = call.argument<String>("text")
                if (text == null) {
                    result.error("INVALID_ARGUMENT", "text is required", null)
                    return@launch
                }

                val response = assistantClient.sendMessage(text)
                result.success(response)
            } catch (e: Exception) {
                result.error("ASSISTANT_ERROR", e.message, e.toString())
            }
        }
    }

    /**
     * Handle getting activity data (running tasks, device activities) via gRPC
     */
    private fun handleGetActivity(call: MethodCall, result: MethodChannel.Result) {
        scope.launch {
            try {
                val includeMetrics = call.argument<Boolean>("include_metrics") ?: false
                val metricsSinceMs = call.argument<Long>("metrics_since_ms") ?: 0L

                val activity = grpcClient.getActivity(includeMetrics, metricsSinceMs)
                result.success(activity)
            } catch (e: Exception) {
                result.error("GET_ACTIVITY_ERROR", e.message, e.toString())
            }
        }
    }
}
