package com.example.edge_mobile

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * AssistantClient handles REST API calls to the EdgeCLI server's /api/assistant endpoint.
 * This endpoint provides natural language command routing, device listing, and job execution.
 */
class AssistantClient(
    private val host: String = "10.0.2.2",
    private val port: Int = 8080
) {
    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(120, TimeUnit.SECONDS)  // LLM responses can take time
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    private val JSON_MEDIA_TYPE = "application/json; charset=utf-8".toMediaType()

    /**
     * Send a message to the assistant and get a response.
     *
     * @param text The natural language query (e.g., "list devices", "run ls on windows-pc")
     * @return Map containing: reply, raw (optional), mode (optional), job_id (optional), plan (optional)
     */
    suspend fun sendMessage(text: String): Map<String, Any?> = withContext(Dispatchers.IO) {
        val url = "http://$host:$port/api/assistant"

        val requestBody = JSONObject().apply {
            put("text", text)
        }.toString().toRequestBody(JSON_MEDIA_TYPE)

        val request = Request.Builder()
            .url(url)
            .post(requestBody)
            .build()

        try {
            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) {
                    throw Exception("Assistant request failed: ${response.code} ${response.message}")
                }

                val responseBody = response.body?.string()
                    ?: throw Exception("Empty response from assistant")

                parseAssistantResponse(responseBody)
            }
        } catch (e: Exception) {
            throw Exception("Failed to send message to assistant: ${e.message}", e)
        }
    }

    /**
     * Parse the assistant response JSON into a Map.
     */
    private fun parseAssistantResponse(json: String): Map<String, Any?> {
        val jsonObject = JSONObject(json)

        val result = mutableMapOf<String, Any?>(
            "reply" to jsonObject.optString("reply", "")
        )

        // Parse optional fields
        if (jsonObject.has("mode") && !jsonObject.isNull("mode")) {
            result["mode"] = jsonObject.getString("mode")
        }

        if (jsonObject.has("job_id") && !jsonObject.isNull("job_id")) {
            result["job_id"] = jsonObject.getString("job_id")
        }

        // Parse raw field (can be array of devices or other data)
        if (jsonObject.has("raw") && !jsonObject.isNull("raw")) {
            val raw = jsonObject.get("raw")
            result["raw"] = when (raw) {
                is JSONArray -> parseJsonArray(raw)
                is JSONObject -> parseJsonObject(raw)
                else -> raw.toString()
            }
        }

        // Parse plan field if present
        if (jsonObject.has("plan") && !jsonObject.isNull("plan")) {
            val plan = jsonObject.get("plan")
            result["plan"] = when (plan) {
                is JSONObject -> parseJsonObject(plan)
                else -> plan.toString()
            }
        }

        return result
    }

    /**
     * Convert JSONArray to List<Map>
     */
    private fun parseJsonArray(array: JSONArray): List<Map<String, Any?>> {
        return (0 until array.length()).map { i ->
            val item = array.get(i)
            when (item) {
                is JSONObject -> parseJsonObject(item)
                else -> mapOf("value" to item)
            }
        }
    }

    /**
     * Convert JSONObject to Map
     */
    private fun parseJsonObject(obj: JSONObject): Map<String, Any?> {
        val map = mutableMapOf<String, Any?>()
        obj.keys().forEach { key ->
            val value = obj.get(key)
            map[key] = when (value) {
                is JSONObject -> parseJsonObject(value)
                is JSONArray -> parseJsonArray(value)
                JSONObject.NULL -> null
                else -> value
            }
        }
        return map
    }

    /**
     * Update the server host and port.
     * Note: This creates a new connection on next request.
     */
    fun configure(newHost: String, newPort: Int): AssistantClient {
        return AssistantClient(newHost, newPort)
    }
}
