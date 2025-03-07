# Outline SDK Web Wrapper Example

This example demonstrates how to use the Outline SDK to create a censorship-resistant mobile app by wrapping an existing website. 

> [!NOTE]
> To turn your own website into a censorship-resistant app from scratch, please follow one of the following guides:
> - [iOS](docs/ios.md)
> - [Android](docs/android.md)


## Starting the Web Wrapper demo site (with same-origin navigation iframe)

* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`NGROK_TOKEN`](https://dashboard.ngrok.com/get-started/your-authtoken) and [`NGROK_DOMAIN`](https://dashboard.ngrok.com/domains)

```sh
NGROK_TOKEN="YOUR_NGROK_AUTH_TOKEN" NGROK_DOMAIN="YOUR_NGROK_DOMAIN" npm run start
```

Open your `NGROK_DOMAIN` in your browser to make sure it's working. You should not see any errors.


## Running the example on the **iOS Simulator** via MacOS

* Make sure the demo site is successfully running at your `NGROK_DOMAIN` ([See above](#starting-the-web-wrapper-demo-site)).
* You will need [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/). [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements)

```sh
# In a new terminal, open the iOS project:
npx cap open ios
```

Click the "play" button to start your iOS app!

## Running the example on the **Android emulator** via MacOS (WIP - currently crashing)

* Make sure the demo site is successfully running at your `NGROK_DOMAIN` ([See above](#starting-the-web-wrapper-demo-site)).
* You will need [OpenJDK 17](https://stackoverflow.com/a/70649641) and [Android Studio](https://developer.android.com/studio/) [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements)

```sh
# In a new terminal, open the android project:
npx cap open android
```

Click the "play" button to start your Android app!