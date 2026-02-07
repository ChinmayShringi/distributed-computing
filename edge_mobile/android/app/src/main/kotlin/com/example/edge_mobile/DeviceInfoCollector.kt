package com.example.edge_mobile

import android.content.Context
import android.os.Build
import android.net.wifi.WifiManager
import com.example.edge_mobile.grpc.DeviceInfo
import java.net.NetworkInterface
import java.util.UUID

object DeviceInfoCollector {
    private const val PREF_NAME = "edge_mobile_prefs"
    private const val KEY_DEVICE_ID = "device_id"

    /**
     * Get or create a persistent device ID
     */
    private fun getOrCreateDeviceId(context: Context): String {
        val prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
        var deviceId = prefs.getString(KEY_DEVICE_ID, null)
        
        if (deviceId == null) {
            deviceId = UUID.randomUUID().toString()
            prefs.edit().putString(KEY_DEVICE_ID, deviceId).apply()
        }
        
        return deviceId
    }

    /**
     * Get the device's Wi-Fi IP address
     */
    private fun getWifiIpAddress(context: Context): String {
        try {
            // Try to get IP from NetworkInterface (works for all network types)
            val interfaces = NetworkInterface.getNetworkInterfaces()
            while (interfaces.hasMoreElements()) {
                val networkInterface = interfaces.nextElement()
                val addresses = networkInterface.inetAddresses
                
                while (addresses.hasMoreElements()) {
                    val address = addresses.nextElement()
                    
                    // Skip loopback and IPv6
                    if (!address.isLoopbackAddress && address.address.size == 4) {
                        return address.hostAddress ?: "0.0.0.0"
                    }
                }
            }
        } catch (e: Exception) {
            // Fallback to WifiManager (deprecated but works)
            try {
                val wifiManager = context.applicationContext.getSystemService(Context.WIFI_SERVICE) as? WifiManager
                wifiManager?.connectionInfo?.let { info ->
                    val ipAddress = info.ipAddress
                    if (ipAddress != 0) {
                        return String.format(
                            "%d.%d.%d.%d",
                            ipAddress and 0xff,
                            ipAddress shr 8 and 0xff,
                            ipAddress shr 16 and 0xff,
                            ipAddress shr 24 and 0xff
                        )
                    }
                }
            } catch (e2: Exception) {
                // Ignore
            }
        }
        
        return "0.0.0.0"
    }

    /**
     * Get CPU architecture
     */
    private fun getArchitecture(): String {
        return when (Build.SUPPORTED_ABIS[0]) {
            "arm64-v8a" -> "arm64"
            "armeabi-v7a" -> "armv7"
            "x86_64" -> "amd64"
            "x86" -> "x86"
            else -> Build.SUPPORTED_ABIS[0]
        }
    }

    /**
     * Check if device has GPU
     */
    private fun hasGpu(): Boolean {
        // Most Android devices have GPU
        return true
    }

    /**
     * Check if device has NPU (Snapdragon)
     */
    private fun hasNpu(): Boolean {
        // Check for Qualcomm Snapdragon with Hexagon DSP/NPU
        val hardware = Build.HARDWARE.lowercase()
        val board = Build.BOARD.lowercase()
        val soc = Build.SOC_MODEL.lowercase()
        
        return hardware.contains("qcom") || 
               hardware.contains("qualcomm") ||
               board.contains("qcom") ||
               soc.contains("snapdragon") ||
               soc.contains("sm") || // Snapdragon model numbers like SM8550
               soc.contains("qc")    // Qualcomm chip designation
    }

    /**
     * Get free RAM in MB
     */
    private fun getFreRamMb(context: Context): Long {
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as? android.app.ActivityManager
        return activityManager?.let {
            val memInfo = android.app.ActivityManager.MemoryInfo()
            it.getMemoryInfo(memInfo)
            (memInfo.availMem / (1024 * 1024))
        } ?: 0L
    }

    /**
     * Collect self device info for registration
     */
    fun getSelfDeviceInfo(context: Context): DeviceInfo {
        val deviceId = getOrCreateDeviceId(context)
        val ipAddress = getWifiIpAddress(context)
        
        return DeviceInfo.newBuilder()
            .setDeviceId(deviceId)
            .setDeviceName("${Build.MANUFACTURER} ${Build.MODEL}")
            .setPlatform("android")
            .setArch(getArchitecture())
            .setHasCpu(true)
            .setHasGpu(hasGpu())
            .setHasNpu(hasNpu())
            .setGrpcAddr("$ipAddress:50051")
            .setHttpAddr("")  // No HTTP bulk server on phone
            .setCanScreenCapture(false)
            .setRamFreeMb(getFreRamMb(context))
            .build()
    }
}
