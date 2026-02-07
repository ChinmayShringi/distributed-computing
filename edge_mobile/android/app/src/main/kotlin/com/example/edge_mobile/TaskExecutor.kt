package com.example.edge_mobile

import android.content.Context
import android.os.Build
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.BufferedReader
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL

object TaskExecutor {
    
    /**
     * Execute a task based on kind
     */
    suspend fun execute(context: Context, kind: String, input: String): String = withContext(Dispatchers.IO) {
        when (kind) {
            "SYSINFO" -> collectSysInfo(context)
            "ECHO" -> "echo: $input"
            "LLM_GENERATE" -> executeLLMGenerate(input)
            else -> throw IllegalArgumentException("Unknown task kind: $kind")
        }
    }

    /**
     * Collect system information
     */
    private fun collectSysInfo(context: Context): String {
        val deviceInfo = DeviceInfoCollector.getSelfDeviceInfo(context)
        
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as? android.app.ActivityManager
        val memInfo = android.app.ActivityManager.MemoryInfo()
        activityManager?.getMemoryInfo(memInfo)
        
        val totalRamMb = memInfo.totalMem / (1024 * 1024)
        val freeRamMb = memInfo.availMem / (1024 * 1024)
        val usedRamMb = totalRamMb - freeRamMb
        
        return buildString {
            appendLine("Device: ${deviceInfo.deviceName}")
            appendLine("Platform: ${deviceInfo.platform}")
            appendLine("Architecture: ${deviceInfo.arch}")
            appendLine("Android Version: ${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})")
            appendLine("CPU Cores: ${Runtime.getRuntime().availableProcessors()}")
            appendLine("Total RAM: ${totalRamMb} MB")
            appendLine("Used RAM: ${usedRamMb} MB")
            appendLine("Free RAM: ${freeRamMb} MB")
            appendLine("Has GPU: ${deviceInfo.hasGpu}")
            appendLine("Has NPU: ${deviceInfo.hasNpu}")
            appendLine("Manufacturer: ${Build.MANUFACTURER}")
            appendLine("Model: ${Build.MODEL}")
            appendLine("Board: ${Build.BOARD}")
            appendLine("Hardware: ${Build.HARDWARE}")
            appendLine("SOC: ${Build.SOC_MODEL}")
            appendLine("Device ID: ${deviceInfo.deviceId}")
            appendLine("gRPC Address: ${deviceInfo.grpcAddr}")
        }
    }

    /**
     * Execute LLM generation
     * Tries multiple backends in order:
     * 1. HTTP endpoint (Ollama/Termux) at localhost:11434
     * 2. Fallback message if no LLM available
     */
    private suspend fun executeLLMGenerate(prompt: String): String = withContext(Dispatchers.IO) {
        // Try HTTP endpoint (Ollama/LM Studio in Termux)
        val endpoints = listOf(
            "http://127.0.0.1:11434",  // Ollama default
            "http://127.0.0.1:8080",   // Alternative
            "http://localhost:11434"
        )
        
        for (endpoint in endpoints) {
            try {
                val result = tryHttpEndpoint(endpoint, prompt)
                if (result != null) {
                    return@withContext result
                }
            } catch (e: Exception) {
                // Continue to next endpoint
            }
        }
        
        // No LLM available
        return@withContext "LLM_GENERATE not available: No on-device LLM endpoint found. " +
                "To enable, install Ollama in Termux and start it with 'ollama serve'."
    }

    /**
     * Try OpenAI-compatible HTTP endpoint
     */
    private fun tryHttpEndpoint(endpoint: String, prompt: String): String? {
        try {
            val url = URL("$endpoint/v1/chat/completions")
            val conn = url.openConnection() as HttpURLConnection
            
            conn.requestMethod = "POST"
            conn.setRequestProperty("Content-Type", "application/json")
            conn.doOutput = true
            conn.connectTimeout = 5000  // 5 second timeout
            conn.readTimeout = 30000     // 30 second timeout
            
            val jsonPayload = """
                {
                    "model": "qwen3:8b",
                    "messages": [{"role": "user", "content": "$prompt"}],
                    "temperature": 0.7,
                    "max_tokens": 1024,
                    "stream": false
                }
            """.trimIndent()
            
            conn.outputStream.use { os ->
                os.write(jsonPayload.toByteArray())
            }
            
            val responseCode = conn.responseCode
            if (responseCode == 200) {
                val response = BufferedReader(InputStreamReader(conn.inputStream)).use { it.readText() }
                
                // Parse JSON response (simple extraction, no library needed)
                val contentRegex = """"content"\s*:\s*"([^"]*)"""".toRegex()
                val match = contentRegex.find(response)
                return match?.groupValues?.get(1)?.replace("\\n", "\n")
            }
        } catch (e: Exception) {
            // Endpoint not available
        }
        
        return null
    }
}
