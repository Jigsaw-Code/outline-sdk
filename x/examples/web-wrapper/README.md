# Outline SDK Web Wrapper Example

This example demonstrates how to use the Outline SDK to create a censorship-resistant mobile app by wrapping an existing website. 

> [!NOTE]
> To turn your own website into a censorship-resistant app from scratch, please follow one of the following guides:
> - [iOS](docs/ios.md)
> - [Android](docs/android.md)


## Starting the Web Wrapper demo site.

* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need [mkcert](https://github.com/FiloSottile/mkcert), which can be installed via `brew install mkcert`. The wrapper will not load the site without TLS.

```sh
# Install the mkcert root CA:
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$(mkcert -CAROOT)/rootCA.pem"

# Start the web server. This also creates the local certificate via `npm run cert:create`.
npm ci
npm run start:www
```

Open `https://local.dev:3000` in your browser to make sure it's working. You should not see any errors.


## Running the example on the **iOS Simulator** via MacOS

* Make sure the demo site is successfully running at `https://local.dev:3000` ([See above](#starting-the-web-wrapper-demo-site)).
* You will need [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/). [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements)

```sh
# In a new terminal, open the iOS project:
npm run open:ios

# Open the finder window containing the root CA:
open "$(mkcert -CAROOT)"
```

Start the app in XCode, drag the `rootCA.pem` that mkcert generated into the simulator, then restart the app.

## Running the example on the **Android emulator** via MacOS (WIP)

* Make sure the demo site is successfully running at `https://local.dev:3000` ([See above](#starting-the-web-wrapper-demo-site)).
* You will need [OpenJDK 17](https://stackoverflow.com/a/70649641) and [Android Studio](https://developer.android.com/studio/) [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements)

```sh
# Get the list of android virtual device (AVD) IDs and pick one:
avdmanager list

# Start your chosen emulator:
emulator -avd '<emulator id>' -writable-system

# Re-mount the emulator in a readable/writable state:
adb root
adb disable-verify
adb reboot
adb root
adb remount

# You should see 'remounted /** as RW'.
# Now you can modify the emulator hosts file:
adb shell "echo '10.0.2.2 localhost' >> /etc/hosts"

# Open the finder window containing the root CA:
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
