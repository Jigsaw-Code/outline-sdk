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
	"time"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)
var httpClient = &http.Client{}

type ConnectivityReport struct {
	// Connection setup
	Connection interface{} `json:"connection"`
	// Observations
	Time       time.Time   `json:"time"`
	DurationMs int64       `json:"durationMs"`
	Error      interface{} `json:"error"`
}

type Configurer interface {
	SetFractions() error
	SetURL() error
	// SetMode() error
}

type Config struct {
	reportTo        string
	successFraction float64
	failureFraction float64
	// Other possible config fields
	// max_age      ints
	// max_retry	int
}

func (c *Config) SetFractions(success, failure float64) error {
	if success < 0 || success > 1 {
		return errors.New("success fraction must be between 0 and 1")
	}
	if failure < 0 || failure > 1 {
		return errors.New("failure fraction must be between 0 and 1")
	}
	c.successFraction = success
	c.failureFraction = failure
	return nil
}

func (c *Config) SetURL(url string) error {
	if url == "" {
		return errors.New("URL cannot be empty")
	}
	c.reportTo = url
	return nil
}

type Reporter interface {
	Transmit() error
	ToJSON() error
	// Configure() error
	FromJSON() error
	IsSuccessful() bool
}

func (r *ConnectivityReport) ToJSON() ([]byte, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		log.Printf("Error encoding JSON: %s\n", err)
		return nil, err
	}
	return jsonData, nil
}

func (r *ConnectivityReport) FromJSON(jsonData []byte) error {
	err := json.Unmarshal(jsonData, r)
	if err != nil {
		log.Printf("Error decoding JSON: %s\n", err)
		return err
	}
	return nil
}

func (r *ConnectivityReport) IsSuccess() bool {
	if r.Error == nil {
		return true
	} else {
		return false
	}
}

func (r *ConnectivityReport) Transmit(c Config) error {
	var samplingRate float64
	if r.IsSuccess() {
		samplingRate = c.successFraction
	} else {
		samplingRate = c.failureFraction
	}
	// Generate a random number between 0 and 1
	random := rand.Float64()
	if random < samplingRate {
		jsonData, err := r.ToJSON()
		if err != nil {
			log.Printf("Error encoding JSON: %s\n", err)
			return err
		}
		err = sendReport(jsonData, c.reportTo)
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

func sendReport(jsonData []byte, collectorURL string) error {

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
	// Access the HTTP response status code
	fmt.Printf("HTTP Response Status Code: %d\n", resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLog.Printf("Error reading the HTTP response body: %s\n", err)
		return err
	}
	debugLog.Printf("Response: %s\n", respBody)
	return nil
}
