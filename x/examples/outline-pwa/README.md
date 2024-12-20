# Wrapping your website in a Censorship-Resistant iOS App with CapacitorJS and the Outline SDK Mobileproxy

## Prerequisites

*   A website you want to make censorship resistant.
*   Set up your development environment with the following.
    *   [Node.js](https://nodejs.org/en/), for the Capacitor build system.
    *   [GoLang](https://go.dev/), to build the Outline Mobile Proxy.
    *   [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/). [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements)

## Important Notes

*   Replace `"www.your-website-url.com"` with your actual website URL.
*   This code lab provides a basic framework. You might need to adjust it depending on your website's specific requirements and the Outline SDK's updates.

## Create your app

**1. Set up the Capacitor Project**

Follow the CapacitorJS Getting Started guide to initialize a new project: [https://capacitorjs.com/docs/getting-started](https://capacitorjs.com/docs/getting-started)

> [!NOTE]
> This will create a new directory containing your project. Make sure you run these commands relative to where you want your project to live.

```bash
npm init @capacitor/app # follow the instructions - make sure to pick an app ID that's unique!
cd <my-app-dir>
npm install
```

Add iOS platform to your project:

```bash
npm install @capacitor/ios
npx cap add ios

npm run build # this builds the stock web app that the default project ships with
npx cap sync ios
```

**2. Confirm that the default app is able to run**

Open the iOS project in XCode with `npx cap open ios`.

Make sure the correct emulator or device is selected and press the ▶️ button.

**3. Configure Capacitor to Load Your Website**

Update `capacitor.config.json`:

```json
{
    "appId": "com.yourcompany.yourapp",
    "appName": "YourAppName",
    "bundledWebRuntime": false,
    "server": {
        "url": "https://www.your-website-url.com"
    }
}
```

> [!NOTE]
> Capacitor will silently fail to read your config if the JSON is invalid. Be sure to check `capacitor.config.json` for any hanging commas!

**4. Integrate the Outline SDK Mobileproxy Library**

 Build the Mobileproxy library:

[Follow the instructions in the Outline SDK repository](https://github.com/Jigsaw-Code/outline-sdk/tree/main/x/mobileproxy#build-the-go-mobile-binaries-with-go-build):

```bash
git clone https://github.com/Jigsaw-Code/outline-sdk.git
cd outline-sdk/x
go build -o "$(pwd)/out/" golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind

PATH="$(pwd)/out:$PATH" gomobile bind -ldflags='-s -w' -target=ios -iosversion=11.0 -o "$(pwd)/out/mobileproxy.xcframework" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
```
Open the XCode project with `npx cap open ios`.
Add the compiled `mobileproxy.xcframework` static library to your Xcode project:
*   Right click on the `Frameworks` folder.
*   Select `Add Files to "App"...`
*   Select the `mobileproxy.xcframework` folder.

Create a new file called `OutlineBridgeViewController.swift`:
*   Right click on the `App` folder.
*   Select `New File...`
*   Select the `Swift File` and click `Next`.
*   Enter in the file name.

Extend the base Capacitor Bridge View Controller:

```swift
import UIKit
import Capacitor

class OutlineBridgeViewController: CAPBridgeViewController {
    private let proxyHost: String = "127.0.0.1"
    private let proxyPort: String = "8080"
}
```

Override the `webView` method to inject the proxy configuration:

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

In `AppDelegate.swift`, set the `rootViewController` to your new `OutlineBridgeViewController`:

```swift
func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {

    self.window?.rootViewController = OutlineBridgeViewController()

    return true
}
```

Import the Mobileproxy framework at the head of the `AppDelegate.swift` file:

```swift
import Mobileproxy
```

Create a `proxy` property on the `AppDelegate`:

```swift
var proxy: MobileproxyProxy?
```

Then set up the dialer and start the proxy in `applicationDidBecomeActive`:

```swift
func applicationDidBecomeActive(_ application: UIApplication) {
    var dialerError: NSError?
    if let dialer = MobileproxyNewSmartStreamDialer(
        MobileproxyNewListFromLines("www.your-website-url.com"),
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

Stop the proxy in `applicationWillResignActive`:

```swift
func applicationWillResignActive(_ application: UIApplication) {
    if let proxy = self.proxy {
        proxy.stop(3000)
        self.proxy = nil
    }
}
```

**5. Verify with Packet Tracing**

Load XCode and connect your iOS device. Make sure that your device allows your developer certificate (Check `Settings > General > VPN & Device Management` on your iOS device). Go to `Product > Profile` in XCode and select the "Network" option from the list of presets. Pressing the record button on the window that pops up should launch your app and record its traffic. Once done, set the view on the packet trace data to `History` and among the results you should see the DNS address `dn9.quad9.net`.

**6. Building and Distributing your App**

* Create a developer account and add it to XCode Settings (`XCode > Settings... > Accounts`)
* Select your App in the file explorer and pick your Account's Team in `Signing & Capabilities`
* Connect your iOS device and select it as your build target in the top bar. XCode should automatically create a provisioning profile for the device and app combination.
* Build a production archive with `Product > Archive`
* On success, a window should pop up with your app listed in it. Select `Distribute App` and follow the instructions.

**7. Advanced: Add Support for Service Workers on iOS**

By default, Service workers aren't supported in iOS webviews. To activate them, you need to do the following:

In `capacitor.config.json`, enable `limitsNavigationsToAppBoundDomains`:

```json
{
    "appId": "com.yourcompany.yourapp",
    "appName": "YourAppName",
    "bundledWebRuntime": false,
    "server": {
        "url": "https://www.your-website-url.com"
    },
    "ios": {
        "limitsNavigationsToAppBoundDomains": true
    }
}
```

In your iOS project's `ios/App/App/Info.plist`, add your Website's URL to a `WKAppBoundDomains` array in the top level `<dict>`:

```xml
<key>WKAppBoundDomains</key>
<array>
    <string>www.your-website-url.com</string>
</array>
```

Be sure to add any URLs you navigate to within your website in this array as well!


---


# Building a Censorship-Resistant Android App with CapacitorJS and the Outline SDK Mobileproxy

## Prerequisites

*   A website you want to make censorship resistant.
*   Make sure your development environment is set up with the following. 
    *   [Node.js](https://nodejs.org/en/), for the Capacitor build system.
    *   [GoLang](https://go.dev/), to build the Outline Mobile Proxy.
    *   [OpenJDK 17](https://stackoverflow.com/a/70649641) and [Android Studio](https://developer.android.com/studio/) [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements)
*   [Wireshark](https://www.wireshark.org/), to confirm traffic is going through the Outline proxy

## Important Notes

*   Replace `"www.your-website-url.com"` with your actual website URL in all the relevant code snippets.
*   This code lab provides a basic framework. You might need to adjust it depending on your website's specific requirements and the Outline SDK's updates.
*   **Security:** Keep your key store file and passwords secure. Losing them can prevent you from updating your app in the future.
*   **Testing:** Thoroughly test your app on different devices and Android versions before releasing it.

## Create your app

**1. Set up the Capacitor Project**

Follow the CapacitorJS Getting Started guide to initialize a new project: [https://capacitorjs.com/docs/getting-started](https://capacitorjs.com/docs/getting-started)

> [!NOTE]
> This will create a new directory containing your project. Make sure you run these commands relative to where you want your project to live.

```bash
npm init @capacitor/app # follow the instructions - make sure to pick an app ID that's unique!
cd <my-app-dir>
npm install
```

Add Android platform to your project:

```bash
npm install @capacitor/android
npx cap add android

mkdir android/app/src/main/assets # sync will fail without this folder
npm run build # this builds the stock web app that the default project ships with
npx cap sync android
```

**2. Confirm that the default app is able to run**

Open the Android project in Android Studio with `npx cap open android`.

Ensure you have an emulator with Android API 35 or later (check `Tools > Device Manager`), then press the ▶️ button.

**3. Configure Capacitor to Load Your Website**

Update `capacitor.config.json`:

```json
{
    "appId": "com.yourcompany.yourapp",
    "appName": "YourAppName",
    "bundledWebRuntime": false,
    "server": {
        "url": "https://www.your-website-url.com"
    }
}
```

> [!NOTE]
> Capacitor will silently fail to read your config if the JSON is invalid. Be sure to check `capacitor.config.json` for any hanging commas!

**4. Integrate the Outline SDK Mobileproxy Library**

**Build the Mobileproxy library:**
[Follow the instructions in the Outline SDK repository](https://github.com/Jigsaw-Code/outline-sdk/tree/main/x/mobileproxy#build-the-go-mobile-binaries-with-go-build):

```bash
    git clone https://github.com/Jigsaw-Code/outline-sdk.git
    cd outline-sdk/x
    go build -o "$(pwd)/out/" golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind

    PATH="$(pwd)/out:$PATH" gomobile bind -ldflags='-s -w' -target=android -androidapi=21 -o "$(pwd)/out/mobileproxy.aar" github.com/Jigsaw-Code/outline-sdk/x/mobileproxy
```

**Convert your Android project to Kotlin.** Open the Android project with `npx cap open android`.

*   Navigate to `java/<your app ID>/MainActivity`
*   Right click on the file and select "Convert Java File to Kotlin File". Confirm the following dialogs.
*   Once done, you will need to right click the `MainActivity` a second time and select "Convert Java File to Kotlin File"

[See the official instructions if you encounter any issues.](https://developer.android.com/kotlin/add-kotlin)

**Update your Gradle files for Kotlin compatibility.**

*   Inside: `/android/app/build.gradle`, add `apply plugin: 'kotlin-android'` on line 2, directly under `apply plugin: 'com.android.application'`.
*   Inside: `/android/variables.gradle`, update the SDK variables to:

```kotlin
minSdkVersion = 26
compileSdkVersion = 35
targetSdkVersion = 35
```

**Import dependencies:**

*   Right click on `app` and select "Open Module Settings"
*   In the sidebar, navigate to "Dependencies"
*   Click the `+` button and select a Library Dependency
*   Search for `androidx.webkit` and select it.
*   Next we need to import the `mobileproxy.aar`. Click the `+` button again and select `JAR/AAR Dependency`.
*   Type in the path `../../outline-sdk/x/out/mobileproxy.aar`
*   Click `Apply`.
*   **Important:** The two added dependencies are initially placed in `/android/app/capacitor.build.gradle`, which is a generated file that gets reset frequently. To avoid losing these dependencies, manually move them to `/android/app/build.gradle`.
*   In the head of your new `MainActivity.kt`, import the following:

```kotlin
import android.os.*
import mobileproxy.*
import androidx.webkit.*
```

Now, in your `MainActivity.kt`, confirm proxy override is available in `onCreate`:

```kotlin
override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)

    if (WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
        // Proxy override is supported
    }
}
```

Initialize `mobileproxy` with a smart dialer in `onCreate`. Don't forget to replace `www.your-website-url.com`!:

```kotlin
private var proxy: Proxy? = null

override fun onCreate(savedInstanceState: Bundle?) {
    super.onCreate(savedInstanceState)

    if (WebViewFeature.isFeatureSupported(WebViewFeature.PROXY_OVERRIDE)) {
        this.proxy = Mobileproxy.runProxy(
            "127.0.0.1:0",
            Mobileproxy.newSmartStreamDialer(
                Mobileproxy.newListFromLines("www.your-website-url.com"),
                "{\"dns\":[{\"https\":{\"name\":\"9.9.9.9\"}}],\"tls\":[\"\",\"split:1\",\"split:2\",\"tlsfrag:1\"]}",
                Mobileproxy.newStderrLogWriter()
            )
        )
    }
}
```

Proxy all app requests after the proxy is initialized using `ProxyController`:

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

Turn off the proxy in `onDestroy`:

```kotlin
override fun onDestroy() {
    this.proxy?.stop(3000)
    this.proxy = null

    super.onDestroy()
}
```

**6. Verify with Packet Tracing**

Start the emulator with `npx cap run android`. Use Wireshark to capture network traffic. Filter by `ip.addr == 9.9.9.9` (your chosen DNS server). You should see TCP and TLS traffic, indicating that your app is using DNS over HTTPS (DoH).

**7. Building and Distributing your App**

First, generate a Key Store and use it to sign your app with Android Studio - follow these instructions: [https://developer.android.com/studio/publish/app-signing#generate-key](https://developer.android.com/studio/publish/app-signing#generate-key)

Note that you can choose to release your app as either an android app bundle (`.aab`) or an APK (`.apk`).

You need an android app bundle (`.aab`) to release your app in the Google Play Store. For this you will have to have a [Google Play Developer Account](https://play.google.com/console/u/0/developers) and at least twenty trusted testers to unlock production access.

*  Create a new application.
*  Fill in the required information (store listing, pricing, etc.).
*  Navigate to "App releases" and select a release track (e.g., internal testing, closed testing, open testing, or production).
*  Upload your `.aab` file.
*  Follow the instructions to complete the release process.

APKs (`.apk`) can be freely sideloaded onto your user's devices. For an APK, you will have to take care of distribution yourself. **Note that users need to enable "Install unknown apps"** on their Android devices to install apps from sources other than the Google Play Store. This setting is usually found under `Settings > Security` or `Settings > Apps & notifications > Special app access`.
