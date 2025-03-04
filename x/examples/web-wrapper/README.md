# Outline SDK Web Wrapper Example

This example demonstrates how to use the Outline SDK to create a censorship-resistant mobile app by wrapping an existing website. 

> [!NOTE]
> To turn your own website into a censorship-resistant app from scratch, please follow one of the following guides:
> - [iOS](docs/ios.md)
> - [Android](docs/android.md)

## Running the example on the **iOS Simulator** via MacOS

### Install dependencies

* [Node.js](https://nodejs.org/en/), for the Capacitor build system.
* [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/). [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements)

```sh
# the demo website requires SSL for the app to load properly
brew install mkcert

npm run reset
```

### Start the demo site

```sh
# this creates the local certificate
npm run start:www
```

### Apply the local SSL certificate authority to the simulator and start the app

```sh
# add local.dev to the MacOS hosts file
echo "127.0.0.1 local.dev" >> /etc/hosts

# open the iOS project
npm run open:ios

# open the finder window containing the root CA
open "$(mkcert -CAROOT)"
```

Start the app, drag the root CA that mkcert generated into the simulator, then restart the app.


## Running the example on the **Android emulator** via MacOS

### Install dependencies

* [Node.js](https://nodejs.org/en/), for the Capacitor build system.
* [OpenJDK 17](https://stackoverflow.com/a/70649641) and [Android Studio](https://developer.android.com/studio/) [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements)

```sh
# the demo website requires SSL for the app to load properly
brew install mkcert

npm run reset
```

### Start the demo site

```sh
# this creates the local certificate
npm run start:www
```

### Apply the local SSL certificate authority to the emulator and start the app

```sh
# get list of avd ids and pick one
avdmanager list

# once your chosen emulator is running
emulator -avd '<emulator id>' -writable-system

# root and mount the emulator
adb root
adb disable-verify
adb reboot
adb root
adb remount

# you should see 'remounted /** as RW'
# now you can modify the emulator hosts file
adb shell "echo '10.0.2.2 local.dev' >> /etc/hosts"

# open the finder window containing the root CA
open "$(mkcert -CAROOT)"
```

Drag the root CA that mkcert generated into the simulator, then:

- Open the 'Settings' app.
- Go to `Security >> Ecryption & Credentials >> Install from storage`
- Select 'CA Certificate' from the list of options and accept the warning.
- Navigate to the root CA on the device and open it.
- Confirm the installation.

```sh
npm run start:android # (?)
```
