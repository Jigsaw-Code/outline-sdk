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
mkcert -key-file dist/localhost-key.pem -cert-file dist/localhost.pem localhost 10.0.2.2

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
npx serve --ssl-cert dist/localhost.pem --ssl-key dist/localhost-key.pem www

# in a separate terminal, run the ios application
npx cap run ios
```

## Running the example on MacOS & Android

TODO