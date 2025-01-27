# Outline SDK Web Wrapper Example

This example demonstrates how to use the Outline SDK to create a censorship-resistant mobile app by wrapping an existing website. 

> [!NOTE]
> To turn your own website into a censorship-resistant app from scratch, please follow one of the following guides:
> - [iOS](docs/ios.md)
> - [Android](docs/android.md)

## Running the example on MacOS & iOS

```sh
# install dependencies and setup project
npm ci
npx cap sync

# the demo site requires SSL for the app to load properly
brew install mkcert

# make sure your JAVA_HOME is setup before generating the SSL certificate
mkcert -install
mkdir dist
mkcert -key-file dist/localhost-key.pem -cert-file dist/localhost.pem localhost

# run the demo site
npx serve --ssl-cert dist/localhost.pem --ssl-key dist/localhost-key.pem www

# open the iOS project
npx cap open ios

# open the finder window containing the root CA
open "$(mkcert -CAROOT)"
```

Start the app and drag the root CA that mkcert generated into the simulator...  (INSTRUCTIONS IN PROGRESS)