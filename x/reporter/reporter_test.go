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
	"encoding/json"
	"fmt"
	"testing"
)

// When success is true and random number is less than successSampleRate, report is sent successfully
func TestSendReportSuccessfully(t *testing.T) {
	// Example JSON data
	jsonData := `{
		"proxy": "192.168.1.1:65000",
		"transport": "shadowsocks",
		"resolver": "8.8.8.8:53",
		"proto": "tcp",
		"prefix": "HTTP1/1",
		"time": "2021-01-01T00:00:00Z",
		"durationMs": 100,
		"error": {
			"operation": "read",
			"posixError": "ETIMEDOUT",
			"msg": "i/o timeout"
		}
	}`
	var testReport Report
	err := json.Unmarshal([]byte(jsonData), &testReport.logRecord)
	if err != nil {
		fmt.Println(err)
		t.Errorf("Expected no error, but got: %v", err)
	}
	testReport.config.reportTo = "https://script.google.com/macros/s/AKfycbzoMBmftQaR9Aw4jzTB-w4TwkDjLHtSfBCFhh4_2NhTEZAUdj85Qt8uYCKCNOEAwCg4/exec"
	testReport.success = true
	testReport.config.successFraction = 1.0
	testReport.config.failureFraction = 0.0

	err = testReport.Collect()
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}
