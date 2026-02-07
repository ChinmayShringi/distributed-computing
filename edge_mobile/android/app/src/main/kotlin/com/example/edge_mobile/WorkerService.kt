package com.example.edge_mobile

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import io.grpc.Server
import io.grpc.ServerBuilder
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

class WorkerService : Service() {
    private val TAG = "WorkerService"
    private val CHANNEL_ID = "edge_worker_channel"
    private val NOTIFICATION_ID = 1001
    
    private var grpcServer: Server? = null
    private var discoveryService: DiscoveryService? = null
    private var serverService: OrchestratorServerService? = null
    private var grpcClient: OrchestratorGrpcClient? = null
    
    private val scope = CoroutineScope(Dispatchers.Default + SupervisorJob())

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
        Log.i(TAG, "WorkerService created")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Log.i(TAG, "WorkerService starting...")
        
        // Start as foreground service
        val notification = buildNotification("Starting worker service...")
        startForeground(NOTIFICATION_ID, notification)
        
        // Start gRPC server and discovery
        scope.launch {
            try {
                startGrpcServer()
                startDiscovery()
                updateNotification("Worker active - Ready to execute tasks")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to start services", e)
                updateNotification("Worker error: ${e.message}")
                stopSelf()
            }
        }
        
        return START_STICKY
    }

    private suspend fun startGrpcServer() {
        serverService = OrchestratorServerService(applicationContext)
        
        // Use Netty server builder for Android
        grpcServer = io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder
            .forPort(50051)
            .addService(serverService!!)
            .build()
            .start()
        
        Log.i(TAG, "gRPC server started on port 50051")
    }

    private suspend fun startDiscovery() {
        val selfDevice = DeviceInfoCollector.getSelfDeviceInfo(applicationContext)
        
        // Initialize gRPC client to register with orchestrator
        val ipAddress = getOrchestratorIp()
        grpcClient = OrchestratorGrpcClient(host = ipAddress, port = 50051)
        
        discoveryService = DiscoveryService(
            port = 50051,  // Use same port as backend for discovery
            selfDeviceInfo = selfDevice,
            onDeviceDiscovered = { device ->
                // When we discover a device, try to register ourselves with it
                scope.launch {
                    registerWithDevice(device)
                }
            }
        )
        
        discoveryService?.start()
        Log.i(TAG, "Discovery service started")
        
        // Also register directly with known orchestrator
        registerSelf()
    }

    /**
     * Get orchestrator IP (Mac coordinator)
     * Default to gateway IP or user-configured value
     */
    private fun getOrchestratorIp(): String {
        // For now, use the same IP as the gRPC client
        // This should be the Mac's LAN IP
        return "192.168.1.195"
    }

    /**
     * Register this device with a discovered peer
     */
    private suspend fun registerWithDevice(device: DiscoveryService.DeviceAnnounce) {
        try {
            // Extract host from grpc_addr (format: "ip:port")
            val parts = device.grpcAddr.split(":")
            if (parts.size != 2) return
            
            val host = parts[0]
            val port = parts[1].toIntOrNull() ?: 50051
            
            // Create temporary client to register
            val client = OrchestratorGrpcClient(host = host, port = port)
            
            // Build DeviceInfo for registration
            val selfDevice = DeviceInfoCollector.getSelfDeviceInfo(applicationContext)
            
            // Note: We would need to add a registerDevice method to OrchestratorGrpcClient
            // For now, log the discovery
            Log.i(TAG, "Discovered peer: ${device.deviceName} at ${device.grpcAddr}")
            
            client.close()
            
        } catch (e: Exception) {
            Log.d(TAG, "Failed to register with ${device.deviceName}: ${e.message}")
        }
    }

    /**
     * Register self with the known orchestrator
     */
    private fun registerSelf() {
        scope.launch {
            try {
                val selfDevice = DeviceInfoCollector.getSelfDeviceInfo(applicationContext)
                Log.i(TAG, "Registering self: ${selfDevice.deviceName} at ${selfDevice.grpcAddr}")
                
                // The orchestrator will discover us via UDP broadcast
                // No explicit registration needed if discovery is working
                
            } catch (e: Exception) {
                Log.e(TAG, "Failed to register self", e)
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        Log.i(TAG, "WorkerService stopping...")
        
        discoveryService?.stop()
        grpcServer?.shutdown()
        serverService?.shutdown()
        grpcClient?.close()
        scope.cancel()
        
        try {
            grpcServer?.awaitTermination(5, java.util.concurrent.TimeUnit.SECONDS)
        } catch (e: InterruptedException) {
            grpcServer?.shutdownNow()
        }
        
        Log.i(TAG, "WorkerService stopped")
    }

    override fun onBind(intent: Intent?): IBinder? {
        return null
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Edge Worker Service",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Keeps the edge worker service running"
                setShowBadge(false)
            }
            
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }
    }

    private fun buildNotification(message: String): Notification {
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Edge Worker")
            .setContentText(message)
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setOngoing(true)
            .build()
    }

    private fun updateNotification(message: String) {
        val notification = buildNotification(message)
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.notify(NOTIFICATION_ID, notification)
    }
}
