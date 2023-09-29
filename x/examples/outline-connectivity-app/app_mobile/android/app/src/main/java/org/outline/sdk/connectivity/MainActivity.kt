package org.getoutline.sdk.example.connectivity

import android.os.Bundle
import com.getcapacitor.BridgeActivity

class MainActivity : BridgeActivity() {
    override fun onCreate(state: Bundle?) {
        registerPlugin(MobileBackendPlugin::class.java)
        super.onCreate(state)
    }
}
