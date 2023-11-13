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
	"testing"
	"time"
)

// When success is true and random number is less than successSampleRate, report is sent successfully
func TestSendReportSuccessfully(t *testing.T) {
	// Example test data
	type ConnectivitySetup struct {
		Proxy    string `json:"proxy"`
		Resolver string `json:"resolver"`
		Proto    string `json:"proto"`
		Prefix   string `json:"prefix"`
	}
	var testSetup = ConnectivitySetup{
		Proxy:    "testProxy",
		Resolver: "8.8.8.8",
		Proto:    "udp",
		Prefix:   "HTTP1/1",
	}
	type ConnectivityError struct {
		Op         string `json:"operation"`
		PosixError string `json:"posixError"`
		Msg        string `json:"msg"`
	}
	var testErr = ConnectivityError{
		Op:         "read",
		PosixError: "ETIMEDOUT",
		Msg:        "i/o timeout",
	}
	var report = ConnectivityReport{
		Connection: testSetup,
		Time:       time.Now().UTC().Truncate(time.Second),
		DurationMs: 1,
		Error:      &testErr,
	}

	var c = Config{}
	err := c.SetFractions(1.0, 1.0)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	err = c.SetURL("https://example.com")

	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	err = report.Transmit(c)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestFromJSON(t *testing.T) {
	var testJSON = []byte(`{
		"connection": {
			"proxy": "testProxy",
			"resolver": "8.8.8.8",
			"proto": "udp",
			"prefix": "HTTP1/1"
		},
		"time": "2021-07-01T00:00:00Z",
		"durationMs": 1,
		"error": {
			"operation": "read",
			"posixError": "ETIMEDOUT",
			"msg": "i/o timeout"
		}
	}`)
	var report = ConnectivityReport{}
	type ConnectivityError struct {
		Op         string `json:"operation"`
		PosixError string `json:"posixError"`
		Msg        string `json:"msg"`
	}
	type ConnectivitySetup struct {
		Proxy    string `json:"proxy"`
		Resolver string `json:"resolver"`
		Proto    string `json:"proto"`
		Prefix   string `json:"prefix"`
	}
	report.Connection = &ConnectivitySetup{}
	report.Error = &ConnectivityError{}
	err := report.FromJSON(testJSON)
	fmt.Println(report.Error)
	fmt.Println(report)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	var c = Config{}
	err = c.SetFractions(1.0, 1.0)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	err = c.SetURL("https://example.com")

	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	err = report.Transmit(c)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
