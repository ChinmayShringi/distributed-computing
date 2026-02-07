package com.example.edge_mobile

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.media.projection.MediaProjectionManager
import android.os.Build
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val CHANNEL = "com.example.edge_mobile/grpc"
    private val SCREEN_CAPTURE_REQUEST = 1001
    
    private lateinit var grpcClient: OrchestratorGrpcClient
    private lateinit var assistantClient: AssistantClient
    private lateinit var methodChannel: MethodChannel
    private var pendingScreenCaptureResult: MethodChannel.Result? = null

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

    private fun getGrpcHost(): String {
        return if (isEmulator()) {
            "10.0.2.2"  // Emulator alias for host machine
        } else {
            "192.168.1.195"  // LAN IP for physical device (host machine)
        }
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        val host = getGrpcHost()

        // Initialize gRPC client for orchestrator (port 50051)
        grpcClient = OrchestratorGrpcClient(host = host, port = 50051)

        // Initialize REST client for assistant (port 8080)
        assistantClient = AssistantClient(host = host, port = 8080)

        // Set up Method Channel
        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL)

        // Set the handler with both clients
        val handler = GrpcMethodChannelHandler(grpcClient, assistantClient, this)
        methodChannel.setMethodCallHandler(handler)
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
        grpcClient.close()
    }
}
