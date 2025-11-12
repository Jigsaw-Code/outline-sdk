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
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/tools/ech-test/internal/tranco"
	"github.com/Jigsaw-Code/outline-sdk/x/tools/ech-test/internal/workspace"
	"golang.org/x/sync/semaphore"
)

type TestResult struct {
	Domain        string
	Rank          int
	ECHGrease     bool
	Error         string
	CurlExitCode  int
	CurlErrorName string
	DNSLookup     time.Duration
	TCPConnection time.Duration
	TLSHandshake  time.Duration
	ServerTime    time.Duration
	TotalTime     time.Duration
	HTTPStatus    int
}

var curlExitCodeNames = map[int]string{
	1:  "CURLE_UNSUPPORTED_PROTOCOL",
	2:  "CURLE_FAILED_INIT",
	3:  "CURLE_URL_MALFORMAT",
	4:  "CURLE_NOT_BUILT_IN",
	5:  "CURLE_COULDNT_RESOLVE_PROXY",
	6:  "CURLE_COULDNT_RESOLVE_HOST",
	7:  "CURLE_COULDNT_CONNECT",
	8:  "CURLE_WEIRD_SERVER_REPLY",
	9:  "CURLE_REMOTE_ACCESS_DENIED",
	11: "CURLE_FTP_WEIRD_PASV_REPLY",
	13: "CURLE_FTP_WEIRD_227_FORMAT",
	14: "CURLE_FTP_CANT_GET_HOST",
	15: "CURLE_FTP_CANT_RECONNECT",
	17: "CURLE_FTP_COULDNT_SET_TYPE",
	18: "CURLE_PARTIAL_FILE",
	19: "CURLE_FTP_COULDNT_RETR_FILE",
	21: "CURLE_QUOTE_ERROR",
	22: "CURLE_HTTP_RETURNED_ERROR",
	23: "CURLE_WRITE_ERROR",
	25: "CURLE_UPLOAD_FAILED",
	26: "CURLE_READ_ERROR",
	27: "CURLE_OUT_OF_MEMORY",
	28: "CURLE_OPERATION_TIMEDOUT",
	30: "CURLE_FTP_PORT_FAILED",
	31: "CURLE_FTP_COULDNT_USE_REST",
	33: "CURLE_RANGE_ERROR",
	34: "CURLE_HTTP_POST_ERROR",
	35: "CURLE_SSL_CONNECT_ERROR",
	36: "CURLE_BAD_DOWNLOAD_RESUME",
	37: "CURLE_FILE_COULDNT_READ_FILE",
	38: "CURLE_LDAP_CANNOT_BIND",
	39: "CURLE_LDAP_SEARCH_FAILED",
	41: "CURLE_FUNCTION_NOT_FOUND",
	42: "CURLE_ABORTED_BY_CALLBACK",
	43: "CURLE_BAD_FUNCTION_ARGUMENT",
	45: "CURLE_INTERFACE_FAILED",
	47: "CURLE_TOO_MANY_REDIRECTS",
	48: "CURLE_UNKNOWN_OPTION",
	49: "CURLE_TELNET_OPTION_SYNTAX",
	51: "CURLE_PEER_FAILED_VERIFICATION",
	52: "CURLE_GOT_NOTHING",
	53: "CURLE_SSL_ENGINE_NOTFOUND",
	54: "CURLE_SSL_ENGINE_SETFAILED",
	55: "CURLE_SEND_ERROR",
	56: "CURLE_RECV_ERROR",
	58: "CURLE_SSL_CERTPROBLEM",
	59: "CURLE_SSL_CIPHER",
	60: "CURLE_SSL_CACERT",
	61: "CURLE_BAD_CONTENT_ENCODING",
	62: "CURLE_LDAP_INVALID_URL",
	63: "CURLE_FILESIZE_EXCEEDED",
	64: "CURLE_USE_SSL_FAILED",
	65: "CURLE_SEND_FAIL_REWIND",
	66: "CURLE_SSL_ENGINE_INITFAILED",
	67: "CURLE_LOGIN_DENIED",
	68: "CURLE_TFTP_NOTFOUND",
	69: "CURLE_TFTP_PERM",
	70: "CURLE_REMOTE_DISK_FULL",
	71: "CURLE_TFTP_ILLEGAL",
	72: "CURLE_TFTP_UNKNOWNID",
	73: "CURLE_REMOTE_FILE_EXISTS",
	74: "CURLE_TFTP_NOSUCHUSER",
	75: "CURLE_CONV_FAILED",
	76: "CURLE_CONV_REQD",
	77: "CURLE_SSL_CACERT_BADFILE",
	78: "CURLE_REMOTE_FILE_NOT_FOUND",
	79: "CURLE_SSH",
	80: "CURLE_SSL_SHUTDOWN_FAILED",
	81: "CURLE_AGAIN",
	82: "CURLE_SSL_CRL_BADFILE",
	83: "CURLE_SSL_ISSUER_ERROR",
	84: "CURLE_FTP_PRET_FAILED",
	85: "CURLE_RTSP_CSEQ_ERROR",
	86: "CURLE_RTSP_SESSION_ERROR",
	87: "CURLE_FTP_BAD_FILE_LIST",
	88: "CURLE_CHUNK_FAILED",
	89: "CURLE_NO_CONNECTION_AVAILABLE",
	90: "CURLE_SSL_PINNEDPUBKEYNOTMATCH",
	91: "CURLE_SSL_INVALIDCERTSTATUS",
	92: "CURLE_HTTP2_STREAM",
	93: "CURLE_RECURSIVE_API_CALL",
	94: "CURLE_AUTH_ERROR",
	95: "CURLE_HTTP3",
	96: "CURLE_QUIC_CONNECT_ERROR",
}

