package org.outline.sdk.connectivity

import android.os.Bundle
import com.getcapacitor.BridgeActivity

class MainActivity : BridgeActivity() {
    @Override
    fun onCreate(state: Bundle) {
        registerPlugin(MobileBackendPlugin::class.java);
        super.onCreate(state);
    }
}