# Outline SDK Web Wrapper

## Getting Started

Clone the SDK to a local folder and navigate to the `x/examples/website-wrapper-app` directory.

```sh
git clone https://github.com/Jigsaw-Code/outline-sdk
cd outline-sdk/x/examples/website-wrapper-app
```

Both **iOS** and **Android** are currently confirmed to be working on MacOS. Use other platforms at your own risk.

## Building the app project for **iOS**

* You will need your site's domain and a list of domains that you would also like to load in your app.
* You will need [go](https://golang.org/) to build the SDK library.
* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/). [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements)

```sh
npm run reset
npm run build:project -- --platform=ios --entryDomain="www.mysite.com" [--additionalDomains="cdn.mysite.com,auth.mysite.com"] [--smartDialerConfig="<MY_SMART_DIALER_CONFIG_TEXT>"]
```

The path to the project will be printed into the console, you can open it in XCode. Click the "play" button in XCode to start your iOS app!

### Adding icon and splash screen assets to your generated iOS project

TODO: automate this process

You'll need to add the following images to the `assets` folder in your generated project:

A 1024x1024 png titled `icon.png` containing your app icon.
A 2732x2732 png titled `splash.png` containing your splash screen.
Another 2732x2732 png titled `splash-dark.png` containing your splash screen in dark mode.

Then, run the following command to generate and place the assets in the appropriate places in your iOS project:

```sh
npx capacitor-assets generate --ios
```

### Viewing your site in the example navigation iframe

Many sites don't handle their own navigation - if this applies to you, you can run a proxy to demonstrate what your site would look like in an example same-origin navigation iframe.

* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`--navigatorToken`](https://dashboard.ngrok.com/get-started/your-authtoken)

```sh
npm run start -- --platform=ios \
  --entryDomain="www.mysite.com" --additionalDomains="cdn.mysite.com,auth.mysite.com" \
  --navigatorToken="<YOUR_NGROK_AUTH_TOKEN>" --navigatorPath="/nav"
```

## Building the app project for **Android**

* You will need your site's domain list of domains that you would also like to load in your app.
* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need [go](https://golang.org/) to build the SDK library.
* You will need [OpenJDK 17](https://stackoverflow.com/a/70649641) and [Android Studio](https://developer.android.com/studio/) [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements)

```sh
npm run clean # no need to do this on a fresh install
npm run build:project -- --platform=android --entryDomain="www.mysite.com" [--additionalDomains="cdn.mysite.com,auth.mysite.com"] [--smartDialerConfig="<MY_SMART_DIALER_CONFIG_TEXT>"]
```

The path to the project will be printed into the console, open it in Android Studio.
Wait for Gradle to load your project. Click the "play" button in Android Studio to start your Android app!

### Adding icon and splash screen assets to your generated Android project

TODO: automate this process

You'll need to add the following images to the `assets` folder in your generated project:

A 1024x1024 png titled `icon.png` containing your app icon.
A 2732x2732 png titled `splash.png` containing your splash screen.
Another 2732x2732 png titled `splash-dark.png` containing your splash screen in dark mode.

Then, run the following command to generate and place the assets in the appropriate places in your Android project:

```sh
npx capacitor-assets generate --android
```

### Viewing your site in the example navigation iframe

Many sites don't handle their own navigation - if this applies to you, you can run a proxy to demonstrate what your site would look like in an example same-origin navigation iframe.

* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`--navigatorToken`](https://dashboard.ngrok.com/get-started/your-authtoken)

```sh
npm run start -- --platform=android \
  --entryDomain="www.mysite.com" --additionalDomains="cdn.mysite.com, auth.mysite.com" \
  --navigatorToken="<YOUR_NGROK_AUTH_TOKEN>" --navigatorPath="/nav"
```

## Troubleshooting

TODO: better troubleshooting
