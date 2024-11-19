package org.getoutline.pwa

import android.content.res.Resources
import android.graphics.Color
import android.os.Bundle
import android.util.Log
import android.view.View
import androidx.coordinatorlayout.widget.CoordinatorLayout
import android.view.WindowInsets
import com.getcapacitor.BridgeActivity
import kotlin.math.max

import mobileproxy.*

import androidx.webkit.ProxyConfig
import androidx.webkit.ProxyController
import androidx.webkit.WebViewFeature

class MainActivity : BridgeActivity() {
    private var proxy: Proxy? = null;

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        Log.d("DIMENSIONS", this.bridge.webView.layoutParams.width.toString())
        Log.d("DIMENSIONS", this.bridge.webView.layoutParams.height.toString())

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

    override fun onAttachedToWindow() {
        super.onAttachedToWindow()

        val rootView = this.window.decorView.rootView
        rootView.setBackgroundColor(Color.parseColor("#000000"))

        this.bridge.webView.overScrollMode = View.OVER_SCROLL_NEVER

        val rootWindowInsets = rootView.rootWindowInsets
        if (rootWindowInsets != null) {
            val systemBarInsets = rootWindowInsets.getInsets(WindowInsets.Type.systemBars())
            val cutoutInsets = rootWindowInsets.getInsets(WindowInsets.Type.displayCutout())

            val top = max(systemBarInsets.top, cutoutInsets.top)
            val left = max(systemBarInsets.left, cutoutInsets.left)
            val bottom = max(systemBarInsets.bottom, cutoutInsets.bottom)
            val right = max(systemBarInsets.right, cutoutInsets.right)

            this.bridge.webView.x = left.toFloat()
            this.bridge.webView.y = top.toFloat()

            this.bridge.webView.layoutParams = CoordinatorLayout.LayoutParams(
                Resources.getSystem().displayMetrics.widthPixels - right - left,
                Resources.getSystem().displayMetrics.heightPixels - bottom - top
            )
        }
    }

    override fun onDestroy() {
        this.proxy?.stop(3000)
        this.proxy = null

        super.onDestroy()
    }
}