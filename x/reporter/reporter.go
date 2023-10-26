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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)
var httpClient = &http.Client{}

type Record struct {
	collectorURL      string
	successSampleRate float64
	failureSampleRate float64
	success           bool
	record            interface{}
}

func Report(r Record) error {

	// Perform custom range validation for sampling rate
	if r.successSampleRate < 0.0 || r.successSampleRate > 1.0 {
		return errors.New("Error: successSampleRate must be between 0 and 1.")
	}

	if r.failureSampleRate < 0.0 || r.failureSampleRate > 1.0 {
		return errors.New("Error: failureSampleRate must be between 0 and 1.")
	}

	var samplingRate float64
	if r.success {
		samplingRate = r.successSampleRate
	} else {
		samplingRate = r.failureSampleRate
	}
	// Generate a random number between 0 and 1
	random := rand.Float64()
	if random < samplingRate {
		err := sendReport(r.record, r.collectorURL)
		if err != nil {
			log.Printf("HTTP request failed: %v", err)
			return err
		} else {
			fmt.Println("Report sent")
			return nil
		}
	} else {
		fmt.Println("Report was not sent this time")
		return nil
	}
}

func sendReport(record interface{}, collectorURL string) error {
	jsonData, err := json.Marshal(record)
	if err != nil {
		log.Printf("Error encoding JSON: %s\n", err)
		return err
	}

	req, err := http.NewRequest("POST", collectorURL, bytes.NewReader(jsonData))
	if err != nil {
		debugLog.Printf("Error creating the HTTP request: %s\n", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error sending the HTTP request: %s\n", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLog.Printf("Error reading the HTTP response body: %s\n", err)
		return err
	}
	debugLog.Printf("Response: %s\n", respBody)
	return nil
}
