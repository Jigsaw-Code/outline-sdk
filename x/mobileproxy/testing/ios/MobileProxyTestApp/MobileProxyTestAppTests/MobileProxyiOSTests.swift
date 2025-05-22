import XCTest
// This import assumes that a framework named "Mobileproxy" will be linked to this test target.
// In a real Xcode project, you would add the Mobileproxy.xcframework to the "Frameworks,
// Libraries, and Embedded Content" section of your test target's settings.
import Mobileproxy

class MobileProxyiOSTests: XCTestCase {

    // A public URL for testing network requests
    let testURL = URL(string: "http://example.com")!

    override func setUpWithError() throws {
        super.setUpWithError()
        // Put setup code here. This method is called before the invocation of each test method in the class.
        // Initialize a log writer for mobileproxy. This can help debug issues by printing logs.
        // Using StderrLogWriter, which should output to Xcode's console or system logs.
        // This is a global operation, so it's fine to do it once if desired, or per test.
        do {
            try MobileproxyNewStderrLogWriter()
            print("Mobileproxy StderrLogWriter initialized.")
        } catch {
            XCTFail("Failed to initialize Mobileproxy StderrLogWriter: \(error)")
        }
    }

    override func tearDownWithError() throws {
        // Put teardown code here. This method is called after the invocation of each test method in the class.
        super.tearDownWithError()
    }

    func testRunProxy_Success() throws {
        // 1. Create a new stream dialer
        var dialerError: NSError?
        let dialer = MobileproxyNewStreamDialerFromConfig("direct://", &dialerError)
        XCTAssertNil(dialerError, "NewStreamDialerFromConfig with 'direct://' should not produce an error. Error: \(dialerError?.localizedDescription ?? "nil")")
        XCTAssertNotNil(dialer, "Dialer should not be nil for a valid config 'direct://'")

        guard let validDialer = dialer else {
            XCTFail("Dialer was nil, cannot proceed with the test.")
            return
        }

        // 2. Run the proxy on a local address with a dynamic port (0)
        var proxyError: NSError?
        let proxy = MobileproxyRunProxy("127.0.0.1:0", validDialer, &proxyError)
        XCTAssertNil(proxyError, "RunProxy should not produce an error with a valid dialer. Error: \(proxyError?.localizedDescription ?? "nil")")
        XCTAssertNotNil(proxy, "Proxy object should not be nil when run with a valid dialer")

        guard let runningProxy = proxy else {
            XCTFail("Proxy was nil, cannot proceed with the test.")
            return
        }

        // 3. Get proxy address details
        let proxyHost = runningProxy.host()
        let proxyPort = runningProxy.port() // This returns Int32 for port

        XCTAssertEqual(proxyHost, "127.0.0.1", "Proxy host should be 127.0.0.1")
        XCTAssertGreaterThan(proxyPort, 0, "Proxy port should be greater than 0")
        print("Mobileproxy started on \(proxyHost ?? "nil"):\(proxyPort)")

        // 4. Perform an HTTP GET request through the running proxy
        let expectationGetSuccess = XCTestExpectation(description: "HTTP GET request through proxy succeeds")
        
        let config = URLSessionConfiguration.default
        // Configure the session to use the proxy
        config.connectionProxyDictionary = [
            kCFNetworkProxiesHTTPEnable: true,
            kCFNetworkProxiesHTTPProxy: proxyHost ?? "127.0.0.1", // Use a default if host is nil, though it shouldn't be
            kCFNetworkProxiesHTTPPort: NSNumber(value: proxyPort)
        ]
        // For HTTPS testing, you'd also need:
        // kCFNetworkProxiesHTTPSEnable: true,
        // kCFNetworkProxiesHTTPSProxy: proxyHost ?? "127.0.0.1",
        // kCFNetworkProxiesHTTPSPort: NSNumber(value: proxyPort)

        let session = URLSession(configuration: config)
        let task = session.dataTask(with: testURL) { data, response, error in
            XCTAssertNil(error, "HTTP GET request error should be nil. Error: \(error?.localizedDescription ?? "nil")")
            if let httpResponse = response as? HTTPURLResponse {
                XCTAssertEqual(httpResponse.statusCode, 200, "HTTP GET request should return status 200 OK. Status: \(httpResponse.statusCode)")
            } else {
                XCTFail("Response was not an HTTPURLResponse.")
            }
            XCTAssertNotNil(data, "Data should not be nil")
            expectationGetSuccess.fulfill()
        }
        task.resume()
        wait(for: [expectationGetSuccess], timeout: 10.0) // 10-second timeout

        // 5. Stop the proxy
        print("Stopping proxy...")
        // The stop method in Go might take some time if there are active connections.
        // The integer argument is a timeout in seconds for graceful shutdown (implementation specific).
        runningProxy.stop(1) 
        print("Proxy stop method called.")

        // Allow a brief moment for the proxy to fully shut down its listener.
        // This can make the subsequent failure assertion more reliable.
        Thread.sleep(forTimeInterval: 0.5)


        // 6. Attempt another HTTP GET request through the stopped proxy (should fail)
        let expectationGetFailure = XCTestExpectation(description: "HTTP GET request through stopped proxy fails")
        let sessionAfterStop = URLSession(configuration: config) // Re-use the same proxy config
        
        let taskFailure = sessionAfterStop.dataTask(with: testURL) { data, response, error in
            XCTAssertNotNil(error, "Error should not be nil after proxy is stopped.")
            if let nsError = error as NSError? {
                // Common error codes for connection failure:
                // NSURLErrorCannotConnectToHost, NSURLErrorNetworkConnectionLost, NSURLErrorTimedOut
                XCTAssertTrue([NSURLErrorCannotConnectToHost, NSURLErrorNetworkConnectionLost, NSURLErrorTimedOut].contains(nsError.code),
                              "Error code should indicate a connection failure. Actual code: \(nsError.code), domain: \(nsError.domain)")
                print("Request through stopped proxy failed as expected with error: \(nsError.localizedDescription)")
            } else {
                XCTFail("Error was not an NSError, or was nil unexpectedly.")
            }
            XCTAssertNil(response, "Response should be nil when request fails due to stopped proxy.")
            XCTAssertNil(data, "Data should be nil when request fails due to stopped proxy.")
            expectationGetFailure.fulfill()
        }
        taskFailure.resume()
        wait(for: [expectationGetFailure], timeout: 10.0)
        print("testRunProxy_Success completed.")
    }

