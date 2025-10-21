# ECH Analysis Tool

This goal of this project is to better understand the impact of deploying the HTTPS Resource Record (RR) and
Encrypted ClientHello (ECH) in order to inform various tradeoffs.

Some research questions:

* **DNS Latency** - How long does it take to receive the HTTPS RR, and how does that compare to the A and AAAA records?
* **Service Support** - Do services implement the TLS standard correctly, or do they fail with the ECH GREASE extension?
  Do the services supporting ECH correctly implement it? What HTTPS RR features do services support?
* **Network Support** - Do networks block or interfere with ECH?

This project contains two main tools to help answer these questions:

1. A Go program to perform large-scale DNS analysis.
2. A custom `curl` build with ECH support for targeted testing.

---

## DNS Analysis Tool

This tool performs DNS queries (A, AAAA, HTTPS) for a large number of domains from the Tranco list.

To run the tool, use the `go run` command:

```sh
go run github.com/Jigsaw-Code/outline-sdk/x/tools/ech-test --topN 100
```

This will:

1. Create a `./workspace` directory if it doesn't exist.
2. Download the Tranco top 1 million domains list (if not already present).
3. Query the top 100 domains for A, AAAA, and HTTPS records using the `8.8.8.8:53` resolver.
4. Save the results to `./workspace/results-top100.csv`.

### Parameters

* `-workspace <path>`: Directory to store intermediate files. Defaults to `./workspace`.
* `-trancoID <id>`: The ID of the Tranco list to use. Defaults to `7NZ4X`.
* `-topN <number>`: The number of top domains to analyze. Defaults to 100.
* `-parallelism <number>`: Maximum number of parallel requests. Defaults to 100.

### Output Format

The tool generates a CSV file (`workspace/results-top<N>.csv`) with the following columns:

* `timestamp`: When the query was performed (RFC3339Nano).
* `duration_ms`: How long the query took in milliseconds.
* `domain`: The domain that was queried.
* `query_type`: The type of DNS query (A, AAAA, HTTPS).
* `error`: Any error that occurred during the query.
* `rcode`: The DNS response code (e.g., NoError, NXDomain).
* `cnames`: The CNAME records in the answer section, formatted as a JSON array.
* `answers`: The resource records in the answer section (excluding CNAMEs), formatted as a JSON array.
* `additionals`: The resource records in the additional section, formatted as a JSON array.

**Example:**

```csv
timestamp,duration_ms,domain,query_type,error,rcode,cnames,answers,additionals
2025-10-16T12:00:00.123456789Z,50,example.com,A,,NoError,[],"["93.184.216.34"]","[]"
2025-10-16T12:00:00.234567890Z,75,example.com,AAAA,,NoError,[],"["2606:2800:220:1:248:1893:25c8:1946"]","[]"
2025-10-16T12:00:00.456789012Z,150,example.com,HTTPS,,NoError,[],"[{"priority":1,"target":"example.com.","params":{"alpn":["h2","http/1.1"]}}]","[]"
```

---

## ECH-enabled `curl`

This is a custom build of `curl` with ECH support from the [DEfO project](https://github.com/defo-project). It is useful for manually testing ECH functionality against specific web servers.

### Prerequisites (for building on macOS)

* Homebrew
* `automake`
* `libtool`

### Building

If you don't have `automake`:

```sh
brew install automake libtool
```

We will put everything under a `$WORKSPACE_DIR` folder with this structure:

* `openssl/`: the DEfO OpenSSL repository clone
* `curl/`: the DEfO curl repository clone
* `output/`: where we will put all the built output
  * `lib/`: there the libraries are stored in the output

Let's create an env var for our workspace folder:

```sh
export WORKSPACE_DIR="$(pwd)"
```

Clone and build OpenSSL with ECH:

```sh
git clone --filter=blob:none https://github.com/defo-project/openssl
cd openssl
./config --libdir=lib --prefix="${WORKSPACE_DIR}/output"
make -j8
make install_sw
```

Clone and build curl with ECH:

```sh
cd "${WORKSPACE_DIR}"
git clone --filter=blob:none https://github.com/defo-project/curl
cd curl
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

```console
% "${WORKSPACE_DIR}/output/bin/curl" --ech=true --doh-url https://1.1.1.1/dns-query 'https://test.defo.ie/echstat.php?format=json' | jq
{
  "SSL_ECH_OUTER_SNI": "public.test.defo.ie",
  "SSL_ECH_INNER_SNI": "test.defo.ie",
  "SSL_ECH_STATUS": "success",
  "date": "2025-10-07T20:19:18+00:00",
  "config": "min-ng.test.defo.ie"
}
```

---
*The domain list used for the DNS analysis is the [Tranco list](https://tranco-list.eu/).*
