# DNS Report

The DNS report is done in a few steps:

1. Collect DNS queries
2. Analyze DNS query latency
3. Analyze HTTPS RR feature usage


## Step 1 - Collect DNS Queries.

From the `ech-test` folder, run:

```sh
go run ./dnsreport -topN 10000 -numQueries 5
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


## Step 2 - Analyze DNS Query Latency

The goal of this step is to determine the impact of waiting for the HTTPS RR before proceeding with TCP or TLS connections.

### Duration sitribution Analysis
We need a cumulative distribution of latencies over all queries, broken down by query type (A/AAAA/HTTPS). X is cummulative probability, Y is duration.

### Impact of Caching
To consider effects of caching, we will group the queries by domain and query type, and take the minimum and median. We will generate the same chart for those metrics.

### Analysis of Slowest Queries.

We should identify all domains for which the median HTTPS query is > 50ms slower than the A query and put them in a table.
There should be 1 column for each of the 5 runs, sorted from fasted to slower. The leftmost is the min, the middle the median, the last the max. The cells whould have the HTTPS query duration, with the difference to the A query in parenthesis (example: "20 (+2)). Cells where the Difference is > +50ms should be in bold.


## Step 3 - Determine HTTPS RR feature usage

The goal of this step is to determine what features of the HTTPS RR are being used in production to inform
the priority of implementing support for them.

The features include the Alias Mode, the various alpn values and the various SVCB parameters (based on their keys).
For example: ["AliasMode", "alpn:h2", "alpn:h3", "ipv4hint", "ipv6hint", "ech"]
