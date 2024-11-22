# Building a Censorship-Resistant Android/iOS App with CapacitorJS and the Outline SDK Mobileproxy

This code lab guides you through creating a censorship-resistant Android/iOS app that wraps your Progressive Web App (PWA) using CapacitorJS and the Outline SDK Mobileproxy. This setup routes your site's traffic through the Outline proxy, bypassing network restrictions. Note that the source in this folder is representative of what your end product should look like once the code lab is completed.

**Prerequisites:**

* An existing PWA
* Make sure your development environment is set up with the following. [You can also follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup)
  * [Node.js](https://nodejs.org/en/)
  * [GoLang](https://go.dev/)
  * [Android Studio](https://developer.android.com/studio/)
  * [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/)
  * [Wireshark](https://www.wireshark.org/) and [Charles Proxy](https://www.charlesproxy.com/), to confirm the app is working

## Set up the Capacitor Project

* Follow the CapacitorJS Getting Started guide to initialize a new project: [https://capacitorjs.com/docs/getting-started](https://capacitorjs.com/docs/getting-started)

   ```bash
   npm init @capacitor/app # follow the instructions - make sure to pick an app ID that's unique!
   cd <my-app-dir>
   npm install
   ```

* Add iOS and Android platforms to your project:

   ```bash
   npm install @capacitor/android @capacitor/ios
   npx cap add android
   npx cap add ios
   ```

## Configure Capacitor to Load Your Website

* **Delete the `src` folder.**  Capacitor's default configuration assumes your web app is in this folder. We'll be loading your PWA directly from its URL.
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
    "appId": "com.yourcompany.yourapp",
    "appName": "YourAppName",
    "bundledWebRuntime": false,
    "server": {
      "url": "https://www.your-pwa-url.com" 
    },
    "ios": {
      "limitsNavigationsToAppBoundDomains": true
    }
   }
   ```

* In your iOS project's `ios/App/App/Info.plist`, add your PWA's URL to a `WKAppBoundDomains` array in the top level `<dict>`:

   ```xml
   <key>WKAppBoundDomains</key>
   <array>
     <string>www.your-pwa-url.com</string>
   </array>
   ```

  Be sure to add any URLs you navigate to within your PWA in this array as well!

## Test the Basic Apps

* Sync your changes, then run your app on a device or emulator:

  > [!NOTE]
  > Capacitor will silently fail to read your config if the JSON is invalid. Be sure to check `capacitor.config.json` for any hanging commas!

  ```bash
  npx cap run ios

  mkdir android/app/src/main/assets # sync will fail without this folder
  npx cap run android
  ```

* Ensure your PWA loads correctly within the Capacitor app.

## Integrate the Outline SDK Mobileproxy Library

**Build the Mobileproxy library:**
  * [Follow the instructions in the Outline SDK repository](https://github.com/Jigsaw-Code/outline-sdk/tree/main/x/mobileproxy#build-the-go-mobile-binaries-with-go-build):

  ```bash
    git clone https://github.com/Jigsaw-Code/outline-sdk.git
    cd outline-sdk/x
    go build -o "$(pwd)/out/" golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind

    PATH="$(pwd)/out:$PATH" gomobile bind -ldflags='-s -w' -target=ios -iosversion=11.0 -o "$(pwd)/out/mobileproxy.xcframework" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
    PATH="$(pwd)/out:$PATH" gomobile bind -ldflags='-s -w' -target=android -androidapi=21 -o "$(pwd)/out/mobileproxy.aar" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
  ```

### iOS Integration
  * Open the XCode project with `npx cap open ios`.
  * Add the compiled `mobileproxy.xcframework` static library to your Xcode project:
    * Right click on the `Frameworks` folder.
    * Select `Add Files to "App"...`
    * Select the `mobileproxy.xcframework` folder.

  * Create a new file called `OutlineBridgeViewController.swift`:
    * Right click on the `App` folder.
    * Select `New File...`
    * Select the `Swift File` and click `Next`.
    * Enter in the file name.

  * Extend the base Capacitor Bridge View Controller:
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

  * Import the Mobileproxy framework at the head of the `AppDelegate.swift` file:

  ```swift
    import Mobileproxy
  ```

  * Create a `proxy` property on the `AppDelegate`:

  ```swift
    var proxy: MobileproxyProxy?
  ```
  
  * Then set up the dialer and start the proxy in `applicationDidBecomeActive`:

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
  * **Convert your Android project to Kotlin.** Open the Android project with `npx cap open android`.
    * Navigate to `java/<your app ID>/MainActivity`
    * Right click on the file and select "Convert Java File to Kotlin File". Confirm the following dialogs.
    * Once done, you will need to right click the `MainActivity` a second time and select "Convert Java File to Kotlin File"

    [See the official instructions if you encounter any issues.](https://developer.android.com/kotlin/add-kotlin)

  
  * **Import dependencies:** 
    * Right click on `app` and select "Open Module Settings"
    * In the sidebar, navigate to "Dependencies"
    * Click the `+` button and select a Library Dependency
    * Search for `androidx.webkit` and select it.
    * Next we need to import the `mobileproxy.aar`. Click the `+` button again and select `JAR/AAR Dependency`.
    * Type in the path `../../outline-sdk/x/out/mobileproxy.aar`
    * Click `Apply`.

    * In the head of your new `MainActivity.kt`, import the following:

    ```kotlin
    import android.os.*
    import mobileproxy.*
    import androidx.webkit.*
    ```
 
  * Now, in your `MainActivity.kt`, confirm proxy override is available in `onCreate`:

    ```kotlin
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
            // Proxy override is supported
        }
    }
    ```

  * Initialize `mobileproxy` with a smart dialer in `onCreate`. Don't forget to replace `www.your-pwa-url.com`!:

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

* **iOS:** Start the emulator with `npx cap run ios`. Use [Charles Proxy](https://www.charlesproxy.com/) to view network traffic coming from your iOS simulator and observe DoH traffic coming from `9.9.9.9`.
* **Android:** Start the emulator with `npx cap run android`. Use Wireshark to capture network traffic. Filter by `ip.addr == 9.9.9.9` (your chosen DNS server). You should see TCP and TLS traffic, indicating that your app is using DNS over HTTPS (DoH).

## Building and Distributing your App

### iOS

TODO: you will probably need to create a provisioning profile for your new app - https://forum.ionicframework.com/t/ios-build-app-provisioning-errors/208170/2

### Android

TODO: you will need to create a keystore in android studio and configure the path in the capacitor.config.json https://forum.ionicframework.com/t/error-missing-options-keystore-path-keystore-password-keystore-key-alias-keystore-key-password/243217/3

**Important Notes:**

* Replace `"www.your-pwa-url.com"` with your actual PWA URL.
* This code lab provides a basic framework. You might need to adjust it depending on your PWA's specific requirements and the Outline SDK's updates.
* Consider adding error handling and UI elements to improve the user experience.

By following these steps, you can create an Android/iOS app that utilizes the Outline SDK Mobileproxy to circumvent censorship and access your PWA securely. This approach empowers users in restricted environments to access information and services freely.
