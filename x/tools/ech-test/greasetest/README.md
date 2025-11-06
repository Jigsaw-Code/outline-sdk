## `greasetest`

This tool, located in the `greasetest/` directory, issues HEAD requests to a list of domains with and without ECH GREASE to test for compatibility.

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

### Output Format

The tool generates a CSV file (`workspace/grease-results-top<N>.csv`) with the following columns:

* `domain`: The domain that was tested.
* `rank`: The rank of the domain in the Tranco list.
* `ech_grease`: `true` if ECH GREASE was enabled for the request, `false` otherwise.
* `error`: Any error that occurred during the request.
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

If you don't have `automake`, `libtool`, `pkg-config`, or `libpsl`:

```sh
brew install automake libtool pkg-config libpsl
```

We will use the `workspace` directory at the root of this project. Inside it, the structure will be:

* `openssl/`: the DEfO OpenSSL repository clone
* `curl/`: the DEfO curl repository clone
* `output/`: where we will put all the built output
  * `bin/`: there the executables are stored in the output
  * `lib/`: there the libraries are stored in the output

Let's create an env var for our workspace folder:

```sh
export WORKSPACE_DIR="$(pwd)/workspace"
```

Clone and build OpenSSL with ECH:

```sh
git clone --filter=blob:none https://github.com/defo-project/openssl "${WORKSPACE_DIR}/openssl"
cd "${WORKSPACE_DIR}/openssl"
./config --libdir=lib --prefix="${WORKSPACE_DIR}/output"
make -j8
make install_sw
```

Clone and build curl with ECH:

```sh
git clone --filter=blob:none https://github.com/defo-project/curl "${WORKSPACE_DIR}/curl"
cd "${WORKSPACE_DIR}/curl"
autoreconf -fi
./configure --with-openssl="${WORKSPACE_DIR}/output" --prefix="${WORKSPACE_DIR}/output" --enable-ech
make
make install
```

Note: you should see the warning below after the `configure` call. If you don't, something went wrong.

```text
configure: WARNING: ECH is enabled but marked EXPERIMENTAL. Use with caution!
configure: WANING: HTTPSRR is enabled but marked EXPERIMENTAL. Use with caution!
```

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
