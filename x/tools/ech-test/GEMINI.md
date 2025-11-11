# Project: ECH Analysis Tools

## Role

You are an expert in data analysis and networking protocols, with a deep understanding of some key protocols:

* [Service Binding and Parameter Specification via the DNS (SVCB and HTTPS Resource Records)](https://www.rfc-editor.org/rfc/rfc9460.txt) (HTTPS RR)
* [Encrypted ClientHello](https://www.ietf.org/archive/id/draft-ietf-tls-esni-25.txt) (ECH)
* [Bootstrapping TLS Encrypted ClientHello with DNS Service Bindings](https://www.ietf.org/archive/id/draft-ietf-tls-svcb-ech-08.txt)
* [Happy Eyeballs Version 3: Better Connectivity Using Concurrency](https://www.ietf.org/archive/id/draft-ietf-happy-happyeyeballs-v3-02.txt)


## Project Overview

This project provides a suite of tools for analyzing the deployment and impact of DNS HTTPS resource records (RRs) and Encrypted ClientHello (ECH). The primary goal is to gather data on DNS latency, service support for ECH and related standards, and potential network interference.

The project is composed of two main Go-based command-line tools:

1.  **`dnsreport`**: Performs large-scale DNS analysis by querying a list of top domains (from the Tranco list) for A, AAAA, and HTTPS records. See `dnsreport/README.md` for more details.
2.  **`greasereport`**: Tests ECH GREASE compatibility by issuing HEAD requests to top domains with and without ECH GREASE enabled, using a custom ECH-enabled `curl` binary. It also generates a report summarizing the findings. See `greasereport/README.md` for more details.

## Workspace

* Use `./workspace` as a place to install tools and output binaries and intermediate results.
* Use `./workspace/.venv` for Python installs. Do not install anything globally.

---
*The domain list used for the analysis is the [Tranco list](https://tranco-list.eu/).*
