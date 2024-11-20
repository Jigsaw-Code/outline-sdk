# Building a Censorship-Resistant Android/iOS App with CapacitorJS and the Outline SDK Mobileproxy

This code lab guides you through creating a censorship-resistant Android/iOS app that wraps your Progressive Web App (PWA) using CapacitorJS and the Outline SDK Mobileproxy. This setup routes your site's traffic through the Outline proxy, bypassing network restrictions. Note that the source in this folder is representative of what your end product should look like once the code lab is completed.

**Prerequisites:**

* Node.js and npm installed
* Android Studio with Kotlin support installed
* Xcode installed
* An existing PWA

## Set up the Capacitor Project

* Follow the CapacitorJS Getting Started guide to initialize a new project: [https://capacitorjs.com/docs/getting-started](https://capacitorjs.com/docs/getting-started)

   ```bash
   npx cap init
   ```

* Add iOS and Android platforms to your project:

   ```bash
   npx cap add android
   npx cap add ios
   ```

## Configure Capacitor to Load Your Website

* **Delete the `www` folder.**  Capacitor's default configuration assumes your web app is in this folder. We'll be loading your PWA directly from its URL.
* **Update `capacitor.config.json`:**

   ```json
   {
     "appId": "com.yourcompany.yourapp",
     "appName": "YourAppName",
     "bundledWebRuntime": false,
     "server": {
       "url": "https://www.your-pwa-url.com" 
     }
   }
   ```

## Add Support for Service Workers on iOS

* In `capacitor.config.json`, enable `limitsNavigationsToAppBoundDomains`:

   ```json
   {
     "ios": {
       "limitsNavigationsToAppBoundDomains": true
     }
   }
   ```

* In your iOS project's `Info.plist`, add your PWA's URL to the `WKAppBoundDomains` array:

   ```xml
   <key>WKAppBoundDomains</key>
   <array>
     <string>www.your-pwa-url.com</string>
   </array>
   ```

## Test the Basic App

* Run your app on a device or emulator:

   ```bash
   npx cap run ios
   npx cap run android
   ```

* Ensure your PWA loads correctly within the Capacitor app.

## Integrate the Mobileproxy Library

**Build the Mobileproxy library:**
  * Follow the instructions in the Outline SDK repository: [https://github.com/Jigsaw-Code/outline-sdk/tree/main/x/mobileproxy#build-the-go-mobile-binaries-with-go-build](https://github.com/Jigsaw-Code/outline-sdk/tree/main/x/mobileproxy#build-the-go-mobile-binaries-with-go-build)

### iOS Integration
  * Add the compiled `mobileproxy` static library to your Xcode project.
  * Create a new file called `OutlineBridgeViewController.swift` that looks like the following:
  ```swift
  import UIKit
  import Capacitor

  class OutlineBridgeViewController: CAPBridgeViewController {
      private let proxyHost: String = "127.0.0.1"
      private let proxyPort: String = "8080"      
  }
  ```
  * Override the `webView` method to inject the proxy configuration:

    ```swift
    override func webView(with frame: CGRect, configuration: WKWebViewConfiguration) -> WKWebView {
        if #available(iOS 17.0, *) {
            let endpoint = NWEndpoint.hostPort(
                host: NWEndpoint.Host(self.proxyHost), 
                port: NWEndpoint.Port(self.proxyPort)! 
            )
            let proxyConfig = ProxyConfiguration.init(httpCONNECTProxy: endpoint)

            let websiteDataStore = WKWebsiteDataStore.default()
            websiteDataStore.proxyConfigurations = [proxyConfig]

            configuration.websiteDataStore = websiteDataStore
        }

        return super.webView(with: frame, configuration: configuration)
    }
    ```

  * In `AppDelegate.swift`, set the `rootViewController` to your new `OutlineBridgeViewController`:

    ```swift
    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {

      self.window?.rootViewController = OutlineBridgeViewController()

      return true
    }
    ```

  * Then, set up the dialer and start the proxy in `applicationDidBecomeActive`:

    ```swift
    func applicationDidBecomeActive(_ application: UIApplication) {
        var dialerError: NSError?
        if let dialer = MobileproxyNewSmartStreamDialer(
            MobileproxyNewListFromLines("www.your-pwa-url.com"), 
            "{\"dns\":[{\"https\":{\"name\":\"9.9.9.9\"}}],\"tls\":[\"\",\"split:1\",\"split:2\",\"tlsfrag:1\"]}", 
            MobileproxyNewStderrLogWriter(),
            &dialerError
        ) {
            var proxyError: NSError?
            self.proxy = MobileproxyRunProxy(
                "127.0.0.1:8080", 
                dialer,
                &proxyError
            )
        }
    }
    ```

  * Stop the proxy in `applicationWillResignActive`:

    ```swift
    func applicationWillResignActive(_ application: UIApplication) {
        if let proxy = self.proxy {
            proxy.stop(3000)
            self.proxy = nil
        }
    }
    ```

### Android Integration
  * **Convert your Android project to Kotlin.** You can do this by following the instructions in the Android documentation: [https://developer.android.com/kotlin/add-kotlin](https://developer.android.com/kotlin/add-kotlin)
  * In your `MainActivity.kt`, confirm proxy override is available in `onCreate`:

    ```kotlin
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
            // Proxy override is supported
        }
    }
    ```

  * Initialize `mobileproxy` with a smart dialer in `onCreate`:

    ```kotlin
    private var proxy: Proxy? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
            this.proxy = Mobileproxy.runProxy(
                "127.0.0.1:0",
                Mobileproxy.newSmartStreamDialer(
                    Mobileproxy.newListFromLines("www.your-pwa-url.com"),
                    "{\"dns\":[{\"https\":{\"name\":\"9.9.9.9\"}}],\"tls\":[\"\",\"split:1\",\"split:2\",\"tlsfrag:1\"]}",
                    Mobileproxy.newStderrLogWriter()
                )
            )
        }
    }
    ```

  * Proxy all app requests after the proxy is initialized using `ProxyController`:

    ```kotlin
    // NOTE: This affects all requests in the application
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
    ```

  * Turn off the proxy in `onDestroy`:

    ```kotlin
    override fun onDestroy() {
        this.proxy?.stop(3000)
        this.proxy = null

        super.onDestroy()
    }
    ```

## Verify with Packet Tracing

* **Android:** Use Wireshark to capture network traffic. Filter by `ip.addr == 9.9.9.9` (your chosen DNS server). You should see TCP and TLS traffic, indicating that your app is using DNS over HTTPS (DoH).
* **iOS:** Use [Charles Proxy](https://www.charlesproxy.com/) to view network traffic coming from your iOS simulator and observe DoH traffic coming from `9.9.9.9`.

**Important Notes:**

* Replace `"www.your-pwa-url.com"` with your actual PWA URL.
* This code lab provides a basic framework. You might need to adjust it depending on your PWA's specific requirements and the Outline SDK's updates.
* Consider adding error handling and UI elements to improve the user experience.

By following these steps, you can create an Android/iOS app that utilizes the Outline SDK Mobileproxy to circumvent censorship and access your PWA securely. This approach empowers users in restricted environments to access information and services freely.
