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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)
var httpClient = &http.Client{}

type BadRequestErr struct {
	StatusCode int
	Message    string
}

func (e BadRequestErr) Error() string {
	return e.Message
}

type ConnectivityReport struct {
	// Connection setup
	Connection interface{} `json:"connection"`
	// Observations
	Time       time.Time   `json:"time"`
	DurationMs int64       `json:"durationMs"`
	Error      interface{} `json:"error"`
}

type Report any

type HasSuccess interface {
	IsSuccess() bool
}

// ConnectivityReport implements the HasSuccess interface
func (r ConnectivityReport) IsSuccess() bool {
	if r.Error == nil {
		return true
	} else {
		return false
	}
}

type Collector interface {
	Collect(context.Context, Report) error
}

type RemoteCollector struct {
	collectorEndpoint *url.URL
}

func (c *RemoteCollector) Collect(ctx context.Context, report Report) error {
	jsonData, err := json.Marshal(report)
	if err != nil {
		log.Printf("Error encoding JSON: %s\n", err)
		return err
	}
	err = sendReport(ctx, jsonData, c.collectorEndpoint)
	if err != nil {
		log.Printf("Send report failed: %v", err)
		return err
	}
	fmt.Println("Report sent")
	return nil
}

type SamplingCollector struct {
	collector       Collector
	successFraction float64
	failureFraction float64
}

func (c *SamplingCollector) Collect(ctx context.Context, report Report) error {
	var samplingRate float64
	hs, ok := report.(HasSuccess)
	if !ok {
		log.Printf("Report does not implement HasSuccess interface")
		return nil
	}
	if hs.IsSuccess() {
		samplingRate = c.successFraction
	} else {
		samplingRate = c.failureFraction
	}
	// Generate a random float64 number between 0 and 1
	random := rand.Float64()
	if random < samplingRate {
		err := c.collector.Collect(ctx, report)
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

type RetryCollector struct {
	collector        Collector
	maxRetry         int
	waitBetweenRetry time.Duration
}

func (c *RetryCollector) Collect(ctx context.Context, report Report) error {
	for i := 0; i < c.maxRetry; i++ {
		err := c.collector.Collect(ctx, report)
		if err != nil {
			if _, ok := err.(BadRequestErr); ok {
				break
			} else {
				time.Sleep(c.waitBetweenRetry)
			}
		} else {
			fmt.Println("Report sent")
			return nil
		}
	}
	return errors.New("max retry exceeded")
}

type MutltiCollector struct {
	collectors []Collector
}

// Collects reports through multiple collectors
func (c *MutltiCollector) Collect(ctx context.Context, report Report) error {
	success := false
	for i := range c.collectors {
		err := c.collectors[i].Collect(ctx, report)
		if err != nil {
			log.Printf("Error collecting report: %v", err)
			success = success || false
		} else {
			success = success || true
		}
	}
	if success {
		// At least one collector succeeded
		return nil
	}
	return errors.New("all collectors failed")
}

type FallbackCollector struct {
	collectors []Collector
}

// Collects reports through multiple collectors
func (c *FallbackCollector) Collect(ctx context.Context, report Report) error {
	for i := range c.collectors {
		err := c.collectors[i].Collect(ctx, report)
		if err == nil {
			debugLog.Println("Report sent!")
			return nil
		}
	}
	return errors.New("all collectors failed")
}

func sendReport(ctx context.Context, jsonData []byte, remote *url.URL) error {
	// TODO: return status code of HTTP response
	req, err := http.NewRequest("POST", remote.String(), bytes.NewReader(jsonData))
	if err != nil {
		debugLog.Printf("Error creating the HTTP request: %s\n", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := httpClient.Do(req.WithContext(ctx))
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
	if resp.StatusCode >= 400 {
		debugLog.Printf("Error sending the report: %s\n", respBody)
		return BadRequestErr{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP request failed with status code %d", resp.StatusCode),
		}
	}
	debugLog.Printf("Response: %s\n", respBody)
	return nil
}
