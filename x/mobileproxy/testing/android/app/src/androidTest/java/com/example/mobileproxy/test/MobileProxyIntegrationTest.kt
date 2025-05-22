package com.example.mobileproxy.test

import androidx.test.ext.junit.runners.AndroidJUnit4
import mobileproxy.Mobileproxy // Assuming this is the correct import for your AAR's classes
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import java.io.BufferedReader
import java.io.IOException
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.InetSocketAddress
import java.net.Proxy
import java.net.URL
import kotlin.test.assertFailsWith

/**
 * Instrumentation tests for the Mobileproxy library.
 * These tests run on an Android device or emulator.
 */
@RunWith(AndroidJUnit4::class)
class MobileProxyIntegrationTest {

    // Test target URL
    private val testUrl = "http://example.com" // Using HTTP for simplicity with HttpURLConnection

    @Test
    fun testRunProxy_Success() {
        // Initialize logger (optional, but good for debugging)
        try {
            Mobileproxy.newStderrLogWriter() // This sends gomobile's stderr to logcat
        } catch (e: Exception) {
            // Log or handle if logger init fails, though it's often non-critical for the test itself
            println("Failed to initialize Mobileproxy.newStderrLogWriter: ${e.message}")
        }

        // 1. Create a new stream dialer
        val dialer = Mobileproxy.newStreamDialerFromConfig("direct://")
        assertNotNull("Dialer should not be null for valid config 'direct://'", dialer)

        // 2. Run the proxy on a local address with a dynamic port (0)
        val proxy = Mobileproxy.runProxy("127.0.0.1:0", dialer)
        assertNotNull("Proxy object should not be null when run with a valid dialer", proxy)

        // 3. Get proxy address details
        val proxyAddress = proxy.address()
        val proxyHost = proxy.host()
        val proxyPort = proxy.port()

        assertNotNull("Proxy address string should not be null", proxyAddress)
        assertEquals("Proxy host should be 127.0.0.1", "127.0.0.1", proxyHost)
        assertTrue("Proxy port should be greater than 0", proxyPort > 0)
        println("Proxy listening on $proxyHost:$proxyPort")

        // 4. Perform an HTTP GET request through the running proxy
        var connection: HttpURLConnection? = null
        try {
            val url = URL(testUrl)
            // Configure HttpURLConnection to use the proxy
            val proxySocketAddress = InetSocketAddress(proxyHost, proxyPort.toInt())
            val httpProxy = Proxy(Proxy.Type.HTTP, proxySocketAddress)
            connection = url.openConnection(httpProxy) as HttpURLConnection
            connection.requestMethod = "GET"
            connection.connectTimeout = 10000 // 10 seconds
            connection.readTimeout = 10000    // 10 seconds

            val responseCode = connection.responseCode
            assertEquals("HTTP GET request through proxy should return 200 OK", HttpURLConnection.HTTP_OK, responseCode)

            // Optionally, read the response body to ensure it's complete (example.com is small)
            val reader = BufferedReader(InputStreamReader(connection.inputStream))
            val responseBody = reader.readText()
            assertTrue("Response body should not be empty", responseBody.isNotEmpty())
            assertTrue("Response body should contain 'Example Domain'", responseBody.contains("Example Domain"))
            reader.close()
            println("Successfully fetched $testUrl through proxy.")

        } catch (e: Exception) {
            e.printStackTrace()
            fail("HTTP GET request through proxy failed: ${e.message}")
        } finally {
            connection?.disconnect()
        }

        // 5. Stop the proxy
        proxy.stop(1) // Stop with a timeout (e.g., 1 second)
        println("Proxy stopped.")

        // 6. Attempt another HTTP GET request through the stopped proxy (should fail)
        var connectionAfterStop: HttpURLConnection? = null
        var exceptionThrown = false
        try {
            val url = URL(testUrl)
            val proxySocketAddress = InetSocketAddress(proxyHost, proxyPort.toInt())
            val httpProxy = Proxy(Proxy.Type.HTTP, proxySocketAddress)
            connectionAfterStop = url.openConnection(httpProxy) as HttpURLConnection
            connectionAfterStop.requestMethod = "GET"
            connectionAfterStop.connectTimeout = 5000 // Shorter timeout for expected failure
            connectionAfterStop.readTimeout = 5000

            // This connection attempt should fail
            connectionAfterStop.connect() // Explicit connect to trigger failure sooner
            val responseCode = connectionAfterStop.responseCode // Or this might throw
            fail("Request through stopped proxy should have failed, but got response code: $responseCode")

        } catch (e: IOException) {
            // Expected exception (e.g., ConnectException, SocketException)
            exceptionThrown = true
            println("Request through stopped proxy failed as expected: ${e.message}")
        } catch (e: Exception) {
            // Catch any other unexpected exceptions
            e.printStackTrace()
            fail("Request through stopped proxy failed with an unexpected exception: ${e.message}")
        } finally {
            connectionAfterStop?.disconnect()
        }
        assertTrue("An IOException should have been thrown when connecting through stopped proxy", exceptionThrown)
    }

    @Test
    fun testRunProxy_NilDialer() {
        // Attempt to run proxy with a nil dialer, expecting an exception
        val exception = assertFailsWith<Exception>("Expected an exception when running proxy with nil dialer") {
            Mobileproxy.runProxy("127.0.0.1:0", null)
        }
        // Optionally, assert on the exception message if it's consistent
        assertTrue("Exception message should indicate nil dialer error", exception.message?.contains("dialer cannot be nil") == true)
        println("testRunProxy_NilDialer completed successfully with exception: ${exception.message}")
    }

    @Test
    fun testNewStreamDialerFromConfig_Invalid() {
        // Attempt to create a dialer with an invalid config string, expecting an exception
        val invalidConfig = "invalid_config_!@#"
        val exception = assertFailsWith<Exception>("Expected an exception for invalid dialer config") {
            Mobileproxy.newStreamDialerFromConfig(invalidConfig)
        }
        // Optionally, assert on the exception message
        // e.g., assertTrue(exception.message?.contains("unknown scheme") == true)
        println("testNewStreamDialerFromConfig_Invalid completed successfully with exception: ${exception.message}")
    }
}
