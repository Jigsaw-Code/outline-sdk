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

// TODO:
// * Save augmented domains to file
// * Introduce Rate-limiting DNS client. But It needs to block the Go routine.

package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/tools/ech-test/internal/tranco"
	"github.com/Jigsaw-Code/outline-sdk/x/tools/ech-test/internal/workspace"
	"github.com/miekg/dns"
	"golang.org/x/sync/semaphore"
)

type Domain struct {
	Name             string
	Rank             int
	CanonicalName    string
	StartOfAuthority string
	Nameservers      []string
}

type QueryResult struct {
	Domain      Domain
	RunNumber   int
	QueryType   uint16
	Timestamp   time.Time
	Duration    time.Duration
	Error       error
	RCode       int
	CNAMEs      []string
	Answers     []dns.RR
	Additionals []dns.RR
}

func resolve(client *dns.Client, resolverAddress string, domain Domain, qtype uint16, run int) QueryResult {
	startTime := time.Now()

	result := QueryResult{
		Timestamp: startTime,
		Domain:    domain,
		RunNumber: run,
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

const (
	resolverAddress = "8.8.8.8:53"
)

func main() {
	workspaceFlag := flag.String("workspace", "./workspace", "Directory to store intermediate files")
	trancoIDFlag := flag.String("trancoID", "7NZ4X", "Tranco list ID to use")
	topNFlag := flag.Int("topN", 100, "Number of top domains to analyze")
	parallelismFlag := flag.Int("parallelism", 10, "Maximum number of parallel requests")
	numQueriesFlag := flag.Int("numQueries", 1, "Number of times to query each domain")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *verboseFlag {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	// Set up workspace directory.
	workspaceDir := workspace.EnsureWorkspace(*workspaceFlag)

	// Ensure Tranco list is present.
	trancoList := tranco.NewTrancoList(workspaceDir, *trancoIDFlag)

	// Read top N domains from Tranco CSV.
	trancoDomains, err := trancoList.TopDomains(*topNFlag)
	if err != nil {
		slog.Error("Failed to read domains from Tranco CSV", "error", err)
		os.Exit(1)
	}

	// Set up DNS client.
	client := new(dns.Client)
	client.ReadTimeout = 5 * time.Second
	client.WriteTimeout = 5 * time.Second

	// Set up parallelism semaphore for rate-limiting.
	sem := semaphore.NewWeighted(int64(*parallelismFlag))

	// Collect extra domain information.
	domains := make([]Domain, len(trancoDomains))
	var domainsWg sync.WaitGroup
	domainsWg.Add(len(trancoDomains))
	for i, d := range trancoDomains {
		sem.Acquire(context.Background(), 1)
		go func(i int, d tranco.Domain) {
			defer sem.Release(1)
			defer domainsWg.Done()
			msg := new(dns.Msg)
			msg.SetQuestion(dns.Fqdn(d.Name), dns.TypeSOA)
			msg.RecursionDesired = true

			slog.Info("Collecting SOA for domain", "rank", d.Rank, "name", d.Name)
			r, _, err := client.Exchange(msg, resolverAddress)
			if err != nil {
				slog.Error("SOA query failed", "domain", d.Name, "type", dns.TypeToString[dns.TypeSOA], "error", err)
				os.Exit(1)
			}
			// Collect the canonical name and start of authority.
			cname := d.Name
			soa := ""
			for _, answer := range r.Answer {
				if cnameRR, ok := answer.(*dns.CNAME); ok {
					cname = cnameRR.Target
				}
				if soaRR, ok := answer.(*dns.SOA); ok {
					soa = soaRR.Hdr.Name
				}
			}
			if soa == "" {
				for _, ns := range r.Ns {
					if soaRR, ok := ns.(*dns.SOA); ok {
						soa = soaRR.Hdr.Name
						break
					}
				}
			}
			nsList, err := net.LookupNS(soa)
			if err != nil {
				slog.Error("NS lookup failed", "domain", d.Name, "soa", soa, "error", err)
				os.Exit(1)
			}
			var nameservers []string
			for _, ns := range nsList {
				nameservers = append(nameservers, ns.Host)
			}
			sort.Strings(nameservers)
			domains[i] = Domain{
				Name:             d.Name,
				Rank:             d.Rank,
				CanonicalName:    cname,
				StartOfAuthority: soa,
				Nameservers:      nameservers,
			}
		}(i, d)
	}
	domainsWg.Wait()

	// Create new output CSV file.
	outputFilename := filepath.Join(workspaceDir, fmt.Sprintf("results-top%d-n%d.csv", *topNFlag, *numQueriesFlag))
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		slog.Error("Failed to create output CSV file", "path", outputFilename, "error", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	header := []string{"domain", "cname", "soa", "nameservers", "rank", "run", "query_type", "timestamp", "duration_ms", "error", "rcode", "cnames", "answers", "additionals"}
	if err := csvWriter.Write(header); err != nil {
		slog.Error("Failed to write CSV header", "error", err)
		os.Exit(1)
	}

	resultsCh := make(chan QueryResult, 3*(*topNFlag)*(*numQueriesFlag))

	var csvWg sync.WaitGroup
	csvWg.Add(1)
	go func() {
		defer csvWg.Done()
		for result := range resultsCh {
			nameserversJSON, err := json.Marshal(result.Domain.Nameservers)
			if err != nil {
				slog.Error("Failed to marshal nameservers", "error", err)
				nameserversJSON = []byte("[]")
			}
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
				result.Domain.CanonicalName,
				result.Domain.StartOfAuthority,
				string(nameserversJSON),
				strconv.Itoa(result.Domain.Rank),
				strconv.Itoa(result.RunNumber),
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

	var resolveWg sync.WaitGroup
	resolveWg.Add(len(domains) * (*numQueriesFlag) * 3)

	for i := 0; i < *numQueriesFlag; i++ {
		for _, domain := range domains {
			d := domain
			run := i + 1
			if err := sem.Acquire(context.Background(), 3); err != nil {
				slog.Error("Failed to acquire semaphore", "domain", d.Name, "error", err)
				os.Exit(1)
			}

			slog.Info("Analyzing domain", "rank", d.Rank, "run", run, "name", d.Name)

			go func() {
				defer sem.Release(1)
				defer resolveWg.Done()
				resultsCh <- resolve(client, resolverAddress, d, dns.TypeA, run)
			}()
			go func() {
				defer sem.Release(1)
				defer resolveWg.Done()
				resultsCh <- resolve(client, resolverAddress, d, dns.TypeAAAA, run)
			}()
			go func() {
				defer sem.Release(1)
				defer resolveWg.Done()
				resultsCh <- resolve(client, resolverAddress, d, dns.TypeHTTPS, run)
			}()
		}
	}
	resolveWg.Wait()
	close(resultsCh)

	csvWg.Wait()

	slog.Info("Done. Results saved to", "path", outputFilename)
}
