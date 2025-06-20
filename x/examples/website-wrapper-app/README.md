# Outline SDK Web Wrapper

## Getting Started

Clone the SDK to a local folder and navigate to the `x/examples/website-wrapper-app` directory.

```sh
git clone https://github.com/Jigsaw-Code/outline-sdk
cd outline-sdk/x/examples/website-wrapper-app
```

To verify that your system has the necessary dependencies to generate your web wrapper project, run the web wrapper doctor:

```sh
./doctor
```

## Building the app project for **iOS**

> [!WARNING]
> You can only build iOS apps on MacOS.
> Currently only works with build targets of iOS 17.2 (and below?)

* You will need the url you want to load initially in your app.
* You will need [go](https://golang.org/) to build the SDK library.
* You will need [Node.js](https://nodejs.org/en/) for the project setup and web server.
* You will need [XCode](https://developer.apple.com/xcode/). 
* You will need [cocoapods](https://cocoapods.org/). 

[Please refer to CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements) and run `./doctor` to check to see if you have all the required dependencies.

```sh
npm run reset
npm run build:project -- --platform=ios --entryUrl="https://www.example.com"
npm run open:ios
```

Click the "play" button in XCode to start your iOS app!

[See below for the list of available configuration options.](#available-configuration-options)

### Adding icon and splash screen assets to your generated iOS project

> [!NOTE]
> TODO: automate this process

You'll need to add the following images to the `assets` folder in your generated project:

- A 1024x1024 png titled `icon.png` containing your app icon.
- A 2732x2732 png titled `splash.png` containing your splash screen.
- Another 2732x2732 png titled `splash-dark.png` containing your splash screen in dark mode.

Then, run the following command to generate and place the assets in the appropriate places in your iOS project:

```sh
npx capacitor-assets generate --ios
```

### Publishing your app in the App Store

[Follow these instructions on how to publish your app for beta testing and the App Store.](https://developer.apple.com/documentation/xcode/distributing-your-app-for-beta-testing-and-releases)

## Building the app project for **Android**

> [!WARNING]
> If you want to build Android on Windows, please use [Windows Subsystem for Linux (WSL)](https://learn.microsoft.com/en-us/windows/wsl/install)

* You will need the url you want to load initially in your app.
* You will need [Node.js](https://nodejs.org/en/) for the project setup and web server.
* You will need [go](https://golang.org/) to build the SDK library.
* You will need [JDK 17](https://stackoverflow.com/a/70649641) to build the app.
* You will need [Android Studio](https://developer.android.com/studio/).
  * Make sure to [install the NDK](https://developer.android.com/studio/projects/install-ndk#default-version).
  * Make sure to [set the correct JDK](https://stackoverflow.com/a/30631386).

[Please refer to CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements) and run `./doctor` to check to see if you have all the required dependencies.

```sh
npm run reset
npm run build:project -- --platform=android --entryUrl="https://www.example.com"
npm run open:android
```

Wait for Gradle to load your project. Click the "play" button in Android Studio to start your Android app!

[See below for the list of available configuration options.](#available-configuration-options)

### Adding icon and splash screen assets to your generated Android project

> [!NOTE]
> TODO: automate this process

You'll need to add the following images to the `assets` folder in your generated project:

- A 1024x1024 png titled `icon.png` containing your app icon.
- A 2732x2732 png titled `splash.png` containing your splash screen.
- Another 2732x2732 png titled `splash-dark.png` containing your splash screen in dark mode.

Then, run the following command to generate and place the assets in the appropriate places in your Android project:

```sh
npx capacitor-assets generate --android
```

### Publishing your app in the Google Play Store

[Follow these instructions to learn how to publish your app to the Google Play Store](https://developer.android.com/studio/publish)

## Available Configuration Options

| Option              | Description                                                                     | Possible Values          |
| ------------------- | ------------------------------------------------------------------------------- | ------------------------ |
| `--platform`        | **(Required)** Specifies the target platform for the build.                                    | `"ios"` or `"android"`   |
| `--entryUrl`     | **(Required)** The primary url of your website.                                             | Any valid url    |
| `--appId`           | The unique identifier for the app (e.g., iOS Bundle ID, Android Application ID). | A reverse domain name string (e.g., `com.company.appname`) |
| `--appName`         | The user-visible name of the application.                                       | Any valid application name string (e.g., "My Awesome App") |
| `--output`          | The directory where the generated app project files will be saved.              | A valid, absolute file path (e.g., `/users/me/my-generated-app`) |
| `--additionalDomains` | A list of other domains that should be accessible within the app.               | Comma-separated domains |
| `--smartDialerConfig` | A JSON string containing the configuration for the [smart dialer feature](../../smart#yaml-config-for-the-smart-dialer).       | Valid JSON string       |

## Viewing your site in the example navigation iframe

Many sites don't handle their own navigation - if this applies to you, you can run a proxy to demonstrate what your site would look like in an example same-origin navigation iframe.

* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`--navigatorToken`](https://dashboard.ngrok.com/get-started/your-authtoken)

```sh
npm run start:navigator -- --entryUrl="https://www.example.com" \
  --navigatorToken="<YOUR_NGROK_AUTH_TOKEN>" --navigatorPath="/nav"
```

Once the server has started, you can then run the build commands above in a separate terminal to view the demo in your app.

## Available Configuration Options

| Option              | Description                                                                     | Possible Values          |
| ------------------- | ------------------------------------------------------------------------------- | ------------------------ |
| `--entryUrl`     | **(Required)** The primary url of your website.                                             | Any valid url    |
| `--smartDialerConfig` | A JSON string containing the configuration for the [smart dialer feature](../../smart#yaml-config-for-the-smart-dialer).       | Valid JSON string       |
| `--navigatorToken`  | Your ngrok authentication token for using the navigation proxy.                 | Your [ngrok auth token](https://dashboard.ngrok.com/get-started/your-authtoken)    |
| `--navigatorPath`   | The path to use for the navigation iframe when using the navigation proxy. | Any valid path           |

## Troubleshooting

When encountering an issue, the first thing you'll want to do is run the doctor script to see if your system has all the required dependencies:

```sh
./doctor
```

Additionally, you should run `npm run reset` to ensure your `output` and `node_modules` folders have not been tampered with!

### Commonly occuring issues

> [!NOTE]
> TODO: compile a list of commonly occuring issues.

