// Copyright 2023 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package report

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ConnectivitySetup struct {
	Proxy    string `json:"proxy"`
	Resolver string `json:"resolver"`
	Proto    string `json:"proto"`
	Prefix   string `json:"prefix"`
}

type ConnectivityError struct {
	Op         string `json:"operation"`
	PosixError string `json:"posixError"`
	Msg        string `json:"msg"`
}

// ConnectivityReport implements the [HasSuccess] interface.
func (r ConnectivityReport) IsSuccess() bool {
	if r.Error == nil {
		return true
	} else {
		return false
	}
}

// ConnectivityReport represents a report containing connectivity information.
type ConnectivityReport struct {
	// Connection setup
	Connection interface{} `json:"connection"`
	// Observations
	Time       time.Time `json:"time"`
	DurationMs int64     `json:"durationMs"`
	// Connectivity error, if any
	Error interface{} `json:"error"`
}

func TestIsSuccess(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}

	var r Report = testReport
	v, ok := r.(HasSuccess)
	if !ok {
		t.Error("Report is expected to implement HasSuccess interface, but it does not")
	}
	// since report does not have Error field, it should be successful
	require.True(t, v.IsSuccess())
}

func TestSendReportSuccessfully(t *testing.T) {
	var requestBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		requestBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Fprintln(w, "OK")
	}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	var testSetup = ConnectivitySetup{
		Proxy:    "testProxy",
		Resolver: "8.8.8.8",
		Proto:    "udp",
		Prefix:   "HTTP1/1",
	}
	var testErr = ConnectivityError{
		Op:         "read",
		PosixError: "ETIMEDOUT",
		Msg:        "i/o timeout",
	}
	var testReport = ConnectivityReport{
		Connection: testSetup,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
		Error:      testErr,
	}

	c := RemoteCollector{
		CollectorURL: u,
		HttpClient:   &http.Client{Timeout: 10 * time.Second},
	}
	err = c.Collect(context.Background(), testReport)
	require.NoError(t, err)

	expected, err := json.Marshal(testReport)
	require.NoError(t, err)
	require.Equal(t, string(expected), string(requestBody))
}

func TestSendReportUnsuccessfully(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	c := RemoteCollector{
		CollectorURL: u,
		HttpClient:   &http.Client{Timeout: 10 * time.Second},
	}

	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}

	err = c.Collect(context.Background(), testReport)
	if err == nil {
		t.Errorf("Expected 405 error no error occurred!")
	} else {
		var e *BadRequestError
		require.ErrorAs(t, err, &e)
	}
}

func TestSamplingCollector(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	c := SamplingCollector{
		Collector: &RemoteCollector{
			CollectorURL: u,
			HttpClient:   &http.Client{Timeout: 10 * time.Second},
		},
		SuccessFraction: 0.5,
		FailureFraction: 0.1,
	}

	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}

	err = c.Collect(context.Background(), testReport)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestSendJSONToServer(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}

	// Start a local HTTP server for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var receivedReport ConnectivityReport
		err := json.Unmarshal(body, &receivedReport)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}

		// Asserting that the received JSON matches the expected JSON
		if testReport != receivedReport {
			t.Errorf("Expected %v, got %v", testReport, receivedReport)
		}
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	var r Report = testReport
	c := RemoteCollector{
		CollectorURL: u,
		HttpClient:   &http.Client{Timeout: 10 * time.Second},
	}
	err = c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestFallbackCollector(t *testing.T) {
	// Create test server that always fails
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer ts1.Close()
	u1, err := url.Parse(ts1.URL)
	require.NoError(t, err)

	// Create test server that always succeeds.
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}))
	defer ts2.Close()
	u2, err := url.Parse(ts2.URL)
	require.NoError(t, err)

	c := FallbackCollector{
		Collectors: []Collector{
			&RemoteCollector{
				CollectorURL: u1,
				HttpClient:   &http.Client{Timeout: 10 * time.Second},
			},
			&RemoteCollector{
				CollectorURL: u2,
				HttpClient:   &http.Client{Timeout: 10 * time.Second},
			},
		},
	}

	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var r Report = testReport
	err = c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestRetryCollector(t *testing.T) {
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount <= 2 {
			w.WriteHeader(500)
		} else {
			fmt.Fprintln(w, "OK")
		}
		requestCount++
	}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	c := RetryCollector{
		Collector: &RemoteCollector{
			CollectorURL: u,
			HttpClient:   &http.Client{Timeout: 10 * time.Second},
		},
		MaxRetry:     3,
		InitialDelay: 1 * time.Second,
	}

	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}

	err = c.Collect(context.Background(), testReport)
	require.NoError(t, err)
}

func TestWriteCollector(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var r Report = testReport
	v, ok := r.(HasSuccess)
	if ok {
		t.Logf("The test report shows success: %v\n", v.IsSuccess())
	}
	c := WriteCollector{
		Writer: io.Discard,
	}
	err := c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

// TestWriteCollectorToFile that opens a file and collects to a temp file
func TestWriteCollectorToFile(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var r Report = testReport
	v, ok := r.(HasSuccess)
	if ok {
		t.Logf("The test report shows success: %v\n", v.IsSuccess())
	}
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	defer os.Remove(f.Name()) // clean up
	c := WriteCollector{
		Writer: f,
	}
	err = c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