func runTest(curlPath string, domain tranco.Domain, echGrease bool, maxTime time.Duration) TestResult {
	result := TestResult{
		Domain:    domain.Name,
		Rank:      domain.Rank,
		ECHGrease: echGrease,
	}

	url := "https://" + domain.Name

	// curl -w "dnslookup:%{time_namelookup},tcpconnect:%{time_connect},tlsconnect:%{time_appconnect},servertime:%{time_starttransfer},total:%{time_total},httpstatus:%{http_code}" --head -s --ech grease https://example.com
	args := []string{
		"-w",
		"dnslookup:%{time_namelookup},tcpconnect:%{time_connect},tlsconnect:%{time_appconnect},servertime:%{time_starttransfer},total:%{time_total},httpstatus:%{http_code}",
		"--head",
		"-s", // silent
		"--max-time",
		strconv.FormatFloat(maxTime.Seconds(), 'f', -1, 64),
	}
	if echGrease {
		args = append(args, "--ech", "grease")
	}
	args = append(args, url)

	slog.Debug("running curl", "path", curlPath, "args", args)
	cmd := exec.Command(curlPath, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		result.Error = fmt.Sprintf("failed to run curl: %v, stderr: %s", err, stderr.String())
		if exitError, ok := err.(*exec.ExitError); ok {
			result.CurlExitCode = exitError.ExitCode()
			result.CurlErrorName = curlExitCodeNames[result.CurlExitCode]
		}
		return result
	}

	// parse the output
	// dnslookup:0.001,tcpconnect:0.002,tlsconnect:0.003,servertime:0.004,total:0.005,httpstatus:200
	parts := strings.Split(out.String(), ",")
	for _, part := range parts {
		kv := strings.Split(part, ":")
		if len(kv) != 2 {
			continue
		}
		key := kv[0]
		value := kv[1]

		switch key {
		case "dnslookup":
			f, _ := strconv.ParseFloat(value, 64)
			result.DNSLookup = time.Duration(f * float64(time.Second))
		case "tcpconnect":
			f, _ := strconv.ParseFloat(value, 64)
			result.TCPConnection = time.Duration(f * float64(time.Second))
		case "tlsconnect":
			f, _ := strconv.ParseFloat(value, 64)
			result.TLSHandshake = time.Duration(f * float64(time.Second))
		case "servertime":
			f, _ := strconv.ParseFloat(value, 64)
			result.ServerTime = time.Duration(f * float64(time.Second))
		case "total":
			f, _ := strconv.ParseFloat(value, 64)
			result.TotalTime = time.Duration(f * float64(time.Second))
		case "httpstatus":
			i, _ := strconv.Atoi(value)
			result.HTTPStatus = i
		}
	}

	return result
}

