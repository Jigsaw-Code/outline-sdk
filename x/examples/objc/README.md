


This works as macOS:

```console
$ GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go -C x run ./examples/objc
Attempting to get iOS process info using Cgo...

--- Successfully Retrieved Process Info ---
Process Name:           objc
Process ID (PID):       72134
User Name:              <redacted>
Full User Name:         <redacted>
Globally Unique ID:     <redacted>
OS Version:             Version 15.6.1 (Build 24G90)
Hostname:               <redacted>
Is Mac Catalyst App:    false
Is iOS App on Mac:      false
Physical Memory (B):    17178869184
System Uptime (s):      198843.16
Processor Count:        10
Active Processor Count: 10
```

This seems to properly build for iOS:

```console
% CC="$(xcrun --sdk iphoneos --find cc) -isysroot \"$(xcrun --sdk iphoneos --show-sdk-path)\"" GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go -C x build  -v ./examples/objc

github.com/Jigsaw-Code/outline-sdk/x/examples/objc
# github.com/Jigsaw-Code/outline-sdk/x/examples/objc
examples/objc/process_info.go:74:47: error: 'userName' is unavailable: not available on iOS
   74 |         p_info->userName = safe_strdup([[info userName] UTF8String]);
      |                                               ^
/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS18.4.sdk/System/Library/Frameworks/Foundation.framework/Headers/NSProcessInfo.h:192:38: note: property 'userName' is declared unavailable here
  192 | @property (readonly, copy) NSString *userName API_AVAILABLE(macosx(10.12)) API_UNAVAILABLE(ios, watchos, tvos);
      |                                      ^
/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS18.4.sdk/System/Library/Frameworks/Foundation.framework/Headers/NSProcessInfo.h:192:38: note: 'userName' has been explicitly marked unavailable here
examples/objc/process_info.go:75:51: error: 'fullUserName' is unavailable: not available on iOS
   75 |         p_info->fullUserName = safe_strdup([[info fullUserName] UTF8String]);
      |                                                   ^
/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS18.4.sdk/System/Library/Frameworks/Foundation.framework/Headers/NSProcessInfo.h:193:38: note: property 'fullUserName' is declared unavailable here
  193 | @property (readonly, copy) NSString *fullUserName API_AVAILABLE(macosx(10.12)) API_UNAVAILABLE(ios, watchos, tvos);
      |                                      ^
/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS18.4.sdk/System/Library/Frameworks/Foundation.framework/Headers/NSProcessInfo.h:193:38: note: 'fullUserName' has been explicitly marked unavailable here
2 errors generated.
```


For the simulator:

```console
% CC="$(xcrun --sdk iphonesimulator --find cc) -isysroot \"$(xcrun --sdk iphonesimulator --show-sdk-path)\"" GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go -C x build  -v ./examples/objc
```

If you build it for the iOS Simulator, you can run on the iOS simulator. It correctly returns the iOS version on the simulator (18.4), though notice that "Is iOS App on Mac" is false, because it's not "on Mac". 

```
% CC="$(xcrun --sdk iphonesimulator --find cc) -isysroot \"$(xcrun --sdk iphonesimulator --show-sdk-path)\"" GOOS=ios GOARCH=arm64 CGO_ENABLED=1 go -C x build  -v -o examples/objc/ProcessInfo.app ./examples/objc

% xcrun simctl boot 529EC4D4-FFC6-4249-A829-0D8181639E9D

% xcrun simctl install booted ./x/examples/objc/ProcessInfo.app

% xcrun simctl launch --console booted org.getoutline.test
org.getoutline.test: 41369
Attempting to get iOS process info using Cgo...

--- Successfully Retrieved Process Info ---
Process Name:           objc
Process ID (PID):       41369
User Name:              
Full User Name:         
Globally Unique ID:     8BA56801-5364-4D9C-A4F6-17F9681C1FF2-41369-00000900466BD6C9
OS Version:             Version 18.4 (Build 22E238)
Hostname:               fortuna-macbookpro2.roam.internal
Is Mac Catalyst App:    false
Is iOS App on Mac:      false
Physical Memory (B):    17179869184
System Uptime (s):      412366.44
Processor Count:        10
Active Processor Count: 10
-------------------------------------------
```
