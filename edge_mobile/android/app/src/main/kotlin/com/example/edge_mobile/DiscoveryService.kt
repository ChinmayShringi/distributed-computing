package com.example.edge_mobile

import android.util.Log
import com.example.edge_mobile.grpc.DeviceInfo
import kotlinx.coroutines.*
import org.json.JSONObject
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.util.concurrent.ConcurrentHashMap

class DiscoveryService(
    private val port: Int = 50051,  // Use same port as backend (50051)
    private val selfDeviceInfo: DeviceInfo,
    private val onDeviceDiscovered: (DeviceAnnounce) -> Unit
) {
    private val TAG = "DiscoveryService"
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    
    private var socket: DatagramSocket? = null
    private var isRunning = false
    
    // Track last seen times for stale detection
    private val lastSeen = ConcurrentHashMap<String, Long>()
    
    companion object {
        private const val BROADCAST_INTERVAL_MS = 5000L
        private const val STALE_TIMEOUT_MS = 30000L
        private const val CLEANUP_INTERVAL_MS = 10000L
        private const val MAX_MESSAGE_SIZE = 1024
        
        const val MESSAGE_TYPE_ANNOUNCE = "ANNOUNCE"
        const val MESSAGE_TYPE_LEAVE = "LEAVE"
    }

    data class DeviceAnnounce(
        val deviceId: String,
        val deviceName: String,
        val grpcAddr: String,
        val httpAddr: String,
        val platform: String,
        val arch: String,
        val hasCpu: Boolean,
        val hasGpu: Boolean,
        val hasNpu: Boolean,
        val canScreenCapture: Boolean
    )

    /**
     * Start the discovery service
     */
    fun start() {
        if (isRunning) {
            Log.w(TAG, "Discovery service already running")
            return
        }
        
        try {
            // Create socket on a different port for discovery since gRPC uses 50051
            // Use port 50052 for UDP discovery to avoid conflicts
            socket = DatagramSocket(50052)
            socket?.broadcast = true
            isRunning = true
            
            Log.i(TAG, "Discovery service started on UDP port 50052")
            
            // Start broadcast loop
            scope.launch { broadcastLoop() }
            
            // Start listen loop
            scope.launch { listenLoop() }
            
            // Start cleanup loop
            scope.launch { cleanupLoop() }
            
        } catch (e: Exception) {
            Log.e(TAG, "Failed to start discovery service", e)
            isRunning = false
        }
    }

    /**
     * Stop the discovery service
     */
    fun stop() {
        if (!isRunning) return
        
        isRunning = false
        
        // Send LEAVE message before stopping
        broadcastMessage(MESSAGE_TYPE_LEAVE)
        
        socket?.close()
        scope.cancel()
        
        Log.i(TAG, "Discovery service stopped")
    }

    /**
     * Broadcast loop - periodically announce presence
     */
    private suspend fun broadcastLoop() {
        // Immediate broadcast on startup
        broadcastMessage(MESSAGE_TYPE_ANNOUNCE)
        
        while (isRunning) {
            delay(BROADCAST_INTERVAL_MS)
            broadcastMessage(MESSAGE_TYPE_ANNOUNCE)
        }
    }

    /**
     * Listen loop - receive broadcasts from other devices
     */
    private suspend fun listenLoop() {
        val buffer = ByteArray(MAX_MESSAGE_SIZE)
        
        while (isRunning) {
            try {
                val packet = DatagramPacket(buffer, buffer.size)
                socket?.receive(packet)
                
                val message = String(packet.data, 0, packet.length)
                handleMessage(message)
                
            } catch (e: Exception) {
                if (isRunning) {
                    // Only log if we're still supposed to be running
                    Log.d(TAG, "Listen error: ${e.message}")
                }
            }
        }
    }

    /**
     * Cleanup loop - remove stale devices
     */
    private suspend fun cleanupLoop() {
        while (isRunning) {
            delay(CLEANUP_INTERVAL_MS)
            purgeStaleDevices()
        }
    }

    /**
     * Broadcast a discovery message
     */
    private fun broadcastMessage(type: String) {
        try {
            val deviceAnnounce = DeviceAnnounce(
                deviceId = selfDeviceInfo.deviceId,
                deviceName = selfDeviceInfo.deviceName,
                grpcAddr = selfDeviceInfo.grpcAddr,
                httpAddr = selfDeviceInfo.httpAddr,
                platform = selfDeviceInfo.platform,
                arch = selfDeviceInfo.arch,
                hasCpu = selfDeviceInfo.hasCpu,
                hasGpu = selfDeviceInfo.hasGpu,
                hasNpu = selfDeviceInfo.hasNpu,
                canScreenCapture = selfDeviceInfo.canScreenCapture
            )
            
            val message = JSONObject().apply {
                put("type", type)
                put("version", 1)
                put("ts", System.currentTimeMillis())
                put("device", JSONObject().apply {
                    put("device_id", deviceAnnounce.deviceId)
                    put("device_name", deviceAnnounce.deviceName)
                    put("grpc_addr", deviceAnnounce.grpcAddr)
                    put("http_addr", deviceAnnounce.httpAddr)
                    put("platform", deviceAnnounce.platform)
                    put("arch", deviceAnnounce.arch)
                    put("has_cpu", deviceAnnounce.hasCpu)
                    put("has_gpu", deviceAnnounce.hasGpu)
                    put("has_npu", deviceAnnounce.hasNpu)
                    put("can_screen_capture", deviceAnnounce.canScreenCapture)
                })
            }
            
            val data = message.toString().toByteArray()
            val broadcastAddr = InetAddress.getByName("255.255.255.255")
            val packet = DatagramPacket(data, data.size, broadcastAddr, port)
            
            socket?.send(packet)
            
        } catch (e: Exception) {
            Log.d(TAG, "Broadcast failed: ${e.message}")
        }
    }

    /**
     * Handle incoming discovery message
     */
    private fun handleMessage(message: String) {
        try {
            val json = JSONObject(message)
            val type = json.getString("type")
            val deviceJson = json.getJSONObject("device")
            
            val device = DeviceAnnounce(
                deviceId = deviceJson.getString("device_id"),
                deviceName = deviceJson.getString("device_name"),
                grpcAddr = deviceJson.getString("grpc_addr"),
                httpAddr = deviceJson.optString("http_addr", ""),
                platform = deviceJson.getString("platform"),
                arch = deviceJson.getString("arch"),
                hasCpu = deviceJson.getBoolean("has_cpu"),
                hasGpu = deviceJson.getBoolean("has_gpu"),
                hasNpu = deviceJson.getBoolean("has_npu"),
                canScreenCapture = deviceJson.getBoolean("can_screen_capture")
            )
            
            // Ignore our own broadcasts
            if (device.deviceId == selfDeviceInfo.deviceId) {
                return
            }
            
            when (type) {
                MESSAGE_TYPE_ANNOUNCE -> {
                    val wasKnown = lastSeen.containsKey(device.deviceId)
                    lastSeen[device.deviceId] = System.currentTimeMillis()
                    
                    if (!wasKnown) {
                        Log.i(TAG, "Found new device: ${device.deviceName} (${device.deviceId.take(8)}) at ${device.grpcAddr}")
                    }
                    
                    onDeviceDiscovered(device)
                }
                MESSAGE_TYPE_LEAVE -> {
                    lastSeen.remove(device.deviceId)
                    Log.i(TAG, "Device left: ${device.deviceName} (${device.deviceId.take(8)})")
                }
            }
            
        } catch (e: Exception) {
            Log.d(TAG, "Failed to parse message: ${e.message}")
        }
    }

    /**
     * Remove stale devices
     */
    private fun purgeStaleDevices() {
        val now = System.currentTimeMillis()
        val staleThreshold = now - STALE_TIMEOUT_MS
        
        val staleDevices = lastSeen.filter { (_, lastSeenTime) ->
            lastSeenTime < staleThreshold
        }
        
        staleDevices.forEach { (deviceId, _) ->
            lastSeen.remove(deviceId)
            Log.i(TAG, "Device ${deviceId.take(8)} marked stale")
        }
    }
}
