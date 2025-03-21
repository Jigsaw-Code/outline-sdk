# Outline SDK Web Wrapper

## Building the app project template for **iOS**

* You will need your site's domain and a list of domains that you would also like to load in your app.
* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need [XCode](https://developer.apple.com/xcode/) and [cocoapods](https://cocoapods.org/). [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#ios-requirements)

```sh
npm run reset
npm run build -- --platform=ios --entryDomain="www.mysite.com" --additionalDomains="cdn.mysite.com,auth.mysite.com" --smartDialerConfig="<MY_SMART_DIALER_CONFIG_TEXT>"
```

XCode will automatically open the compiled project. Click the "play" button in XCode to start your iOS app! You can also 

### Viewing your site in the example navigation iframe

Many sites don't handle their own navigation - if this applies to you, you can run a proxy to demonstrate what your site would look like in an example same-origin navigation iframe.

* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`--proxyToken`](https://dashboard.ngrok.com/get-started/your-authtoken)

```sh
npm run start -- --platform=ios \
  --entryDomain="www.mysite.com" --additionalDomains="cdn.mysite.com,auth.mysite.com" \
  --proxyToken="<YOUR_NGROK_AUTH_TOKEN>" --navigationPath="/nav"
```

## Building the app project template for **Android**

* You will need your site's domain list of domains that you would also like to load in your app.
* You will need [Node.js](https://nodejs.org/en/) for the web server.
* You will need [OpenJDK 17](https://stackoverflow.com/a/70649641) and [Android Studio](https://developer.android.com/studio/) [Please follow CapacitorJS's environment setup guide](https://capacitorjs.com/docs/getting-started/environment-setup#android-requirements)

```sh
npm run clean # no need to do this on a fresh install
npm run build -- --platform=android --entryDomain="www.mysite.com" --additionalDomains="cdn.mysite.com,auth.mysite.com" --smartDialerConfig="<MY_SMART_DIALER_CONFIG_TEXT>"
```

Click the "play" button in Android Studio to start your Android app!

## Viewing your site in the example navigation iframe

Many sites don't handle their own navigation - if this applies to you, you can run a proxy to demonstrate what your site would look like in an example same-origin navigation iframe.

* You will need an [ngrok account](https://ngrok.com/), from which you can get your [`--proxyToken`](https://dashboard.ngrok.com/get-started/your-authtoken)

```sh
npm run start -- --platform=android \
  --entryDomain="www.mysite.com" --additionalDomains="cdn.mysite.com,auth.mysite.com" \
  --proxyToken="<YOUR_NGROK_AUTH_TOKEN>" --navigationPath="/nav"
```
