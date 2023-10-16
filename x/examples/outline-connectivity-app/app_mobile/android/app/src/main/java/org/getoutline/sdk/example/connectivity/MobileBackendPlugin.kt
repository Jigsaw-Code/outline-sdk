package org.getoutline.sdk.example.connectivity

import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin

import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString

import shared_backend.Shared_backend
import java.nio.charset.Charset

@CapacitorPlugin(name = "MobileBackend")
class MobileBackendPlugin: Plugin() {
    @PluginMethod
    fun Request(call: PluginCall) {
        val output = JSObject()
        val response: FrontendResponse

        try {
            // TODO: encode directly to byte array
            val rawInputMessage = Json.encodeToString(
                FrontendRequest(
                    call.getString("resourceName")!!,
                    call.getString("parameters") ?: "{}"
                )
            )

            response = Json.decodeFromString(
                Shared_backend.handleRequest(
                    rawInputMessage.toByteArray(Charsets.UTF_8)
                ).toString(Charsets.UTF_8)
            )
        } catch (error: Exception) {
            output.put("error", error.message)

            return call.resolve(output)
        }

        output.put("body", response.body)
        output.put("error", response.error)

        return call.resolve(output)
    }
}