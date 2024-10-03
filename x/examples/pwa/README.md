# [WIP] Outline PWA

> [!CAUTION]
> This is currently a WIP placeholder for the Outline PWA project. It has no actual logic in it yet.

Outline PWA will be an sdk example project dedicated to demonstrating how one can use CapacitorJS to wrap an existing website to make it censorship resistant. To load the site you want to test into the project, cd into this directory and run the following command:

```sh
npm run setup https://www.bbc.com/persian
```

This will update the ios and android subprojects with that site's index.html. You will then be able to open those projects in Xcode and Android Studio to see how effectively the entire site loads.

## Strategies

We'll be defining our own custom scheme - `outline://` - and loading the index.html via `outline://index.html`. Subresources that don't have their scheme defined should load via `outline://` as well.

### iOS

TODO - define a `WKURLSchemeHandler` for `outline://` and handle the traffic that way.

### Android

TODO - create a new `WebViewClient` with a `shouldOverrideUrlLoading` that handles the `outline://` scheme. Set that web client in the `BridgeActivity` that Capacitor provides.
