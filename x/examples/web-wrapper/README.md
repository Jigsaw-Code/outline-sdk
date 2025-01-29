# Outline SDK Web Wrapper Example

This example demonstrates how to use the Outline SDK to create a censorship-resistant mobile app by wrapping an existing website. 

> [!NOTE]
> To turn your own website into a censorship-resistant app from scratch, please follow one of the following guides:
> - [iOS](docs/ios.md)
> - [Android](docs/android.md)

## Running the example on MacOS & iOS

### Install dependencies

```sh
npm ci

# the demo website requires SSL for the app to load properly
brew install mkcert
```

### Generate and apply the local SSL certificate authority

```sh
# make sure your JAVA_HOME is setup before generating the SSL certificate
mkcert -install
mkdir dist
JAVA_HOME=$(/usr/libexec/java_home) mkcert -key-file dist/dev-key.pem -cert-file dist/dev.pem *.dev

# add local.dev to the hosts file
echo "127.0.0.1 local.dev" >> /etc/hosts

# open the iOS project
npx cap sync
npx cap open ios

# open the finder window containing the root CA
open "$(mkcert -CAROOT)"
```

Start the app and drag the root CA that mkcert generated into the simulator, then stop the app.

### Run the project

```sh
# run the demo site
npx serve --ssl-cert dist/dev.pem --ssl-key dist/dev-key.pem www

# in a separate terminal, run the ios application
npx cap run ios
```

## Running the example on MacOS & Android

TODO

```sh
# once your chosen emulator is running
emulator -avd <emulator id> -writable-system

# root and mount the emulator
adb root
adb disable-verify
adb reboot
adb root
adb remount

# you should see 'remounted /** as RW'
# now you can modify the hosts file
adb shell "echo '10.0.2.2 local.dev' >> /etc/hosts"

# TODO: generate and install the root CA on your android device
```