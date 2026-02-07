package com.example.edge_mobile

import android.content.Context
import android.util.Log
import com.example.edge_mobile.grpc.*
import io.grpc.stub.StreamObserver
import kotlinx.coroutines.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class OrchestratorServerService(private val context: Context) : OrchestratorServiceGrpc.OrchestratorServiceImplBase() {
    
    private val scope = CoroutineScope(Dispatchers.Default + SupervisorJob())
    private val TAG = "OrchestratorServerService"

    /**
     * RunTask - Execute a task and return the result
     */
    override fun runTask(request: TaskRequest, responseObserver: StreamObserver<TaskResult>) {
        Log.d(TAG, "RunTask: task_id=${request.taskId} job_id=${request.jobId} kind=${request.kind}")
        
        scope.launch {
            val startTime = System.currentTimeMillis()
            
            try {
                val output = TaskExecutor.execute(context, request.kind, request.input)
                val endTime = System.currentTimeMillis()
                
                val result = TaskResult.newBuilder()
                    .setTaskId(request.taskId)
                    .setOk(true)
                    .setOutput(output)
                    .setTimeMs((endTime - startTime).toDouble())
                    .build()
                
                responseObserver.onNext(result)
                responseObserver.onCompleted()
                
                Log.d(TAG, "Task completed: ${request.taskId}")
                
            } catch (e: Exception) {
                val endTime = System.currentTimeMillis()
                Log.e(TAG, "Task failed: ${request.taskId}", e)
                
                val result = TaskResult.newBuilder()
                    .setTaskId(request.taskId)
                    .setOk(false)
                    .setError("Task execution failed: ${e.message}")
                    .setTimeMs((endTime - startTime).toDouble())
                    .build()
                
                responseObserver.onNext(result)
                responseObserver.onCompleted()
            }
        }
    }

    /**
     * HealthCheck - Verify server is alive
     */
    override fun healthCheck(request: Empty, responseObserver: StreamObserver<HealthStatus>) {
        Log.d(TAG, "HealthCheck received")
        
        val deviceInfo = DeviceInfoCollector.getSelfDeviceInfo(context)
        
        val response = HealthStatus.newBuilder()
            .setDeviceId(deviceInfo.deviceId)
            .setServerTime(System.currentTimeMillis() / 1000)
            .setMessage("ok")
            .build()
        
        responseObserver.onNext(response)
        responseObserver.onCompleted()
    }

    /**
     * RegisterDevice - Register this device with the orchestrator
     * (Not typically called on the worker, but included for completeness)
     */
    override fun registerDevice(request: DeviceInfo, responseObserver: StreamObserver<DeviceAck>) {
        Log.d(TAG, "RegisterDevice: ${request.deviceName}")
        
        val response = DeviceAck.newBuilder()
            .setOk(true)
            .setRegisteredAt(System.currentTimeMillis() / 1000)
            .build()
        
        responseObserver.onNext(response)
        responseObserver.onCompleted()
    }

    /**
     * GetDeviceStatus - Return current device status
     */
    override fun getDeviceStatus(request: DeviceId, responseObserver: StreamObserver<DeviceStatus>) {
        Log.d(TAG, "GetDeviceStatus: ${request.deviceId}")
        
        val deviceInfo = DeviceInfoCollector.getSelfDeviceInfo(context)
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as? android.app.ActivityManager
        val memInfo = android.app.ActivityManager.MemoryInfo()
        activityManager?.getMemoryInfo(memInfo)
        
        val totalRamMb = (memInfo.totalMem / (1024 * 1024)).toULong()
        val freeRamMb = (memInfo.availMem / (1024 * 1024)).toULong()
        val usedRamMb = totalRamMb - freeRamMb
        
        val response = DeviceStatus.newBuilder()
            .setDeviceId(deviceInfo.deviceId)
            .setLastSeen(System.currentTimeMillis() / 1000)
            .setCpuLoad(-1.0)  // CPU load not easily available on Android
            .setMemUsedMb(usedRamMb.toLong())
            .setMemTotalMb(totalRamMb.toLong())
            .build()
        
        responseObserver.onNext(response)
        responseObserver.onCompleted()
    }

    /**
     * RunLLMTask - Execute LLM inference using local Ollama on this device
     */
    override fun runLLMTask(request: LLMTaskRequest, responseObserver: StreamObserver<LLMTaskResponse>) {
        Log.d(TAG, "RunLLMTask: prompt_len=${request.prompt.length}")
        
        scope.launch {
            val deviceInfo = DeviceInfoCollector.getSelfDeviceInfo(context)
            if (!deviceInfo.hasLocalModel || deviceInfo.localChatEndpoint.isEmpty()) {
                responseObserver.onNext(LLMTaskResponse.newBuilder()
                    .setError("No local LLM model available on this device")
                    .build())
                responseObserver.onCompleted()
                return@launch
            }
            
            val model = if (request.model.isNotEmpty()) request.model else deviceInfo.localModelName
            if (model.isEmpty()) {
                responseObserver.onNext(LLMTaskResponse.newBuilder()
                    .setError("No model specified and device has no default model")
                    .build())
                responseObserver.onCompleted()
                return@launch
            }
            
            try {
                // Ollama runs on device - use 127.0.0.1 from app's perspective
                val baseUrl = "http://127.0.0.1:11434"
                val output = callOllamaChat(baseUrl, model, request.prompt)
                
                responseObserver.onNext(LLMTaskResponse.newBuilder()
                    .setOutput(output)
                    .setModelUsed(model)
                    .setTokensGenerated((output.length / 4).toLong())  // rough estimate
                    .build())
                responseObserver.onCompleted()
                Log.d(TAG, "RunLLMTask completed: output_len=${output.length}")
            } catch (e: Exception) {
                Log.e(TAG, "RunLLMTask failed", e)
                responseObserver.onNext(LLMTaskResponse.newBuilder()
                    .setError("LLM inference failed: ${e.message}")
                    .build())
                responseObserver.onCompleted()
            }
        }
    }
    
    private fun callOllamaChat(baseUrl: String, model: String, prompt: String): String {
        val client = OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(120, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
        
        val json = JSONObject().apply {
            put("model", model)
            put("stream", false)
            put("messages", org.json.JSONArray().apply {
                put(JSONObject().apply {
                    put("role", "user")
                    put("content", prompt)
                })
            })
        }
        
        val url = baseUrl.trimEnd('/') + "/api/chat"
        val request = Request.Builder()
            .url(url)
            .post(json.toString().toRequestBody("application/json; charset=utf-8".toMediaType()))
            .build()
        
        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                val body = response.body?.string() ?: ""
                throw Exception("Ollama returned ${response.code}: $body")
            }
            val body = response.body?.string() ?: throw Exception("Empty response")
            val obj = JSONObject(body)
            val message = obj.optJSONObject("message")
            return message?.optString("content", "") ?: ""
        }
    }

    /**
     * StartWebRTC - WebRTC currently requires native implementation
     */
    override fun startWebRTC(request: WebRTCConfig, responseObserver: StreamObserver<WebRTCOffer>) {
        Log.d(TAG, "StartWebRTC: session=${request.sessionId}")
        
        // Android WebRTC streaming is complex and requires native video capture.
        // For now, return a helpful error message
        responseObserver.onError(
            io.grpc.Status.UNIMPLEMENTED
                .withDescription("WebRTC streaming from Android requires native WebRTC library integration. " +
                    "Consider using screenshot-based streaming or implement flutter_webrtc integration.")
                .asException()
        )
    }

    /**
     * CompleteWebRTC - Set answer SDP to complete WebRTC handshake
     */
    override fun completeWebRTC(request: WebRTCAnswer, responseObserver: StreamObserver<Empty>) {
        Log.d(TAG, "CompleteWebRTC: stream=${request.streamId}")
        
        responseObserver.onError(
            io.grpc.Status.UNIMPLEMENTED
                .withDescription("WebRTC streaming not implemented on Android")
                .asException()
        )
    }

    /**
     * StopWebRTC - Stop WebRTC stream and cleanup
     */
    override fun stopWebRTC(request: WebRTCStop, responseObserver: StreamObserver<Empty>) {
        Log.d(TAG, "StopWebRTC: stream=${request.streamId}")
        
        responseObserver.onError(
            io.grpc.Status.UNIMPLEMENTED
                .withDescription("WebRTC streaming not implemented on Android")
                .asException()
        )
    }

    /**
     * Clean up coroutines when service stops
     */
    fun shutdown() {
        scope.cancel()
    }
}
