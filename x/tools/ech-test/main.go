// Copyright 2025 The Outline Authors
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

package main

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/sync/semaphore"
)

// downloadFile downloads the file from fileURL and saves it as localFilename.
func downloadFile(fileURL, localFilename string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	localFile, err := os.Create(localFilename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer localFile.Close()

	if _, err := io.Copy(localFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

type Domain struct {
	Name        string
	Rank        int
	Nameservers []string
}

type QueryResult struct {
	Domain      Domain
	QueryType   uint16
	Timestamp   time.Time
	Duration    time.Duration
	Error       error
	RCode       int
	CNAMEs      []string
	Answers     []dns.RR
	Additionals []dns.RR
}

func resolve(client *dns.Client, resolverAddress string, domain Domain, qtype uint16) QueryResult {
	startTime := time.Now()

	result := QueryResult{
		Timestamp: startTime,
		Domain:    domain,
		QueryType: qtype,
	}

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain.Name), qtype)
	msg.RecursionDesired = true

	r, _, err := client.Exchange(msg, resolverAddress)
	result.Duration = time.Since(startTime)
	if err != nil {
		slog.Debug("Query failed", "domain", domain.Name, "type", dns.TypeToString[qtype], "error", err)
		result.Error = fmt.Errorf("query failed: %v", formatError(err))
		return result
	}
	if r == nil {
		slog.Debug("Query failed", "domain", domain.Name, "type", dns.TypeToString[qtype], "error", "nil response")
		result.Error = fmt.Errorf("query failed: nil response")
		return result
	}

	result.RCode = r.Rcode
	result.Answers = r.Answer
	result.Additionals = r.Extra
	return result
}

func formatResourceBody(rr dns.RR) (interface{}, error) {
	switch r := rr.(type) {
	case *dns.A:
		return r.A.String(), nil
	case *dns.AAAA:
		return r.AAAA.String(), nil
	case *dns.HTTPS:
		params := make(map[string]interface{})
		for _, p := range r.Value {
			var value interface{}
			switch p.Key() {
			case dns.SVCB_ALPN:
				alpn := p.(*dns.SVCBAlpn)
				value = alpn.Alpn
			case dns.SVCB_IPV4HINT:
				v4hint := p.(*dns.SVCBIPv4Hint)
				var ips []string
				for _, ip := range v4hint.Hint {
					ips = append(ips, ip.String())
				}
				value = ips
			case dns.SVCB_IPV6HINT:
				v6hint := p.(*dns.SVCBIPv6Hint)
				var ips []string
				for _, ip := range v6hint.Hint {
					ips = append(ips, ip.String())
				}
				value = ips
			default:
				value = p.String()
			}
			params[p.Key().String()] = value
		}
		return map[string]interface{}{
			"priority": r.Priority,
			"target":   r.Target,
			"params":   params,
		}, nil
	case *dns.CNAME:
		return r.Target, nil
	default:
		return r.String(), nil
	}
}

func formatResources(resources []dns.RR) (string, error) {
	var out []interface{}
	for _, r := range resources {
		if r.Header().Rrtype == dns.TypeOPT {
			continue
		}
		body, err := formatResourceBody(r)
		if err != nil {
			return "", err
		}
		out = append(out, body)
	}
	if len(out) == 0 {
		return "[]", nil
	}
	jsonBytes, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// ensureWorkspace ensures the workspace directory exists, creating it if needed.
func ensureWorkspace(workspaceDir string) string {
	workspaceAbsDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		slog.Error("Failed to resolve workspace path", "error", err)
		os.Exit(1)
	}
	if _, err := os.Stat(workspaceAbsDir); os.IsNotExist(err) {
		slog.Info("Creating workspace directory", "path", workspaceAbsDir)
		if err := os.MkdirAll(workspaceAbsDir, 0755); err != nil {
			slog.Error("Failed to create workspace directory", "error", err)
			os.Exit(1)
		}
	}
	return workspaceAbsDir
}

// ensureTrancoList ensures the Tranco list is in the workspace directory, downloading it if needed.
func ensureTrancoList(workspaceDir, trancoID string) string {
	trancoZipFilename := filepath.Join(workspaceDir, fmt.Sprintf("tranco_%s-1m.csv.zip", trancoID))
	if _, err := os.Stat(trancoZipFilename); os.IsNotExist(err) {
		trancoZipURL := fmt.Sprintf("https://tranco-list.eu/download/daily/tranco_%s-1m.csv.zip", trancoID)
		slog.Info("Downloading Tranco list", "url", trancoZipURL, "to", trancoZipFilename)
		if err := downloadFile(trancoZipURL, trancoZipFilename); err != nil {
			slog.Error("Failed to get Tranco list", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("Found Tranco list", "path", trancoZipFilename)
	}
	return trancoZipFilename
}

func readDomainsFromTrancoCSV(trancoZipFilename string, topN int) ([]Domain, error) {
	zipReader, err := zip.OpenReader(trancoZipFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to open Tranco ZIP file: %w", err)
	}
	defer zipReader.Close()

	csvFile, err := zipReader.Open("top-1m.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file inside ZIP: %w", err)
	}
	defer csvFile.Close()
	csvReader := csv.NewReader(csvFile)
	var domains []Domain
	for i := 0; i < topN; i++ {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read from Tranco CSV: %w", err)
		}
		// Format is <rank>,<domain>
		rank, err := strconv.Atoi(record[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse rank: %w", err)
		}
		domains = append(domains, Domain{Name: record[1], Rank: rank})
	}
	return domains, nil
}

func extractCNAMEs(resources []dns.RR) ([]dns.RR, []dns.RR) {
	var cnames []dns.RR
	var cleanAnswers []dns.RR
	for _, r := range resources {
		if r.Header().Rrtype == dns.TypeCNAME {
			cnames = append(cnames, r)
		} else {
			cleanAnswers = append(cleanAnswers, r)
		}
	}
	return cnames, cleanAnswers
}

func isTimeout(err error) bool {
	var timeErr interface{ Timeout() bool }
	return errors.As(err, &timeErr) && timeErr.Timeout()
}

func formatError(err error) string {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno.Error()
	} else if isTimeout(err) {
		return "ETIMEDOUT"
	}
	return err.Error()
}

func main() {
	workspaceFlag := flag.String("workspace", "./workspace", "Directory to store intermediate files")
	trancoIDFlag := flag.String("trancoID", "7NZ4X", "Tranco list ID to use")
	topNFlag := flag.Int("topN", 100, "Number of top domains to analyze")
	parallelismFlag := flag.Int("parallelism", 10, "Maximum number of parallel requests")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *verboseFlag {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	// Set up workspace directory.
	workspaceDir := ensureWorkspace(*workspaceFlag)

	// Ensure Tranco list is present.
	trancoZipFilename := ensureTrancoList(workspaceDir, *trancoIDFlag)

	// Read top N domains from Tranco CSV.
	domains, err := readDomainsFromTrancoCSV(trancoZipFilename, *topNFlag)
	if err != nil {
		slog.Error("Failed to read domains from Tranco CSV", "error", err)
		os.Exit(1)
	}

	// Create new output CSV file.
	outputFilename := filepath.Join(workspaceDir, fmt.Sprintf("results-top%d.csv", *topNFlag))
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		slog.Error("Failed to create output CSV file", "path", outputFilename, "error", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	header := []string{"domain", "rank", "query_type", "timestamp", "duration_ms", "error", "rcode", "cnames", "answers", "additionals"}
	if err := csvWriter.Write(header); err != nil {
		slog.Error("Failed to write CSV header", "error", err)
		os.Exit(1)
	}

	resultsCh := make(chan QueryResult, 3*(*topNFlag))

	var csvWg sync.WaitGroup
	csvWg.Add(1)
	go func() {
		defer csvWg.Done()
		for result := range resultsCh {
			cnames, cleanAnswers := extractCNAMEs(result.Answers)
			cnamesJSON, err := formatResources(cnames)
			if err != nil {
				slog.Error("Failed to format cnames", "error", err)
				cnamesJSON = "[]"
			}
			answersJSON, err := formatResources(cleanAnswers)
			if err != nil {
				slog.Error("Failed to format answers", "error", err)
				answersJSON = "[]"
			}
			additionalsJSON, err := formatResources(result.Additionals)
			if err != nil {
				slog.Error("Failed to format additionals", "error", err)
				additionalsJSON = "[]"
			}
			errorText := ""
			if result.Error != nil {
				errorText = result.Error.Error()
			}
			record := []string{
				result.Domain.Name,
				strconv.Itoa(result.Domain.Rank),
				dns.TypeToString[result.QueryType],
				result.Timestamp.Format(time.RFC3339Nano),
				strconv.FormatInt(result.Duration.Milliseconds(), 10),
				errorText,
				dns.RcodeToString[result.RCode],
				cnamesJSON,
				answersJSON,
				additionalsJSON,
			}
			if err := csvWriter.Write(record); err != nil {
				slog.Error("Failed to write record to CSV", "error", err)
			}
		}
	}()

	sem := semaphore.NewWeighted(int64(*parallelismFlag))
	var resolveWg sync.WaitGroup
	resolveWg.Add(len(domains))
	client := new(dns.Client)
	client.ReadTimeout = 5 * time.Second
	client.WriteTimeout = 5 * time.Second
	resolverAddress := "8.8.8.8:53"

	for _, domain := range domains {
		if err := sem.Acquire(context.Background(), 3); err != nil {
			slog.Error("Failed to acquire semaphore", "domain", domain.Name, "error", err)
			return
		}
		go func(d Domain) {
			defer resolveWg.Done()
			slog.Info("Analyzing", "domain", d.Name)

			resolveWg.Add(3)
			go func() {
				resultsCh <- resolve(client, resolverAddress, d, dns.TypeA)
				sem.Release(1)
				resolveWg.Done()
			}()
			go func() {
				resultsCh <- resolve(client, resolverAddress, d, dns.TypeAAAA)
				sem.Release(1)
				resolveWg.Done()
			}()
			go func() {
				resultsCh <- resolve(client, resolverAddress, d, dns.TypeHTTPS)
				sem.Release(1)
				resolveWg.Done()
			}()
		}(domain)
	}
	resolveWg.Wait()
	close(resultsCh)

	csvWg.Wait()

	slog.Info("Done. Results saved to", "path", outputFilename)
}
