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
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
)

func download(zipURL, localFilename string) error {
	slog.Info("Downloading Tranco list", "url", zipURL, "to", localFilename)
	resp, err := http.Get(zipURL)
	if err != nil {
		return fmt.Errorf("failed to download zip file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	zipFile, err := os.Create(localFilename)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	if _, err := io.Copy(zipFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save zip file: %w", err)
	}

	return nil
}

func resolve(resolver dns.Resolver, domain string, qtype dnsmessage.Type) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q, err := dns.NewQuestion(domain, qtype)
	if err != nil {
		return fmt.Sprintf("error: failed to create question: %v", err)
	}

	resp, err := resolver.Query(ctx, *q)
	if err != nil {
		return fmt.Sprintf("error: query failed: %v", err)
	}

	if resp.RCode != dnsmessage.RCodeSuccess {
		return fmt.Sprintf("error: rcode is %v", resp.RCode)
	}

	var answers []string
	for _, ans := range resp.Answers {
		answers = append(answers, ans.Body.GoString())
	}
	return fmt.Sprintf("response: [%s]", strings.Join(answers, ", "))
}

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

func ensureTrancoList(workspaceDir, trancoID string) string {
	trancoZipFilename := filepath.Join(workspaceDir, fmt.Sprintf("tranco_%s-1m.csv.zip", trancoID))
	if _, err := os.Stat(trancoZipFilename); os.IsNotExist(err) {
		trancoZipURL := fmt.Sprintf("https://tranco-list.eu/download/daily/tranco_%s-1m.csv.zip", trancoID)
		if err := download(trancoZipURL, trancoZipFilename); err != nil {
			slog.Error("Failed to get Tranco list", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("Found Tranco list", "path", trancoZipFilename)
	}
	return trancoZipFilename
}

func main() {
	workspaceFlag := flag.String("workspace", "./workspace", "Directory to store intermediate files")
	trancoIDFlag := flag.String("trancoID", "7NZ4X", "Tranco list ID to use")
	topNFlag := flag.Int("topN", 100, "Number of top domains to analyze")
	flag.Parse()

	// Set up workspace directory.
	workspaceDir := ensureWorkspace(*workspaceFlag)

	// Ensure Tranco list is present.
	trancoZipFilename := ensureTrancoList(workspaceDir, *trancoIDFlag)

	zipReader, err := zip.OpenReader(trancoZipFilename)
	if err != nil {
		slog.Error("Failed to open Tranco ZIP file", "path", trancoZipFilename, "error", err)
		os.Exit(1)
	}
	defer zipReader.Close()

	csvFile, err := zipReader.Open("top-1m.csv")
	if err != nil {
		slog.Error("Failed to open CSV file inside ZIP", "error", err)
		os.Exit(1)
	}
	defer csvFile.Close()
	csvReader := csv.NewReader(csvFile)
	var domains []string
	for i := 0; i < *topNFlag; i++ {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("Failed to read from Tranco CSV", "error", err)
			os.Exit(1)
		}
		domains = append(domains, record[1])
	}

	/*
		outputFilename := filepath.Join(workspaceDir, fmt.Sprintf("results-top%d.csv", *topNFlag))
		outputFile, err := os.Create(outputFilename)
		if err != nil {
			slog.Error("Failed to create output CSV file", "path", outputFilename, "error", err)
			os.Exit(1)
		}
		defer outputFile.Close()

		csvWriter := csv.NewWriter(outputFile)
		defer csvWriter.Flush()

		csvWriter.Write([]string{"domain", "A", "AAAA", "HTTPS"})
	*/
	resolver := dns.NewUDPResolver(&transport.UDPDialer{}, "8.8.8.8:53")
	var wg sync.WaitGroup
	// resultsCh := make(chan []string, len(domains))
	for _, domain := range domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			slog.Info("Analyzing", "domain", d)
			aResponse := resolve(resolver, d, dnsmessage.TypeA)
			aaaaResponse := resolve(resolver, d, dnsmessage.TypeAAAA)
			httpsResponse := resolve(resolver, d, dnsmessage.Type(65))
			// resultsCh <- []string{d, aResponse, aaaaResponse, httpsResponse}
			slog.Info("Result", "domain", d, "A", aResponse, "AAAA", aaaaResponse, "HTTPS", httpsResponse)
			// resultsCh <- []string{d, aResponse, aaaaResponse, ""}
		}(domain)
	}

	wg.Wait()
	//close(resultsCh)
	/*
		for result := range resultsCh {
			csvWriter.Write(result)
		}
		slog.Info("Done. Results saved to", "path", outputFilename)
	*/
}
