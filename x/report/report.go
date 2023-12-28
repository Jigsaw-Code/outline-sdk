// Package report provides functionality for collecting and sending reports.
//
// It defines the behavior of a report collector and provides implementations of the [Collector] interface.
// It also defines a report type and a [HasSuccess] interface that is implemented by the report type.
// The report type is used to represent a connectivity test report.
// The [HasSuccess] interface is used to determine the success status of a report. This will be used to control [SamplingCollector] behavior.
// The report package also defines a [BadRequestError] type that is used to represent an error that occurs when a sending the report to remote collector fails.
package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// BadRequestError represents an error that occurs when a request fails.
type BadRequestError struct {
	Err error
}

// Error returns the error message associated with the [BadRequestError].
func (e BadRequestError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error wrapped by the [BadRequestError].
func (e BadRequestError) Unwrap() error {
	return e.Err
}

// Report is an alias for any type of report.
type Report any

// HasSuccess is an interface that represents an object that has a success status.
type HasSuccess interface {
	IsSuccess() bool
}

// Collector is an interface that defines the behavior of a report collector.
// Implementations of this interface should be able to collect a report in a given context.
type Collector interface {
	Collect(context.Context, Report) error
}

// RemoteCollector represents a collector that communicates with a remote endpoint.
type RemoteCollector struct {
	HttpClient   *http.Client
	CollectorURL *url.URL
}

// Collect sends the given report to the remote collector.
// It marshals the report into JSON format and sends it using the [sendReport] method.
// If there is an error encoding the JSON or sending the report, it returns the error.
// Otherwise, it returns nil.
func (c *RemoteCollector) Collect(ctx context.Context, report Report) error {
	jsonData, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	err = c.sendReport(ctx, jsonData)
	if err != nil {
		return err
	}
	return nil
}

// SamplingCollector represents a collector that randomly samples and collects a report.
type SamplingCollector struct {
	Collector       Collector
	SuccessFraction float64
	FailureFraction float64
}

// Collect collects the given report based on the sampling rate defined in the [SamplingCollector].
// It checks if the report implements the [HasSuccess] interface and determines the sampling rate based on the success status.
// If the randomly generated number is less than the sampling rate, the report is collected using the underlying collector.
// Otherwise, the report is not sent.
// It returns an error if there is an issue collecting the report.
// Sampling rate of 1.0 means report is always sent, and 0.0 means report is never sent.
func (c *SamplingCollector) Collect(ctx context.Context, report Report) error {
	var samplingRate float64
	hs, ok := report.(HasSuccess)
	if !ok {
		return nil
	}
	if hs.IsSuccess() {
		samplingRate = c.SuccessFraction
	} else {
		samplingRate = c.FailureFraction
	}
	// Generate a random float64 number between 0 and 1
	random := rand.Float64()
	if random < samplingRate {
		err := c.Collector.Collect(ctx, report)
		if err != nil {
			return err
		}
		return nil
	} else {
		return nil
	}
}

// RetryCollector represents a collector that supports retrying failed operations.
type RetryCollector struct {
	Collector    Collector
	MaxRetry     int
	InitialDelay time.Duration
}

// Collect collects the report by making multiple attempts with retries.
// It uses the provided context and report to call the underlying collector's [Collect] method.
// If a [BadRequestError] is encountered during the collection, it breaks the retry loop.
// It sleeps for a specified duration between retries.
// Returns an error if the maximum number of retries is exceeded.
func (c *RetryCollector) Collect(ctx context.Context, report Report) error {
	var e *BadRequestError
	for i := 0; i < c.MaxRetry+1; i++ {
		err := c.Collector.Collect(ctx, report)
		if err != nil {
			if errors.As(err, &e) {
				break
			} else {
				time.Sleep(time.Duration(math.Pow(2, float64(i))) * c.InitialDelay)
			}
		} else {
			return nil
		}
	}
	return errors.New("max retry exceeded")
}

// FallbackCollector is a type that represents a collector that falls back to multiple collectors.
type FallbackCollector struct {
	Collectors []Collector
}

// Collect implements [Collector] interface on [FallbackCollector] type that collects a report using the provided context and report data.
// It iterates over a list of collectors and attempts to collect the report using each collector.
// If any of the collectors succeeds in collecting the report, operation aborts, and it returns nil.
// If all collectors fail to collect the report, it returns an error indicating the failure.
func (c *FallbackCollector) Collect(ctx context.Context, report Report) error {
	for i := range c.Collectors {
		err := c.Collectors[i].Collect(ctx, report)
		if err == nil {
			return nil
		}
	}
	return errors.New("all collectors failed")
}

// sendReport sends a report to the remote collector.
// It takes a context.Context object for cancellation and deadline propagation,
// and a []byte containing the JSON data to be sent.
// It returns an error if there was a problem sending the report or reading the response.
func (c *RemoteCollector) sendReport(ctx context.Context, jsonData []byte) error {
	// TODO: return status code of HTTP response
	req, err := http.NewRequest("POST", c.CollectorURL.String(), bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.HttpClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if 400 <= resp.StatusCode && resp.StatusCode < 500 {
		return &BadRequestError{
			Err: fmt.Errorf("http request failed with status code %d", resp.StatusCode),
		}
	}
	return nil
}

// WriteCollector represents a collector that writes the report to an io.Writer.
type WriteCollector struct {
	Writer io.Writer
}

// Collect writes the report to the underlying io.Writer.
// It returns an error if there was a problem writing the report.
func (c *WriteCollector) Collect(ctx context.Context, report Report) error {
	jsonData, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, err = fmt.Fprintln(c.Writer, string(jsonData))
	if err != nil {
		return err
	}
	return nil
}
