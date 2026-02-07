package com.example.edge_mobile

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.media.projection.MediaProjectionManager
import android.os.Build
import android.util.Log
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity(), DiscoveryManager.DiscoveryListener {
    private val TAG = "MainActivity"
    private val CHANNEL = "com.example.edge_mobile/grpc"
    private val SCREEN_CAPTURE_REQUEST = 1001

    private var grpcClient: OrchestratorGrpcClient? = null
    private var assistantClient: AssistantClient? = null
    private lateinit var methodChannel: MethodChannel
    private var pendingScreenCaptureResult: MethodChannel.Result? = null

    // Track if we've initialized clients
    private var clientsInitialized = false

    private fun isEmulator(): Boolean {
        return (Build.FINGERPRINT.startsWith("generic")
            || Build.FINGERPRINT.startsWith("unknown")
            || Build.MODEL.contains("google_sdk")
            || Build.MODEL.contains("Emulator")
            || Build.MODEL.contains("Android SDK built for x86")
            || Build.MANUFACTURER.contains("Genymotion")
            || (Build.BRAND.startsWith("generic") && Build.DEVICE.startsWith("generic"))
            || "google_sdk" == Build.PRODUCT)
    }

    /**
     * Get fallback host for emulator or when discovery fails
     */
    private fun getFallbackHost(): String {
        return if (isEmulator()) {
            "10.0.2.2"  // Emulator alias for host machine
        } else {
            "192.168.1.195"  // Fallback LAN IP
        }
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // Set up Method Channel first
        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL)

        // Start discovery to find orchestrator servers
        DiscoveryManager.addListener(this)
        DiscoveryManager.start(this, broadcastSelf = false)

        Log.i(TAG, "Started discovery, waiting for servers...")

        // Check if we already have a discovered server
        val activeServer = DiscoveryManager.getActiveServer()
        if (activeServer != null) {
            initializeClients(activeServer.grpcHost, activeServer.grpcPort, activeServer.httpPort)
        } else {
            // Use fallback after a short delay if no server discovered
            android.os.Handler(mainLooper).postDelayed({
                if (!clientsInitialized) {
                    Log.w(TAG, "No server discovered, using fallback host")
                    val fallback = getFallbackHost()
                    DiscoveryManager.setActiveServerManual(fallback, 50051, 8080)
                    initializeClients(fallback, 50051, 8080)
                }
            }, 3000) // Wait 3 seconds for discovery
        }

        // Set up discovery method channel handlers
        methodChannel.setMethodCallHandler { call, result ->
            when (call.method) {
                "getDiscoveredServers" -> handleGetDiscoveredServers(result)
                "setActiveServer" -> handleSetActiveServer(call.argument("device_id"), result)
                "getConnectionStatus" -> handleGetConnectionStatus(result)
                else -> {
                    // Delegate to GrpcMethodChannelHandler if clients are initialized
                    val grpc = grpcClient
                    val assistant = assistantClient
                    if (grpc != null && assistant != null) {
                        val handler = GrpcMethodChannelHandler(grpc, assistant, this)
                        handler.onMethodCall(call, result)
                    } else {
                        result.error("NOT_CONNECTED", "No server connection available", null)
                    }
                }
            }
        }
    }

    /**
     * Initialize gRPC and Assistant clients with discovered server
     */
    private fun initializeClients(host: String, grpcPort: Int, httpPort: Int) {
        Log.i(TAG, "Initializing clients: $host:$grpcPort (gRPC), $host:$httpPort (HTTP)")

        // Close existing clients if any
        grpcClient?.close()

        // Initialize gRPC client for orchestrator
        grpcClient = OrchestratorGrpcClient(host = host, port = grpcPort)

        // Initialize REST client for assistant
        assistantClient = AssistantClient(host = host, port = httpPort)

        clientsInitialized = true

        // Notify Flutter about connection
        methodChannel.invokeMethod("onConnectionChanged", mapOf(
            "connected" to true,
            "host" to host,
            "grpc_port" to grpcPort,
            "http_port" to httpPort
        ))
    }

    // ========== Discovery Listener ==========

    override fun onServerDiscovered(server: DiscoveryManager.ServerInfo) {
        Log.i(TAG, "Server discovered: ${server.deviceName} at ${server.grpcHost}:${server.grpcPort}")

        // Notify Flutter
        runOnUiThread {
            methodChannel.invokeMethod("onServerDiscovered", mapOf(
                "device_id" to server.deviceId,
                "device_name" to server.deviceName,
                "grpc_host" to server.grpcHost,
                "grpc_port" to server.grpcPort,
                "http_host" to server.httpHost,
                "http_port" to server.httpPort,
                "platform" to server.platform,
                "has_local_model" to server.hasLocalModel
            ))
        }
    }

    override fun onServerLost(deviceId: String) {
        Log.i(TAG, "Server lost: $deviceId")

        runOnUiThread {
            methodChannel.invokeMethod("onServerLost", mapOf(
                "device_id" to deviceId
            ))
        }
    }

    override fun onActiveServerChanged(server: DiscoveryManager.ServerInfo?) {
        if (server != null) {
            Log.i(TAG, "Active server changed to: ${server.deviceName}")
            initializeClients(server.grpcHost, server.grpcPort, server.httpPort)
        } else {
            Log.w(TAG, "No active server available")
            clientsInitialized = false

            runOnUiThread {
                methodChannel.invokeMethod("onConnectionChanged", mapOf(
                    "connected" to false
                ))
            }
        }
    }

    // ========== Method Channel Handlers ==========

    private fun handleGetDiscoveredServers(result: MethodChannel.Result) {
        val servers = DiscoveryManager.getDiscoveredServers().map { server ->
            mapOf(
                "device_id" to server.deviceId,
                "device_name" to server.deviceName,
                "grpc_host" to server.grpcHost,
                "grpc_port" to server.grpcPort,
                "http_host" to server.httpHost,
                "http_port" to server.httpPort,
                "platform" to server.platform,
                "has_local_model" to server.hasLocalModel,
                "is_active" to (server.deviceId == DiscoveryManager.getActiveServer()?.deviceId)
            )
        }
        result.success(servers)
    }

    private fun handleSetActiveServer(deviceId: String?, result: MethodChannel.Result) {
        if (deviceId == null) {
            result.error("INVALID_ARGUMENT", "device_id is required", null)
            return
        }

        val success = DiscoveryManager.setActiveServer(deviceId)
        if (success) {
            result.success(true)
        } else {
            result.error("NOT_FOUND", "Server with id $deviceId not found", null)
        }
    }

    private fun handleGetConnectionStatus(result: MethodChannel.Result) {
        val server = DiscoveryManager.getActiveServer()
        result.success(mapOf(
            "connected" to (server != null && clientsInitialized),
            "host" to (server?.grpcHost ?: ""),
            "grpc_port" to (server?.grpcPort ?: 50051),
            "http_port" to (server?.httpPort ?: 8080),
            "device_name" to (server?.deviceName ?: ""),
            "discovered_count" to DiscoveryManager.getDiscoveredServers().size
        ))
    }

    /**
     * Request screen capture permission from user
     */
    fun requestScreenCapture(result: MethodChannel.Result) {
        pendingScreenCaptureResult = result
        
        val projectionManager = getSystemService(Context.MEDIA_PROJECTION_SERVICE) as MediaProjectionManager
        val captureIntent = projectionManager.createScreenCaptureIntent()
        startActivityForResult(captureIntent, SCREEN_CAPTURE_REQUEST)
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        
        if (requestCode == SCREEN_CAPTURE_REQUEST) {
            if (resultCode == Activity.RESULT_OK && data != null) {
                // Permission granted - store for future WebRTC implementation
                pendingScreenCaptureResult?.success(true)
            } else {
                // Permission denied
                pendingScreenCaptureResult?.success(false)
            }
            pendingScreenCaptureResult = null
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        DiscoveryManager.removeListener(this)
        DiscoveryManager.stop()
        grpcClient?.close()
    }
}
