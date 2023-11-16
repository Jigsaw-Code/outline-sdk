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

package reporter

import (
	"fmt"
	"net/url"
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

func TestIsSuccess(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}

	var r Report = testReport
	if !r.IsSuccess() {
		t.Errorf("Expected false, but got: %v", r.IsSuccess())
	} else {
		fmt.Println("IsSuccess Test Passed")
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
	fmt.Printf("The test report shows success: %v\n", r.IsSuccess())
	u, err := url.Parse("https://script.google.com/macros/s/AKfycbzoMBmftQaR9Aw4jzTB-w4TwkDjLHtSfBCFhh4_2NhTEZAUdj85Qt8uYCKCNOEAwCg4/exec")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := RemoteCollector{
		collectorEndpoint: u,
	}
	err = c.Collect(r)
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
	var r Report = testReport
	fmt.Printf("The test report shows success: %v\n", r.IsSuccess())
	u, err := url.Parse("https://google.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := RemoteCollector{
		collectorEndpoint: u,
	}
	err = c.Collect(r)
	if err == nil {
		t.Errorf("Expected 405 error no error occurred!")
	} else {
		if err, ok := err.(StatusErr); ok {
			if err.StatusCode != 405 {
				t.Errorf("Expected 405 error no error occurred!")
			}
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
	fmt.Printf("The test report shows success: %v\n", r.IsSuccess())
	u, err := url.Parse("https://example.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := SamplingCollector{
		collector: &RemoteCollector{
			collectorEndpoint: u,
		},
		successFraction: 0.5,
		failureFraction: 0.1,
	}
	err = c.Collect(r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestRotatingCollector(t *testing.T) {
	var testReport = ConnectivityReport{
		Connection: nil,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
	}
	var r Report = testReport
	fmt.Printf("The test report shows success: %v\n", r.IsSuccess())
	u1, err := url.Parse("https://example.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	u2, err := url.Parse("https://google.com")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	u3, err := url.Parse("https://script.google.com/macros/s/AKfycbzoMBmftQaR9Aw4jzTB-w4TwkDjLHtSfBCFhh4_2NhTEZAUdj85Qt8uYCKCNOEAwCg4/exec")
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	c := RotatingCollector{
		collectors: []CollectorTarget{
			{collector: &RemoteCollector{
				collectorEndpoint: u1,
			},
				priority: 1,
				maxRetry: 2,
			},
			{
				collector: &RemoteCollector{
					collectorEndpoint: u2,
				},
				priority: 2,
				maxRetry: 3,
			},
			{
				collector: &RemoteCollector{
					collectorEndpoint: u3,
				},
				priority: 3,
				maxRetry: 2,
			},
		},
		stopOnSuccess: false,
	}
	err = c.Collect(r)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
