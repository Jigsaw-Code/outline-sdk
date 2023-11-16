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
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
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

type Report interface {
	IsSuccess() bool
	CanMarshal() bool
}

// ConnectivityReport implements the Report interface
func (r ConnectivityReport) IsSuccess() bool {
	if r.Error == nil {
		return true
	} else {
		return false
	}
}

// Makes sure the Report type can be marshalled into JSON
func (r ConnectivityReport) CanMarshal() bool {
	_, err := json.Marshal(r)
	if err != nil {
		log.Printf("Error encoding JSON: %s\n", err)
		return false
	} else {
		return true
	}
}

type Collector interface {
	Collect(Report) error
}

type RemoteCollector struct {
	collectorEndpoint *url.URL
}

type SamplingCollector struct {
	collector       Collector
	successFraction float64
	failureFraction float64
}

type CollectorTarget struct {
	collector Collector
	priority  int
	maxRetry  int
}

// TODO: implement a rotating collector
type RotatingCollector struct {
	collectors    []CollectorTarget
	stopOnSuccess bool
}

// SortByPriority sorts the collectors based on their priority in ascending order
func (rc *RotatingCollector) SortByPriority() {
	sort.Slice(rc.collectors, func(i, j int) bool {
		return rc.collectors[i].priority < rc.collectors[j].priority
	})
}

func (c *RotatingCollector) Collect(report Report) error {
	// sort collectors in RotatingCollector by priority
	// into a new slice
	c.SortByPriority()
	for _, target := range c.collectors {
		for i := 0; i < target.maxRetry; i++ {
			err := target.collector.Collect(report)
			if err != nil {
				if err, ok := err.(StatusErr); ok {
					switch {
					case err.StatusCode == 500:
						// skip retrying
						break
					case err.StatusCode == 408:
						// wait for 1 second before retry
						time.Sleep(time.Duration(1000 * time.Millisecond))
					default:
						break
					}
				} else {
					return err
				}
			} else {
				fmt.Println("Report sent")
				if c.stopOnSuccess {
					return nil
				}
				break
			}
		}
	}
	return nil
}

type StatusErr struct {
	StatusCode int
	Message    string
}

func (e StatusErr) Error() string {
	return e.Message
}

func (c *RemoteCollector) Collect(report Report) error {
	jsonData, err := json.Marshal(report)
	var statusCode int
	if err != nil {
		log.Printf("Error encoding JSON: %s\n", err)
		return err
	}
	statusCode, err = sendReport(jsonData, c.collectorEndpoint)
	if err != nil {
		log.Printf("Send report failed: %v", err)
		return err
	} else if statusCode > 400 {
		return StatusErr{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("HTTP request failed with status code %d", statusCode),
		}
	} else {
		fmt.Println("Report sent")
		return nil
	}
}

func (c *SamplingCollector) Collect(report Report) error {
	var samplingRate float64
	if report.IsSuccess() {
		samplingRate = c.successFraction
	} else {
		samplingRate = c.failureFraction
	}
	// Generate a random float64 number between 0 and 1
	random := rand.Float64()
	if random < samplingRate {
		err := c.collector.Collect(report)
		if err != nil {
			log.Printf("Error collecting report: %v", err)
			return err
		}
		return nil
	} else {
		fmt.Println("Report was not sent this time")
		return nil
	}
}

func sendReport(jsonData []byte, remote *url.URL) (int, error) {
	// TODO: return status code of HTTP response
	req, err := http.NewRequest("POST", remote.String(), bytes.NewReader(jsonData))
	if err != nil {
		debugLog.Printf("Error creating the HTTP request: %s\n", err)
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error sending the HTTP request: %s\n", err)
		return 0, err
	}
	defer resp.Body.Close()
	// Access the HTTP response status code
	fmt.Printf("HTTP Response Status Code: %d\n", resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLog.Printf("Error reading the HTTP response body: %s\n", err)
		return 0, err
	}
	debugLog.Printf("Response: %s\n", respBody)
	return resp.StatusCode, nil
}
