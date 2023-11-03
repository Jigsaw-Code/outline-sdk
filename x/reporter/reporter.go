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

type reporterConfig struct {
	reportTo        string
	successFraction float64
	failureFraction float64
	// Other possible config fields
	// max_age      int
	// max_retry	int
}

func (r *reporterConfig) SetFractions(success, failure float64) error {
	if success < 0 || success > 1 {
		return errors.New("success fraction must be between 0 and 1")
	}
	if failure < 0 || failure > 1 {
		return errors.New("failure fraction must be between 0 and 1")
	}
	r.successFraction = success
	r.failureFraction = failure
	return nil
}

type Report struct {
	logRecord any
	success   bool
	config    reporterConfig
}

type Reporter interface {
	Collect() error
}

func (r Report) Collect() error {
	var samplingRate float64
	if r.success {
		samplingRate = r.config.successFraction
	} else {
		samplingRate = r.config.failureFraction
	}
	// Generate a random number between 0 and 1
	random := rand.Float64()
	if random < samplingRate {
		err := sendReport(r.logRecord, r.config.reportTo)
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

func sendReport(logRecord any, collectorURL string) error {
	jsonData, err := json.Marshal(logRecord)
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
