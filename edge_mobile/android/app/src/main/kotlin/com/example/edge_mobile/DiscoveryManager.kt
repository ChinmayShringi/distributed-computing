package com.example.edge_mobile

import android.content.Context
import android.util.Log
import com.example.edge_mobile.grpc.DeviceInfo
import kotlinx.coroutines.*
import org.json.JSONObject
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.NetworkInterface
import java.util.concurrent.ConcurrentHashMap

/**
 * DiscoveryManager is a singleton that handles UDP broadcast discovery
 * to find orchestrator servers on the local network.
 *
 * This mirrors the Go implementation in internal/discovery/discovery.go
 */
object DiscoveryManager {
    private const val TAG = "DiscoveryManager"

    // Discovery constants (matching Go implementation)
    private const val DISCOVERY_PORT = 50051
    private const val BROADCAST_INTERVAL_MS = 5000L
    private const val STALE_TIMEOUT_MS = 30000L
    private const val CLEANUP_INTERVAL_MS = 10000L
    private const val MAX_MESSAGE_SIZE = 1024

    private const val MESSAGE_TYPE_ANNOUNCE = "ANNOUNCE"
    private const val MESSAGE_TYPE_LEAVE = "LEAVE"

    // State
    private var socket: DatagramSocket? = null
    private var isRunning = false
    private var scope: CoroutineScope? = null

    // Discovered servers (device_id -> ServerInfo)
    private val discoveredServers = ConcurrentHashMap<String, ServerInfo>()
    private val lastSeen = ConcurrentHashMap<String, Long>()

    // Active server (the one we're connected to)
    @Volatile
    private var activeServer: ServerInfo? = null

    // Self device info (for broadcasting when in worker mode)
    private var selfDeviceInfo: DeviceInfo? = null
    private var shouldBroadcast = false

    // Listeners
    private val listeners = mutableListOf<DiscoveryListener>()

    data class ServerInfo(
        val deviceId: String,
        val deviceName: String,
        val grpcHost: String,
        val grpcPort: Int,
        val httpHost: String,
        val httpPort: Int,
        val platform: String,
        val arch: String,
        val hasLocalModel: Boolean = false,
        val localModelName: String = ""
    )

    interface DiscoveryListener {
        fun onServerDiscovered(server: ServerInfo)
        fun onServerLost(deviceId: String)
        fun onActiveServerChanged(server: ServerInfo?)
    }

    /**
     * Start discovery service (listen for broadcasts)
     * @param context Android context
     * @param broadcastSelf If true, also broadcast this device's presence
     * @param deviceInfo Device info for self-broadcast (required if broadcastSelf=true)
     */
    fun start(context: Context, broadcastSelf: Boolean = false, deviceInfo: DeviceInfo? = null) {
        if (isRunning) {
            Log.w(TAG, "Discovery already running")
            return
        }

        selfDeviceInfo = deviceInfo
        shouldBroadcast = broadcastSelf && deviceInfo != null

        scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

        try {
            // Bind to discovery port - use SO_REUSEADDR to allow multiple listeners
            socket = DatagramSocket(null).apply {
                reuseAddress = true
                broadcast = true
                bind(InetSocketAddress(DISCOVERY_PORT))
            }

            isRunning = true
            Log.i(TAG, "Discovery started on UDP port $DISCOVERY_PORT")

            // Start loops
            scope?.launch { listenLoop() }
            scope?.launch { cleanupLoop() }

            if (shouldBroadcast) {
                scope?.launch { broadcastLoop() }
            }

        } catch (e: Exception) {
            Log.e(TAG, "Failed to start discovery: ${e.message}")
            // Fallback: try alternate port if 50051 is in use
            tryFallbackPort()
        }
    }

    private fun tryFallbackPort() {
        try {
            // If 50051 is in use (e.g., gRPC server running), listen on 50052
            socket = DatagramSocket(null).apply {
                reuseAddress = true
                broadcast = true
                bind(InetSocketAddress(50052))
            }

            isRunning = true
            Log.i(TAG, "Discovery started on fallback UDP port 50052")

            scope?.launch { listenLoop() }
            scope?.launch { cleanupLoop() }

            if (shouldBroadcast) {
                scope?.launch { broadcastLoop() }
            }

        } catch (e: Exception) {
            Log.e(TAG, "Failed to start discovery on fallback port: ${e.message}")
            isRunning = false
        }
    }

