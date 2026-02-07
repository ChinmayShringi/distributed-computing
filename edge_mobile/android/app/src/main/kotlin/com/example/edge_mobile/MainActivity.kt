package com.example.edge_mobile

import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val CHANNEL = "com.example.edge_mobile/grpc"
    private lateinit var grpcClient: OrchestratorGrpcClient
    private lateinit var methodChannel: MethodChannel

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // Initialize gRPC client with default host (10.0.2.2 for emulator)
        grpcClient = OrchestratorGrpcClient()

        // Set up Method Channel
        methodChannel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL)
        
        // Set the handler
        val handler = GrpcMethodChannelHandler(grpcClient)
        methodChannel.setMethodCallHandler(handler)
    }

    override fun onDestroy() {
        super.onDestroy()
        grpcClient.close()
    }
}
