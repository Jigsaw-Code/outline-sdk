## `greasetest`

This tool, located in the `greasetest/` directory, issues HEAD requests to a list of domains with and without ECH GREASE to test for compatibility. It uses a custom-built ECH-enabled `curl` binary for these requests.

To run the tool, use the `go run` command from the `ech-test` directory:

```sh
go run ./greasetest --topN 100
```

This will:

1. Create a `./workspace` directory if it doesn't exist.
2. Download the Tranco top 1 million domains list (if not already present).
3. Issue HEAD requests to the top 100 domains, once with ECH GREASE and once without.
4. Save the results to `./workspace/grease-results-top100.csv`.

### Parameters

* `-workspace <path>`: Directory to store intermediate files. Defaults to `./workspace`.
* `-trancoID <id>`: The ID of the Tranco list to use. Defaults to `7NZ4X`.
* `-topN <number>`: The number of top domains to analyze. Defaults to 100.
* `-parallelism <number>`: Maximum number of parallel requests. Defaults to 10.
* `-curl <path>`: Path to the ECH-enabled curl binary. Defaults to `./workspace/output/bin/curl`.
* `-maxTime <duration>`: Maximum time per curl request. Defaults to `10s`.

### Output Format

The tool generates a CSV file (`workspace/grease-results-top<N>.csv`) with the following columns:

* `domain`: The domain that was tested.
* `rank`: The rank of the domain in the Tranco list.
* `ech_grease`: `true` if ECH GREASE was enabled for the request, `false` otherwise.
* `error`: Any error that occurred during the request.
* `curl_exit_code`: The exit code returned by the `curl` command.
* `curl_error_name`: The human-readable name corresponding to the `curl` exit code.
* `dns_lookup_ms`: The duration of the DNS lookup.
* `tcp_connection_ms`: The duration of the TCP connection.
* `tls_handshake_ms`: The duration of the TLS handshake.
* `server_time_ms`: The time from the end of the TLS handshake to the first byte of the response.
* `total_time_ms`: The total duration of the request.
* `http_status`: The HTTP status code of the response.

---

## ECH-enabled `curl`

This is a custom build of `curl` with ECH support from the [DEfO project](https://github.com/defo-project). It is useful for manually testing ECH functionality against specific web servers.

### Prerequisites (for building on macOS)

* Homebrew
* `automake`
* `libtool`
* `pkg-config`
* `libpsl`

### Building

A helper script, `build-curl.sh`, is provided to automate the build process for `curl` and its dependency, `openssl`.

To build the ECH-enabled `curl`, run the script from the `greasetest` directory and provide an output path:

```sh
./build-curl.sh <output_directory>
```

For example, to build `curl` and place the output in the `workspace` directory:

```sh
./build-curl.sh ../workspace
```

The script will download the source code for `openssl` and `curl`, build them, and install the final binaries in the specified output directory.

For more details on how to use `curl` with ECH, see the [official documentation](https://github.com/defo-project/curl/blob/master/docs/ECH.md).

### Verifying the build

To test that your custom `curl` build is working correctly, run it against the DEfO test server:

```sh
"$(pwd)/workspace/output/bin/curl" --ech=true --doh-url https://1.1.1.1/dns-query 'https://test.defo.ie/echstat.php?format=json' | jq
```

Example output:
```json
{
  "SSL_ECH_OUTER_SNI": "public.test.defo.ie",
  "SSL_ECH_INNER_SNI": "test.defo.ie",
  "SSL_ECH_STATUS": "success",
  "date": "2025-11-06T19:36:47+00:00",
  "config": "min-ng.test.defo.ie"
}
```

### Report

After running the `greasetest` tool, a `report` subdirectory is created within the `greasetest` directory. This directory contains:

*   `report.md`: A summary of the ECH GREASE connectivity analysis.
*   `analyze.py`: The Python script used for the analysis.
*   `grease-results-top<N>.csv`: The raw data from the test run.