    /**
     * Stop discovery service
     */
    fun stop() {
        if (!isRunning) return

        isRunning = false

        // Send LEAVE if we were broadcasting
        if (shouldBroadcast) {
            broadcastMessage(MESSAGE_TYPE_LEAVE)
        }

        socket?.close()
        scope?.cancel()
        scope = null

        Log.i(TAG, "Discovery stopped")
    }

    /**
     * Get list of all discovered servers
     */
    fun getDiscoveredServers(): List<ServerInfo> {
        return discoveredServers.values.toList()
    }

    /**
     * Get the currently active server
     */
    fun getActiveServer(): ServerInfo? {
        return activeServer
    }

    /**
     * Get gRPC host for active server (or null if none)
     */
    fun getActiveGrpcHost(): String? {
        return activeServer?.grpcHost
    }

    /**
     * Get gRPC port for active server (default 50051)
     */
    fun getActiveGrpcPort(): Int {
        return activeServer?.grpcPort ?: 50051
    }

    /**
     * Get HTTP host for active server (or null if none)
     */
    fun getActiveHttpHost(): String? {
        return activeServer?.httpHost
    }

    /**
     * Get HTTP port for active server (default 8080)
     */
    fun getActiveHttpPort(): Int {
        return activeServer?.httpPort ?: 8080
    }

    /**
     * Check if we have an active server connection
     */
    fun hasActiveServer(): Boolean {
        return activeServer != null
    }

    /**
     * Manually set the active server by device ID
     */
    fun setActiveServer(deviceId: String): Boolean {
        val server = discoveredServers[deviceId]
        if (server != null) {
            activeServer = server
            notifyActiveServerChanged(server)
            Log.i(TAG, "Active server set to: ${server.deviceName} at ${server.grpcHost}:${server.grpcPort}")
            return true
        }
        return false
    }

    /**
     * Manually set the active server by host/port (for fallback/manual config)
     */
    fun setActiveServerManual(host: String, grpcPort: Int = 50051, httpPort: Int = 8080) {
        val server = ServerInfo(
            deviceId = "manual-$host",
            deviceName = "Manual: $host",
            grpcHost = host,
            grpcPort = grpcPort,
            httpHost = host,
            httpPort = httpPort,
            platform = "unknown",
            arch = "unknown"
        )
        activeServer = server
        notifyActiveServerChanged(server)
        Log.i(TAG, "Active server manually set to: $host:$grpcPort")
    }

    /**
     * Add a discovery listener
     */
    fun addListener(listener: DiscoveryListener) {
        synchronized(listeners) {
            listeners.add(listener)
        }
    }

    /**
     * Remove a discovery listener
     */
    fun removeListener(listener: DiscoveryListener) {
        synchronized(listeners) {
            listeners.remove(listener)
        }
    }

    // ========== Internal Methods ==========

    private suspend fun listenLoop() {
        val buffer = ByteArray(MAX_MESSAGE_SIZE)

        while (isRunning) {
            try {
                val packet = DatagramPacket(buffer, buffer.size)

                // Set timeout to allow periodic checks
                socket?.soTimeout = 1000

                try {
                    socket?.receive(packet)
                } catch (e: java.net.SocketTimeoutException) {
                    continue // Normal timeout, check isRunning and retry
                }

                val sourceIp = packet.address.hostAddress ?: continue
                val message = String(packet.data, 0, packet.length)

                handleMessage(message, sourceIp)

            } catch (e: Exception) {
                if (isRunning) {
                    Log.d(TAG, "Listen error: ${e.message}")
                }
            }
        }
    }

