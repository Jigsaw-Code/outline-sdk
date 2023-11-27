// Copyright 2023 Jigsaw Operations LLC
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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
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
	if v.IsSuccess() {
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	} else {
		t.Errorf("Expected true, but got: %v", v.IsSuccess())
	}
}

func TestSendReportSuccessfully(t *testing.T) {
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

	var r Report = testReport
	v, ok := r.(HasSuccess)
	if ok {
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	}
	u, err := url.Parse("https://script.google.com/macros/s/AKfycbzoMBmftQaR9Aw4jzTB-w4TwkDjLHtSfBCFhh4_2NhTEZAUdj85Qt8uYCKCNOEAwCg4/exec")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := RemoteCollector{
		collectorEndpoint: u,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
	}
	err = c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestSendReportUnsuccessfully(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var e *BadRequestError
	var r Report = testReport
	v, ok := r.(HasSuccess)
	if ok {
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	}
	u, err := url.Parse("https://google.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := RemoteCollector{
		collectorEndpoint: u,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
	}
	err = c.Collect(context.Background(), r)
	if err == nil {
		t.Errorf("Expected 405 error no error occurred!")
	} else {
		if errors.As(err, &e) {
			fmt.Printf("Error was expected: %v\n", err)
		} else {
			t.Errorf("Expected 405 error, but got: %v", err)
		}
	}
}

func TestSamplingCollector(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var r Report = testReport
	v, ok := r.(HasSuccess)
	if ok {
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	}
	u, err := url.Parse("https://example.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := SamplingCollector{
		collector: &RemoteCollector{
			collectorEndpoint: u,
			httpClient:        &http.Client{Timeout: 10 * time.Second},
		},
		successFraction: 0.5,
		failureFraction: 0.1,
	}
	err = c.Collect(context.Background(), r)
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
		collectorEndpoint: u,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
	}
	err = c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestFallbackCollector(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var r Report = testReport
	v, ok := r.(HasSuccess)
	if ok {
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	}
	u1, err := url.Parse("https://example.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	u2, err := url.Parse("https://google.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := FallbackCollector{
		collectors: []Collector{
			&RemoteCollector{
				collectorEndpoint: u1,
				httpClient:        &http.Client{Timeout: 10 * time.Second},
			},
			&RemoteCollector{
				collectorEndpoint: u2,
				httpClient:        &http.Client{Timeout: 10 * time.Second},
			},
		},
	}
	err = c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestRetryCollector(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var r Report = testReport
	v, ok := r.(HasSuccess)
	if ok {
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	}
	u, err := url.Parse("https://google.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := RetryCollector{
		collector: &RemoteCollector{
			collectorEndpoint: u,
			httpClient:        &http.Client{Timeout: 10 * time.Second},
		},
		maxRetry:     3,
		initialDelay: 1 * time.Second,
	}
	err = c.Collect(context.Background(), r)
	if err == nil {
		t.Errorf("max retry error expcted not got none!")
	} else {
		fmt.Printf("Error was expected: %v\n", err)
	}
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
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	}
	c := WriteCollector{
		writer: io.Discard,
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
		fmt.Printf("The test report shows success: %v\n", v.IsSuccess())
	}
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	defer os.Remove(f.Name()) // clean up
	c := WriteCollector{
		writer: f,
	}
	err = c.Collect(context.Background(), r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
