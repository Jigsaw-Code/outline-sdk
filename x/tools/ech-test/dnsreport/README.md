# DNS Report Generation

This document outlines the steps to generate a report on DNS query latency and HTTPS RR feature usage. The process involves collecting data, running analysis scripts, and generating a final report.

The final report is `report/report.md` and can be converted to a PDF.

## Step 1: Setup

The analysis scripts are written in Python.

1.  **Create a Python virtual environment:**
    From the `ech-test` directory, run:
    ```sh
    python3 -m venv ./workspace/.venv
    ```

2.  **Activate the virtual environment:**
    ```sh
    source ./workspace/.venv/bin/activate
    ```
    You will need to do this every time you work on the report in a new terminal session.

3.  **Install dependencies:**
    From the `ech-test` directory, run:
    ```sh
    pip install -r ./dnsreport/tools/requirements.txt
    ```

## Step 2: Collect DNS Data

From the `ech-test` folder, run the data collection tool. The following command will query the top 10,000 domains 5 times each, which is a good sample for the report.

```sh
go run ./dnsreport -topN 10000 -numQueries 5
```

This will save the results to `./workspace/results-top10000-n5.csv`.

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
* `run`: The run number of the query.
* `query_type`: The type of DNS query (A, AAAA, HTTPS).
* `timestamp`: When the query was performed (RFC3339Nano).
* `duration_ms`: How long the query took in milliseconds.
* `error`: Any error that occurred during the query.
* `rcode`: The DNS response code (e.g., NoError, NXDomain).
* `cnames`: The CNAME records in the answer section, formatted as a JSON array.
* `answers`: The resource records in the answer section (excluding CNAMEs), formatted as a JSON array.
* `additionals`: The resource records in the additional section, formatted as a JSON array.


## Step 3: Analyze DNS Query Latency

The goal of this step is to determine the impact of waiting for the HTTPS RR before proceeding with TCP or TLS connections. The analysis is broken down into:

*   **Duration Distribution:** We create a cumulative distribution of latencies over all queries, broken down by query type (A/AAAA/HTTPS), to visualize the overall performance.
*   **Impact of Caching:** To consider the effects of caching, we group queries by domain and query type, and analyze the minimum and median durations.
*   **Slowest Queries:** We identify all domains for which the median HTTPS query is > 50ms slower than the A query and put them in a table for detailed inspection.

### Generating the Latency Analysis

The analysis scripts are in `dnsreport/tools`. The generated plots and tables will be placed in `dnsreport/report`.

First, sort the data:
```sh
./workspace/.venv/bin/python3 dnsreport/tools/sort_csv_by_rank.py ./workspace/results-top10000-n5.csv
```
This creates `./workspace/results-top10000-n5-sorted.csv`.

Now, run the analysis scripts.

1.  **Generate Latency Plots:**
    ```sh
    ./workspace/.venv/bin/python3 dnsreport/tools/generate_charts.py ./workspace/results-top10000-n5-sorted.csv ./dnsreport/report
    ```
    **Outputs:** This generates the plots in the `dnsreport/report` directory.
    *   `duration_by_type_quantile_plot.png`: Overall latency distribution.
    *   `min_duration_quantile_plot.png`: Best-case (cached) latency distribution.
    *   `median_duration_quantile_plot.png`: Typical latency distribution.

2.  **Generate Slow Queries Table:**
    ```sh
    ./workspace/.venv/bin/python3 dnsreport/tools/generate_filtered_table.py ./workspace/results-top10000-n5-sorted.csv > ./dnsreport/report/slow_https_queries.md
    ```
    **Output:** This creates `slow_https_queries.md` in the `dnsreport/report` directory.

## Step 4: Analyze HTTPS RR Feature Usage

The goal of this step is to determine what features of the HTTPS RR are being used in production to inform the priority of implementing support for them. The features include the Alias Mode, the various `alpn` values and the various SVCB parameters (e.g., `ipv4hint`, `ipv6hint`, `ech`).

### Generating the Feature Analysis

1.  **Generate Feature Usage Plots:**
    ```sh
    ./workspace/.venv/bin/python3 dnsreport/tools/generate_charts.py ./workspace/results-top10000-n5-sorted.csv ./dnsreport/report
    ./workspace/.venv/bin/python3 dnsreport/tools/unique_domain_analysis.py ./workspace/results-top10000-n5-sorted.csv ./dnsreport/report
    ```
    **Outputs:** This generates the plots in the `dnsreport/report` directory.
    *   `param_usage.png`: Usage frequency of all HTTPS RR parameters.
    *   `param_usage_unique_domains.png`: Parameter usage counted only once per domain.

2.  **Generate Feature Usage Table:**
    ```sh
    ./workspace/.venv/bin/python3 dnsreport/tools/unique_domain_analysis.py ./workspace/results-top10000-n5-sorted.csv ./dnsreport/report > ./dnsreport/report/feature_usage_table.md
    ```
    **Output:** This creates `feature_usage_table.md` in the `dnsreport/report` directory.

## Step 5: Analyze Broken Domains

The goal of this step is to identify domains where HTTPS queries consistently time out (duration > 2s) and investigate the reasons behind these failures by analyzing RCODEs, errors, and querying authoritative nameservers.

### Generating the Broken Domains Analysis

1.  **Run the analysis script:**
    ```sh
    ./workspace/.venv/bin/python3 dnsreport/tools/analyze_broken_domains.py ./workspace/results-top10000-n5-sorted.csv
    ```
    **Output:** This generates `broken_domains_report.md` in the `dnsreport/report` directory.

## Step 6: Assemble the Report

1.  **Fill in the report:**
    Open `dnsreport/report/report.md`. The generated charts are already linked. You can copy the contents of `dnsreport/report/slow_https_queries.md`, `dnsreport/report/feature_usage_table.md`, and `dnsreport/report/broken_domains_report.md` to replace the example tables in the report. Finally, write a conclusion based on the findings.

Remember to `cd ../..` to return to the `ech-test` directory when you are done.