    private fun handleMessage(message: String, sourceIp: String) {
        try {
            val json = JSONObject(message)
            val type = json.getString("type")
            val deviceJson = json.getJSONObject("device")

            val deviceId = deviceJson.getString("device_id")
            val deviceName = deviceJson.getString("device_name")
            var grpcAddr = deviceJson.getString("grpc_addr")
            var httpAddr = deviceJson.optString("http_addr", "")

            // Ignore our own broadcasts
            if (selfDeviceInfo != null && deviceId == selfDeviceInfo?.deviceId) {
                return
            }

            // Fix addresses that use 0.0.0.0 or 127.0.0.1 (same as Go)
            grpcAddr = fixAddress(grpcAddr, sourceIp)
            httpAddr = fixAddress(httpAddr, sourceIp)

            // Parse host:port
            val (grpcHost, grpcPort) = parseHostPort(grpcAddr, 50051)
            val (httpHost, httpPort) = parseHostPort(httpAddr, 8080)

            val server = ServerInfo(
                deviceId = deviceId,
                deviceName = deviceName,
                grpcHost = grpcHost,
                grpcPort = grpcPort,
                httpHost = httpHost,
                httpPort = httpPort,
                platform = deviceJson.getString("platform"),
                arch = deviceJson.getString("arch"),
                hasLocalModel = deviceJson.optBoolean("has_local_model", false),
                localModelName = deviceJson.optString("local_model_name", "")
            )

            when (type) {
                MESSAGE_TYPE_ANNOUNCE -> {
                    val wasKnown = discoveredServers.containsKey(deviceId)
                    discoveredServers[deviceId] = server
                    lastSeen[deviceId] = System.currentTimeMillis()

                    if (!wasKnown) {
                        Log.i(TAG, "Found server: ${server.deviceName} at ${server.grpcHost}:${server.grpcPort}")
                        notifyServerDiscovered(server)

                        // Auto-select first discovered server if none active
                        if (activeServer == null) {
                            activeServer = server
                            notifyActiveServerChanged(server)
                            Log.i(TAG, "Auto-selected server: ${server.deviceName}")
                        }
                    }
                }
                MESSAGE_TYPE_LEAVE -> {
                    discoveredServers.remove(deviceId)
                    lastSeen.remove(deviceId)
                    Log.i(TAG, "Server left: $deviceName")
                    notifyServerLost(deviceId)

                    // If active server left, try to select another
                    if (activeServer?.deviceId == deviceId) {
                        activeServer = discoveredServers.values.firstOrNull()
                        notifyActiveServerChanged(activeServer)
                    }
                }
            }

        } catch (e: Exception) {
            Log.d(TAG, "Failed to parse message: ${e.message}")
        }
    }

    /**
     * Fix addresses that use 0.0.0.0, 127.0.0.1, or localhost
     * Replace with actual source IP from UDP packet
     */
    private fun fixAddress(addr: String, sourceIp: String): String {
        if (addr.isEmpty()) return "$sourceIp:50051"

        val parts = addr.split(":")
        if (parts.size != 2) return addr

        val host = parts[0]
        val port = parts[1]

        return if (host == "0.0.0.0" || host == "127.0.0.1" || host == "localhost") {
            "$sourceIp:$port"
        } else {
            addr
        }
    }

    private fun parseHostPort(addr: String, defaultPort: Int): Pair<String, Int> {
        if (addr.isEmpty()) return Pair("", defaultPort)

        val parts = addr.split(":")
        return if (parts.size == 2) {
            Pair(parts[0], parts[1].toIntOrNull() ?: defaultPort)
        } else {
            Pair(addr, defaultPort)
        }
    }

    private suspend fun broadcastLoop() {
        // Immediate broadcast on startup
        broadcastMessage(MESSAGE_TYPE_ANNOUNCE)

        while (isRunning) {
            delay(BROADCAST_INTERVAL_MS)
            if (isRunning) {
                broadcastMessage(MESSAGE_TYPE_ANNOUNCE)
            }
        }
    }