func main() {
	workspaceFlag := flag.String("workspace", "./workspace", "Directory to store intermediate files")
	trancoIDFlag := flag.String("trancoID", "7NZ4X", "Tranco list ID to use")
	topNFlag := flag.Int("topN", 100, "Number of top domains to analyze")
	parallelismFlag := flag.Int("parallelism", 10, "Maximum number of parallel requests")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	maxTimeFlag := flag.Duration("maxTime", 10*time.Second, "Maximum time per curl request")
	curlPathFlag := flag.String("curl", "", "Path to the ECH-enabled curl binary")
	flag.Parse()

	if *verboseFlag {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	// Set up workspace directory.
	workspaceDir := workspace.EnsureWorkspace(*workspaceFlag)

	// Determine curl binary path.
	curlPath := *curlPathFlag
	if curlPath == "" {
		curlPath = filepath.Join(workspaceDir, "output", "bin", "curl")
	}

	// Ensure Tranco list is present.
	trancoList := tranco.NewTrancoList(workspaceDir, *trancoIDFlag)

	// Read top N domains from Tranco CSV.
	domains, err := trancoList.TopDomains(*topNFlag)
	if err != nil {
		slog.Error("Failed to read domains from Tranco CSV", "error", err)
		os.Exit(1)
	}

	// Create new output CSV file.
	outputFilename := filepath.Join(workspaceDir, fmt.Sprintf("grease-results-top%d.csv", *topNFlag))
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		slog.Error("Failed to create output CSV file", "path", outputFilename, "error", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	header := []string{"domain", "rank", "ech_grease", "error", "curl_exit_code", "curl_error_name", "dns_lookup_ms", "tcp_connection_ms", "tls_handshake_ms", "server_time_ms", "total_time_ms", "http_status"}
	if err := csvWriter.Write(header); err != nil {
		slog.Error("Failed to write CSV header", "error", err)
		os.Exit(1)
	}

	resultsCh := make(chan TestResult, 2*(*topNFlag))

	var csvWg sync.WaitGroup
	csvWg.Add(1)
	go func() {
		defer csvWg.Done()
		for result := range resultsCh {
			record := []string{
				result.Domain,
				strconv.Itoa(result.Rank),
				strconv.FormatBool(result.ECHGrease),
				result.Error,
				strconv.Itoa(result.CurlExitCode),
				result.CurlErrorName,
				strconv.FormatInt(result.DNSLookup.Milliseconds(), 10),
				strconv.FormatInt(result.TCPConnection.Milliseconds(), 10),
				strconv.FormatInt(result.TLSHandshake.Milliseconds(), 10),
				strconv.FormatInt(result.ServerTime.Milliseconds(), 10),
				strconv.FormatInt(result.TotalTime.Milliseconds(), 10),
				strconv.Itoa(result.HTTPStatus),
			}
			if err := csvWriter.Write(record); err != nil {
				slog.Error("Failed to write record to CSV", "error", err)
			}
		}
	}()

	sem := semaphore.NewWeighted(int64(*parallelismFlag))
	var wg sync.WaitGroup

	for _, domain := range domains {
		wg.Add(2)
		if err := sem.Acquire(context.Background(), 1); err != nil {
			slog.Error("Failed to acquire semaphore", "domain", domain.Name, "error", err)
			continue
		}
		go func(d tranco.Domain) {
			defer sem.Release(1)
			defer wg.Done()
			slog.Info("Testing domain", "domain", d.Name, "ech_grease", false)
			resultsCh <- runTest(curlPath, d, false, *maxTimeFlag)
		}(domain)

		if err := sem.Acquire(context.Background(), 1); err != nil {
			slog.Error("Failed to acquire semaphore", "domain", domain.Name, "error", err)
			continue
		}
		go func(d tranco.Domain) {
			defer sem.Release(1)
			defer wg.Done()
			slog.Info("Testing domain", "domain", d.Name, "ech_grease", true)
			resultsCh <- runTest(curlPath, d, true, *maxTimeFlag)
		}(domain)
	}

	wg.Wait()
	close(resultsCh)

	csvWg.Wait()

	slog.Info("Done. Results saved to", "path", outputFilename)
}
