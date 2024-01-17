# Local Proxy with Fyne

This folder has a graphical application that runs a local proxy given a address and configuration.
It uses [Fyne](https://fyne.io/) for the UI.

<img width="231" alt="image" src="https://github.com/Jigsaw-Code/outline-sdk/assets/113565/33bdcefc-7fff-44b4-a70b-a892fd5c9d3b">

## Desktop

You can run the app without explicitly cloning the repository with:

```sh
go run github.com/Jigsaw-Code/outline-sdk/x/examples/fyne-proxy@latest
```

To run the local version while developing, from the `fyne-proxy` directory:

```sh
go run .
```

To package, from the app folder:

```sh
go run fyne.io/fyne/v2/cmd/fyne package
```


## Android

To run the app, start the emulator, call fyne install to build and install the app. See https://developer.android.com/studio/run/emulator-commandline

```sh
# Point ANDROID_NDK_HOME to the right location
export ANDROID_NDK_HOME="$HOME/Library/Android/sdk/ndk/26.1.10909125"
# Start the emulator.
$ANDROID_HOME/emulator/emulator -no-boot-anim -avd Pixel_3a_API_33_arm64-v8a
go run fyne.io/fyne/v2/cmd/fyne install -os android
```

If you need, you can build the APK, then install like this:

```sh
go run fyne.io/fyne/v2/cmd/fyne package -os android -appID com.example.myapp
$ANDROID_HOME/platform-tools/adb install ./Local_Proxy_Prototype.apk
```

## iOS

Install on a running simulator:

```sh
go run fyne.io/fyne/v2/cmd/fyne install -os iossimulator
```

If you package it first, you can install the .app with:

```sh
xcrun simctl install booted ./Local_Proxy_Prototype.app
```

To install on a real device, you need `ios-deploy` (`brew install ios-deploy`). After you connect your phone, run:

```sh
go run fyne.io/fyne/v2/cmd/fyne install -os ios
```
