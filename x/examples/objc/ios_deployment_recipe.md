# How to Build & Deploy a Go Application on a Physical iPhone

This guide details the end-to-end process of compiling a Go application with Cgo, manually creating an application bundle, signing it with a developer certificate, and deploying it to a physical iOS device.

This is an advanced procedure that manually performs the steps Xcode typically automates.

---

### Part 1: Prerequisites

1. **Apple Developer Program Membership:** You must have an active membership.
2. **Physical iPhone or iPad:** The device you want to deploy to.
3. **Xcode and Command Line Tools:** Install the latest version from the Mac App Store.
4. **Go:** A working Go installation.
5. **Your Go Project:** For this example, we'll use the `x/examples/objc` project from the `outline-sdk`.

### Part 2: Apple Developer Portal Setup

First, you need to create the necessary credentials and profiles on the Apple Developer website.

1. **Get Your Device's UDID:**
    * Connect your iPhone to your Mac.
    * Open Xcode, go to `Window -> Devices and Simulators`.
    * Select your device to see the `Identifier` (this is the UDID).

2. **Register Your Device:**
    * Log in to the [Apple Developer Portal](https://developer.apple.com/account/).
    * Navigate to `Certificates, Identifiers & Profiles -> Devices`.
    * Click the `+` button and add your device using the UDID you just found.

3. **Create an App ID:**
    * Go to `Identifiers` and click `+`.
    * Select `App IDs` and continue.
    * Select `App` and continue.
    * **Description:** Give it a name (e.g., "Go ProcessInfo App").
    * **Bundle ID:** Select `Wildcard`. A wildcard ID is simpler for internal tools as it can be reused. Enter a Bundle ID like `com.yourcompany.*`. Note that wildcard App IDs cannot be used for apps that require specific capabilities like Push Notifications or In-App Purchase.
    * Note your **Team ID** (shown in the top right of the portal). Your final wildcard identifier will be `TEAM_ID.com.yourcompany.*`.

### Part 3: Building and Packaging the App

Now, you'll build the Go executable and package it into a `.app` bundle.

1. **Build the Go Executable:**
    Run the `go build` command with the correct environment variables for iOS. This command compiles the Go code into an ARM64 binary for iOS.

    ```sh
    CC="$(xcrun --sdk iphoneos --find cc) -isysroot \"$(xcrun --sdk iphoneos --show-sdk-path)\""
    GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
    go -C x build -v -o ./examples/objc/ProcessInfo.app ./examples/objc/main.go
    ```

    * This command creates a directory `ProcessInfo.app` and places the compiled executable named `main` inside it.

2. **Create the `Info.plist` File:**
    Every iOS app needs an `Info.plist`. Create this file inside your `ProcessInfo.app` directory.

    **File:** `x/examples/objc/ProcessInfo.app/Info.plist`

    ```xml
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
        <key>CFBundleIdentifier</key>
        <string>YOUR_TEAM_ID.com.yourcompany.testapp</string>
        <key>CFBundleExecutable</key>
        <string>main</string>
        <key>CFBundleName</key>
        <string>ProcessInfo</string>
        <key>CFBundlePackageType</key>
        <string>APPL</string>
        <key>CFBundleSignature</key>
        <string>????</string>
        <key>CFBundleVersion</key>
        <string>1.0</string>
        <key>CFBundleShortVersionString</key>
        <string>1.0</string>
    </dict>
    </plist>
    ```

    * **CRITICAL:** Change `YOUR_TEAM_ID.com.yourcompany.testapp` to a bundle ID that matches your wildcard App ID (e.g., `QT8Z3Q9V3A.org.getoutline.test`).
    * Ensure `CFBundleExecutable` matches the name of your executable (`main`).

3. **Create the `entitlements.plist` File:**
    This file must contain the entitlements specified in your provisioning profile. Create this file **outside** the `.app` bundle, in the same directory.

    **File:** `x/examples/objc/entitlements.plist`

    ```xml
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
        <key>application-identifier</key>
        <string>YOUR_TEAM_ID.com.yourcompany.testapp</string>
    </dict>
    </plist>
    ```

    * **CRITICAL:** Replace `YOUR_TEAM_ID` with your actual Team ID.
    * Replace `YOUR_TEAM_ID.com.yourcompany.testapp` with the same bundle ID you used in `Info.plist`.

### Part 4: Signing the Application

1. **Find Your Signing Identity:**
    Use the `security` command to find the name of your development certificate.

    ```sh
    security find-identity -v -p codesigning
    ```

    Look for a line that says `1) XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX "Apple Development: Your Name (XXXXXXXXXX)"`. The part in quotes is your identity.

2. **Run `codesign`:**
    Execute the `codesign` command from the directory containing your `.app` bundle and `entitlements.plist`.

    ```sh
    # Navigate to the correct directory
    cd x/examples/objc

    # Sign the app
    codesign -f -s "Apple Development: Your Name (XXXXXXXXXX)" --entitlements entitlements.plist ProcessInfo.app
    ```

You can use the identity `"Apple Development"` if it's unambiguous.

### Part 5: Deploying and Running

With the app correctly signed, you can now deploy it.

1. **Install the App:**
    Use `xcrun devicectl` to install the app. Replace `your_device_name` with your device's name (you can find it with `xcrun devicectl list devices`).

    ```sh
    xcrun devicectl device install app --device your_device_name x/examples/objc/ProcessInfo.app
    ```

2. **Launch and View Output:**
    Use `xcrun devicectl` again to launch the app and stream its console output directly to your terminal.

    ```sh
    xcrun devicectl device process launch --console --device your_device_name YOUR_TEAM_ID.com.yourcompany.testapp
    ```

    * Replace the bundle ID with the one you used.
    * The app will run on your device, and its output will appear in your terminal.

Here is a one-liner:

```sh
codesign -f -s "Apple Development" --entitlements ./x/examples/objc/entitlements.plist ./x/examples/objc/ProcessInfo.app && \
xcrun devicectl device install app --device your_device_name x/examples/objc/ProcessInfo.app && \
xcrun devicectl device process launch --console --device your_device_name YOUR_BUNDLE_ID
```
