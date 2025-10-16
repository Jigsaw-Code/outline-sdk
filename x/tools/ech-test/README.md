# ECH Analisys Tool

## Running the DNS analysis

This tool performs DNS queries for the top domains from the Tranco list to check for A, AAAA, and HTTPS records.

To run the tool, use the `go run` command from the `x/` directory:

```sh
go run github.com/Jigsaw-Code/outline-sdk/x/tools/ech-test --topN 100
```

This will:
1. Create a `workspace` directory if it doesn't exist.
2. Download the Tranco top 1 million domains list (if not already present).
3. Query the top 100 domains for A, AAAA, and HTTPS records using the `8.8.8.8:53` resolver.
4. Save the results to `workspace/results-top100.csv`.

### Parameters

* `-workspace <path>`: Directory to store intermediate files. Defaults to `./workspace`.
* `-trancoID <id>`: The ID of the Tranco list to use. Defaults to `7NZ4X`.
* `-topN <number>`: The number of top domains to analyze. Defaults to 100.

## Domain List

We are using the [Tranco list](https://tranco-list.eu/) of top 1 million domains.

We use the October 2, 2025 version with subdomains ([reference](https://tranco-list.eu/list/7NZ4X/1000000)).
You can find the zip file at https://tranco-list.eu/download/daily/tranco_7NZ4X-1m.csv.zip

## Building `curl`

If you don't have automake:

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

To test:

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
