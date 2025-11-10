## `dnsreport`

This tool, located in the `dnsreport/` directory, performs DNS queries (A, AAAA, HTTPS) for a large number of domains from the Tranco list.

To run the tool, use the `go run` command from the `ech-test` directory:

```sh
go run ./dnsreport --topN 100
```

This will:

1. Create a `./workspace` directory if it doesn't exist.
2. Download the Tranco top 1 million domains list (if not already present).
3. Query the top 100 domains for A, AAAA, and HTTPS records using the `8.8.8.8:53` resolver.
4. Save the results to `./workspace/results-top100-n1.csv`.

### Parameters

* `-workspace <path>`: Directory to store intermediate files. Defaults to `./workspace`.
* `-trancoID <id>`: The ID of the Tranco list to use. Defaults to `7NZ4X`.
* `-topN <number>`: The number of top domains to analyze. Defaults to 100.
* `-parallelism <number>`: Maximum number of parallel requests. Defaults to 10.
* `-numQueries <number>`: Number of times to query each domain. Defaults to 1.

### Output Format

The tool generates a CSV file (`workspace/results-top<N>-n<M>.csv`) with the following columns:

* `domain`: The domain that was queried.
* `rank`: The rank of the domain in the Tranco list.
* `query_type`: The type of DNS query (A, AAAA, HTTPS).
* `timestamp`: When the query was performed (RFC3339Nano).
* `duration_ms`: How long the query took in milliseconds.
* `error`: Any error that occurred during the query.
* `rcode`: The DNS response code (e.g., NoError, NXDomain).
* `cnames`: The CNAME records in the answer section, formatted as a JSON array.
* `answers`: The resource records in the answer section (excluding CNAMEs), formatted as a JSON array.
* `additionals`: The resource records in the additional section, formatted as a JSON array.

---

## Plotting Results

Python scripts are available in the `dnstest/report/` directory to generate various plots from the CSV results.

### Prerequisites

- Python 3
- `venv` (usually included with Python)

### Setup

1.  Create a virtual environment:
    ```sh
    python3 -m venv .venv
    ```

2.  Activate the virtual environment:
    ```sh
    source .venv/bin/activate
    ```

3.  Install the required packages:
    ```sh
    pip install pandas seaborn matplotlib
    ```

### Usage

Run the plotting scripts from the `dnsreport/` directory.

**Example:**

```sh
python report/plot_durations.py report/results-top1000.csv report/duration_distribution.png
```

---
*The domain list used for the analysis is the [Tranco list](https://tranco-list.eu/).*