    func testRunProxy_NilDialer() {
        // Attempt to call MobileproxyRunProxy with a nil dialer
        // The Swift binding for gomobile typically converts a Go error return into a thrown Swift error.
        // However, the `Mobileproxy*` functions provided in the prompt description use explicit NSError** arguments.
        
        var error: NSError?
        let proxy = MobileproxyRunProxy("127.0.0.1:0", nil, &error) // Pass nil for the dialer

        // Assert that an error was returned and the proxy object is nil
        XCTAssertNotNil(error, "MobileproxyRunProxy with a nil dialer should return an error.")
        XCTAssertNil(proxy, "MobileproxyRunProxy with a nil dialer should return a nil proxy object.")
        
        // Optionally, check the error domain and code if they are known and consistent
        if let returnedError = error {
            print("testRunProxy_NilDialer received error as expected: \(returnedError.localizedDescription)")
            // Example: XCTAssertEqual(returnedError.domain, "com.example.mobileproxy.error", "Error domain mismatch")
            // XCTAssertEqual(returnedError.code, EXPECTED_NIL_DIALER_ERROR_CODE, "Error code mismatch")
            // For Go-generated errors, the message often contains the Go error string.
            XCTAssertTrue(returnedError.localizedDescription.contains("dialer cannot be nil"), "Error message should indicate nil dialer. Message: \(returnedError.localizedDescription)")
        }
        print("testRunProxy_NilDialer completed.")
    }

    func testNewStreamDialerFromConfig_Invalid() {
        // Attempt to create a stream dialer with an invalid configuration string
        var error: NSError?
        let dialer = MobileproxyNewStreamDialerFromConfig("invalid_config_!@#", &error)

        // Assert that an error was returned and the dialer object is nil
        XCTAssertNotNil(error, "NewStreamDialerFromConfig with invalid config should return an error.")
        XCTAssertNil(dialer, "NewStreamDialerFromConfig with invalid config should return a nil dialer object.")

        if let returnedError = error {
            print("testNewStreamDialerFromConfig_Invalid received error as expected: \(returnedError.localizedDescription)")
            // The error message from Go's `parseNetAddr` might be something like "unknown scheme" or "invalid format"
            // This depends on the Go implementation of NewStreamDialerFromConfig.
             XCTAssertTrue(returnedError.localizedDescription.contains("unknown scheme") || returnedError.localizedDescription.contains("invalid format"),
                           "Error message should indicate invalid config. Message: \(returnedError.localizedDescription)")
        }
        print("testNewStreamDialerFromConfig_Invalid completed.")
    }
}