    private fun broadcastMessage(type: String) {
        val deviceInfo = selfDeviceInfo ?: return

        try {
            val message = JSONObject().apply {
                put("type", type)
                put("version", 1)
                put("ts", System.currentTimeMillis())
                put("device", JSONObject().apply {
                    put("device_id", deviceInfo.deviceId)
                    put("device_name", deviceInfo.deviceName)
                    put("grpc_addr", deviceInfo.grpcAddr)
                    put("http_addr", deviceInfo.httpAddr)
                    put("platform", deviceInfo.platform)
                    put("arch", deviceInfo.arch)
                    put("has_cpu", deviceInfo.hasCpu)
                    put("has_gpu", deviceInfo.hasGpu)
                    put("has_npu", deviceInfo.hasNpu)
                    put("can_screen_capture", deviceInfo.canScreenCapture)
                })
            }

            val data = message.toString().toByteArray()

            // Send to all broadcast addresses
            val broadcastAddrs = getBroadcastAddresses()
            for (addr in broadcastAddrs) {
                try {
                    val packet = DatagramPacket(data, data.size, addr, DISCOVERY_PORT)
                    socket?.send(packet)
                } catch (e: Exception) {
                    Log.d(TAG, "Broadcast to $addr failed: ${e.message}")
                }
            }

            // Also send to 255.255.255.255 as fallback
            try {
                val fallbackAddr = InetAddress.getByName("255.255.255.255")
                val packet = DatagramPacket(data, data.size, fallbackAddr, DISCOVERY_PORT)
                socket?.send(packet)
            } catch (e: Exception) {
                Log.d(TAG, "Fallback broadcast failed: ${e.message}")
            }

        } catch (e: Exception) {
            Log.d(TAG, "Broadcast failed: ${e.message}")
        }
    }

    /**
     * Get all broadcast addresses for local networks (matching Go implementation)
     */
    private fun getBroadcastAddresses(): List<InetAddress> {
        val results = mutableListOf<InetAddress>()

        try {
            val interfaces = NetworkInterface.getNetworkInterfaces()
            while (interfaces.hasMoreElements()) {
                val iface = interfaces.nextElement()

                // Skip loopback and down interfaces
                if (iface.isLoopback || !iface.isUp) continue

                for (addr in iface.interfaceAddresses) {
                    val broadcast = addr.broadcast
                    if (broadcast != null) {
                        results.add(broadcast)
                    }
                }
            }
        } catch (e: Exception) {
            Log.d(TAG, "Failed to get broadcast addresses: ${e.message}")
        }

        return results
    }

    private suspend fun cleanupLoop() {
        while (isRunning) {
            delay(CLEANUP_INTERVAL_MS)
            purgeStaleServers()
        }
    }

    private fun purgeStaleServers() {
        val now = System.currentTimeMillis()
        val staleThreshold = now - STALE_TIMEOUT_MS

        val staleIds = lastSeen.filter { (_, time) -> time < staleThreshold }.keys

        for (deviceId in staleIds) {
            discoveredServers.remove(deviceId)
            lastSeen.remove(deviceId)
            Log.i(TAG, "Server ${deviceId.take(8)} marked stale")
            notifyServerLost(deviceId)

            // If active server went stale, try to select another
            if (activeServer?.deviceId == deviceId) {
                activeServer = discoveredServers.values.firstOrNull()
                notifyActiveServerChanged(activeServer)
            }
        }
    }

    // ========== Listener Notifications ==========

    private fun notifyServerDiscovered(server: ServerInfo) {
        synchronized(listeners) {
            for (listener in listeners) {
                try {
                    listener.onServerDiscovered(server)
                } catch (e: Exception) {
                    Log.e(TAG, "Listener error: ${e.message}")
                }
            }
        }
    }

    private fun notifyServerLost(deviceId: String) {
        synchronized(listeners) {
            for (listener in listeners) {
                try {
                    listener.onServerLost(deviceId)
                } catch (e: Exception) {
                    Log.e(TAG, "Listener error: ${e.message}")
                }
            }
        }
    }

    private fun notifyActiveServerChanged(server: ServerInfo?) {
        synchronized(listeners) {
            for (listener in listeners) {
                try {
                    listener.onActiveServerChanged(server)
                } catch (e: Exception) {
                    Log.e(TAG, "Listener error: ${e.message}")
                }
            }
        }
    }
}
