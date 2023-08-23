package org.outline.sdk.connectivity

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import kotlinx.serialization.json.Json;
import kotlinx.serialization.encodeToString;

import shared_backend.Shared_backend;
import java.nio.charset.Charset;

@CapacitorPlugin(name = "MobilePlugin")
class MobileBackendPlugin: Plugin() {
    @PluginMethod()
    fun Invoke(call: PluginCall) {
        val output = JSObject();
        val outputMessage: RawBackendCallOutputMessage;

        try {
            // TODO: encode directly to byte array
            val rawInputMessage = Json.encodeToString(
                RawBackendCallInputMessage(
                    call.getString("method")!!,
                    call.getString("input")!!
                )
            );

            outputMessage = Json.decodeFromString(
                Shared_backend.sendRawCall(
                    rawInputMessage.toByteArray(
                        Charset.forName("utf8")
                    )
                ).toString()
            );
        } catch (error: Exception) {
            output.put("errors", listOf(error.message));

            return call.resolve(output);
        }

        output.put("result", outputMessage.result);
        output.put("errors", outputMessage.errors);

        return call.resolve(output);
    }
}