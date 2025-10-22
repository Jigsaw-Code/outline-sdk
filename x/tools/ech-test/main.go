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
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
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
	QueryType   dnsmessage.Type
	Timestamp   time.Time
	Duration    time.Duration
	Error       error
	RCode       dnsmessage.RCode
	CNAMEs      []string
	Answers     []dnsmessage.Resource
	Additionals []dnsmessage.Resource
}

func resolve(resolver dns.Resolver, domain Domain, qtype dnsmessage.Type) QueryResult {
	startTime := time.Now()

	result := QueryResult{
		Timestamp: startTime,
		Domain:    domain,
		QueryType: qtype,
	}

	q, err := dns.NewQuestion(domain.Name, qtype)
	if err != nil {
		result.Error = fmt.Errorf("failed to create question: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	queryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := resolver.Query(queryCtx, *q)
	result.Duration = time.Since(startTime)
	if err != nil {
		slog.Debug("Query failed", "domain", domain.Name, "type", qtype.String(), "error", err)
		result.Error = fmt.Errorf("query failed: %v", formatError(err))
		return result
	}

	result.RCode = resp.RCode
	result.Answers = resp.Answers
	result.Additionals = resp.Additionals
	return result
}

func formatResourceBody(body dnsmessage.ResourceBody) (interface{}, error) {
	switch b := body.(type) {
	case *dnsmessage.AResource:
		return net.IP(b.A[:]).String(), nil
	case *dnsmessage.AAAAResource:
		return net.IP(b.AAAA[:]).String(), nil
	case *dnsmessage.HTTPSResource:
		params := make(map[string]interface{})
		for _, p := range b.Params {
			var value interface{}
			switch p.Key {
			case dnsmessage.SVCParamALPN:
				var alpnValues []string
				alpnBytes := p.Value
				for len(alpnBytes) > 0 {
					strLen := int(alpnBytes[0])
					if len(alpnBytes) < 1+strLen {
						value = fmt.Sprintf("malformed_alpn: %x", p.Value)
						break
					}
					alpnValues = append(alpnValues, string(alpnBytes[1:1+strLen]))
					alpnBytes = alpnBytes[1+strLen:]
				}
				if value == nil {
					value = alpnValues
				}
			case dnsmessage.SVCParamIPv4Hint:
				var ips []string
				for i := 0; i < len(p.Value); i += net.IPv4len {
					ips = append(ips, net.IP(p.Value[i:i+net.IPv4len]).String())
				}
				value = ips
			case dnsmessage.SVCParamIPv6Hint:
				var ips []string
				for i := 0; i < len(p.Value); i += net.IPv6len {
					ips = append(ips, net.IP(p.Value[i:i+net.IPv6len]).String())
				}
				value = ips
			default:
				value = fmt.Sprintf("%x", p.Value)
			}
			params[p.Key.String()] = value
		}
		return map[string]interface{}{
			"priority": b.Priority,
			"target":   b.Target.String(),
			"params":   params,
		}, nil
	case *dnsmessage.CNAMEResource:
		return b.CNAME.String(), nil
	default:
		return body.GoString(), nil
	}
}

func formatResources(resources []dnsmessage.Resource) (string, error) {
	var out []interface{}
	for _, r := range resources {
		if r.Header.Type == dnsmessage.TypeOPT {
			continue
		}
		body, err := formatResourceBody(r.Body)
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

func extractCNAMEs(resources []dnsmessage.Resource) ([]dnsmessage.Resource, []dnsmessage.Resource) {
	var cnames []dnsmessage.Resource
	var cleanAnswers []dnsmessage.Resource
	for _, r := range resources {
		if r.Header.Type == dnsmessage.TypeCNAME {
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
				strings.TrimPrefix(result.QueryType.String(), "Type"),
				result.Timestamp.Format(time.RFC3339Nano),
				strconv.FormatInt(result.Duration.Milliseconds(), 10),
				errorText,
				strings.TrimPrefix(result.RCode.String(), "RCode"),
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
	for _, domain := range domains {
		go func(d Domain) {
			defer resolveWg.Done()
			slog.Info("Analyzing", "domain", d.Name)
			resolver := dns.NewUDPResolver(&transport.UDPDialer{}, "8.8.8.8:53")

			if err := sem.Acquire(context.Background(), 3); err != nil {
				slog.Error("Failed to acquire semaphore", "domain", d.Name, "error", err)
				return
			}
			resolveWg.Add(3)
			go func() {
				resultsCh <- resolve(resolver, d, dnsmessage.TypeA)
				sem.Release(1)
				resolveWg.Done()
			}()
			go func() {
				resultsCh <- resolve(resolver, d, dnsmessage.TypeAAAA)
				sem.Release(1)
				resolveWg.Done()
			}()
			go func() {
				resultsCh <- resolve(resolver, d, dnsmessage.TypeHTTPS)
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
