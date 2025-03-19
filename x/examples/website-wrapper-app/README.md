# Outline SDK Web Wrapper Example

This example demonstrates how to use the Outline SDK to create a censorship-resistant mobile app by wrapping an existing website. 

> [!NOTE]
> To turn your own website into a censorship-resistant app from scratch, please follow one of the following guides:
> - [iOS](docs/ios.md)
> - [Android](docs/android.md)


## Starting the Web Wrapper demo site (with same-origin navigation iframe) on the **iOS Simulator**

* You will need your site's domain.
* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`NGROK_TOKEN`](https://dashboard.ngrok.com/get-started/your-authtoken)
* You will need [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/). [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements)

```sh
NGROK_TOKEN="YOUR_NGROK_AUTH_TOKEN" TARGET_DOMAIN="YOUR_WEBSITE_DOMAIN" npm run start:app ios
```

Click the "play" button in XCode to start your iOS app!

## Starting the Web Wrapper demo site (with same-origin navigation iframe) in the **Android Emulator**

* You will need your site's domain.
* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`NGROK_TOKEN`](https://dashboard.ngrok.com/get-started/your-authtoken)
* You will need [OpenJDK 17](https://stackoverflow.com/a/70649641) and [Android Studio](https://developer.android.com/studio/) [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements)

```sh
NGROK_TOKEN="YOUR_NGROK_AUTH_TOKEN" TARGET_DOMAIN="YOUR_WEBSITE_DOMAIN" npm run start:app ios
```

Click the "play" button in Android Studio to start your Android app!

## TODO: adding additional whitelist domains