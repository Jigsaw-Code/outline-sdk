package org.getoutline.pwa

import android.os.Bundle
import com.getcapacitor.BridgeActivity

import mobileproxy.*

import androidx.webkit.ProxyConfig
import androidx.webkit.ProxyController
import androidx.webkit.WebViewFeature

// TODO: resize webview so the web content is not occluded by the device UI
class MainActivity : BridgeActivity() {
    private var proxy: Proxy? = null;

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
            this.proxy = Mobileproxy.runProxy(
                "127.0.0.1:0",
                Mobileproxy.newSmartStreamDialer(
                    Mobileproxy.newListFromLines("www.radiozamaneh.com"),
                    "{\"dns\":[{\"https\":{\"name\":\"9.9.9.9\"}}],\"tls\":[\"\",\"split:1\",\"split:2\",\"tlsfrag:1\"]}",
                    Mobileproxy.newStderrLogWriter()
                )
            )

            // NOTE: this affects all requests in the application
            ProxyController.getInstance()
                .setProxyOverride(
                    ProxyConfig.Builder()
                        .addProxyRule(this.proxy!!.address())
                        .build(),
                    {
                        runOnUiThread {
                            // Capacitor does not expose a way to defer the loading of the webview,
                            // so we simply refresh the page
                            this.bridge.webView.reload()
                        }
                    },
                    {}
                )
        }
    }

    override fun onDestroy() {
        this.proxy?.stop(3000)
        this.proxy = null

        super.onDestroy()
    }
}