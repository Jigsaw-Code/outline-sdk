# Project: ECH Analysis Tools

## Project Overview

This project provides a suite of tools for analyzing the deployment and impact of DNS HTTPS resource records (RRs) and Encrypted ClientHello (ECH). The primary goal is to gather data on DNS latency, service support for ECH and related standards, and potential network interference.

The project is composed of two main Go-based command-line tools:

1.  **`dnsreport`**: Performs large-scale DNS analysis by querying a list of top domains (from the Tranco list) for A, AAAA, and HTTPS records. See `dnsreport/README.md` for more details.
2.  **`greasereport`**: Tests ECH GREASE compatibility by issuing HEAD requests to top domains with and without ECH GREASE enabled, using a custom ECH-enabled `curl` binary. It also generates a report summarizing the findings. See `greasereport/README.md` for more details.

## Workspace

* Use `./workspace` as a place to install tools and output binaries and intermediate results.
* Use `./workspace/.venv` for Python installs. Do not install anything globally.
