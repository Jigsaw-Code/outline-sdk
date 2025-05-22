package mobileproxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewStreamDialerFromConfig_Valid(t *testing.T) {
	dialer, err := NewStreamDialerFromConfig("direct://")
	if err != nil {
		t.Errorf("NewStreamDialerFromConfig with valid config string failed: %v", err)
	}
	if dialer == nil {
		t.Error("NewStreamDialerFromConfig with valid config string returned a nil dialer")
	}
}

func TestNewStreamDialerFromConfig_Invalid(t *testing.T) {
	_, err := NewStreamDialerFromConfig("invalid://")
	if err == nil {
		t.Error("NewStreamDialerFromConfig with invalid config string did not return an error")
	}
}

func TestRunProxy_StartStop(t *testing.T) {
	dialer, err := NewStreamDialerFromConfig("direct://")
	if err != nil {
		t.Fatalf("NewStreamDialerFromConfig failed: %v", err)
	}

	proxy, err := RunProxy("localhost:0", dialer)
	if err != nil {
		t.Fatalf("RunProxy failed: %v", err)
	}

	if proxy.Address() == "" {
		t.Error("proxy.Address() returned empty string")
	}
	if proxy.Host() == "" {
		t.Error("proxy.Host() returned empty string")
	}
	if proxy.Port() == 0 {
		t.Error("proxy.Port() returned 0")
	}

	proxy.Stop(1)

	// Attempt to connect to the proxy, it should fail
	// This is a simple way to check; more robust checks might involve http.Get
	// For now, we assume that if Stop() completed, the port is released or will be soon.
	// A more direct test would be to try to establish a new listener on the same port.
}

func TestRunProxy_NilDialer(t *testing.T) {
	_, err := RunProxy("localhost:0", nil)
	if err == nil {
		t.Error("RunProxy with nil dialer did not return an error")
	}
	if !strings.Contains(err.Error(), "dialer cannot be nil") {
		t.Errorf("RunProxy with nil dialer returned unexpected error: %v", err)
	}
}

func TestProxy_AddURLProxy(t *testing.T) {
	dialer, err := NewStreamDialerFromConfig("direct://")
	if err != nil {
		t.Fatalf("NewStreamDialerFromConfig failed: %v", err)
	}

	proxy, err := RunProxy("localhost:0", dialer)
	if err != nil {
		t.Fatalf("RunProxy failed: %v", err)
	}
	defer proxy.Stop(1)

	// Create a dummy target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	// The dialer for AddURLProxy needs to point to our test server
	urlProxyDialer, err := NewStreamDialerFromConfig("direct://" + strings.TrimPrefix(targetServer.URL, "http://"))
	if err != nil {
		t.Fatalf("NewStreamDialerFromConfig for target server failed: %v", err)
	}

	err = proxy.AddURLProxy("/test", urlProxyDialer)
	if err != nil {
		t.Fatalf("AddURLProxy failed: %v", err)
	}

	// Test that the proxy forwards to the target server
	// To do this, we'll make a request through the proxy to the /test path
	// The proxy itself doesn't directly expose its http.Handler, so we make an HTTP request.

	req, err := http.NewRequest("GET", "http://"+proxy.Address()+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create a transport that uses the proxy's dialer (this is a bit meta,
	// normally you'd use a system configured proxy or http.ProxyURL)
	// However, for this test, we want to ensure our proxy instance handles it.
	// A simpler way for this specific test: make a request to the proxy's address.

	client := &http.Client{} // No special transport needed, just hit the proxy's address
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request through proxy to /test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for /test, got %s", resp.Status)
	}

	// Test a path that shouldn't be handled by the URLProxy
	reqNotFound, err := http.NewRequest("GET", "http://"+proxy.Address()+"/notfound", nil)
	if err != nil {
		t.Fatalf("Failed to create request for /notfound: %v", err)
	}
	respNotFound, err := client.Do(reqNotFound)
	if err != nil {
		// This depends on how the underlying proxy handles unmapped URLs.
		// goproxy by default returns 503 if it cannot connect to an upstream proxy
		// or if the dialer fails. If it's a direct dial and the target doesn't exist,
		// it could also be a connection error at the client.Do level.
		// For this test, we expect the default goproxy behavior for unhandled paths.
		// It might try to dial directly. Let's assume it will result in a non-200 status.
		// A more robust test would involve ensuring it *doesn't* hit `targetServer`.
		t.Logf("Request to /notfound through proxy failed as expected (or handled by default proxy logic): %v", err)
	} else {
		defer respNotFound.Body.Close()
		if respNotFound.StatusCode == http.StatusOK {
			t.Errorf("Request to /notfound was unexpectedly successful with status OK")
		} else {
			t.Logf("Request to /notfound got status: %s", respNotFound.Status)
		}
	}
}

func TestStringList(t *testing.T) {
	// Test NewListFromLines
	lines := "line1\nline2\nline3"
	list := NewListFromLines(lines)
	if len(list.list) != 3 {
		t.Errorf("NewListFromLines: expected 3 items, got %d", len(list.list))
	}
	if list.list[0] != "line1" || list.list[1] != "line2" || list.list[2] != "line3" {
		t.Errorf("NewListFromLines: content mismatch, got %v", list.list)
	}

	// Test Append
	list.Append("line4")
	if len(list.list) != 4 {
		t.Errorf("Append: expected 4 items, got %d", len(list.list))
	}
	if list.list[3] != "line4" {
		t.Errorf("Append: new item not added correctly, got %v", list.list)
	}

	// Test Append with empty string
	list.Append("")
	if len(list.list) != 5 {
		t.Errorf("Append empty string: expected 5 items, got %d", len(list.list))
	}
	if list.list[4] != "" {
		t.Errorf("Append empty string: item not added correctly, got %v", list.list)
	}

	// Test NewListFromLines with empty input
	emptyList := NewListFromLines("")
	// Expecting one empty string if input is not empty but contains no newlines,
	// or zero items if input is truly empty.
	// Based on strings.Split behavior, an empty string results in a slice with one empty string.
	// If we want truly empty for empty input, NewListFromLines should handle it.
	// Let's assume current strings.Split behavior is acceptable for now.
	if len(emptyList.list) != 1 || emptyList.list[0] != "" {
		 // If the desired behavior is an empty list for an empty string input, this check needs adjustment.
		 // For now, sticking to `strings.Split` direct behavior.
		t.Errorf("NewListFromLines with empty string: expected 1 empty item, got %v", emptyList.list)
	}

	// Test NewListFromLines with input ending with newline
	linesWithTrailingNewline := "lineA\nlineB\n"
	listTrailing := NewListFromLines(linesWithTrailingNewline)
	// strings.Split will produce an empty string at the end if the string ends with a separator.
	if len(listTrailing.list) != 3 {
		t.Errorf("NewListFromLines with trailing newline: expected 3 items, got %d. Items: %v", len(listTrailing.list), listTrailing.list)
	}
	if listTrailing.list[0] != "lineA" || listTrailing.list[1] != "lineB" || listTrailing.list[2] != "" {
		t.Errorf("NewListFromLines with trailing newline: content mismatch, got %v", listTrailing.list)
	}
}
