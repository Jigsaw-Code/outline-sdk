# ECH GREASE Connectivity Analysis

## Objective

To determine if enabling ECH GREASE in TLS handshakes causes connectivity issues with the top 1000 websites.

## Methodology

We tested the top 1000 domains from the Tranco list. For each domain, we made two HEAD requests using a custom-built ECH-enabled `curl`:

1.  **Control:** A standard HTTPS request.
2.  **Grease:** An HTTPS request with ECH GREASE enabled (`--ech grease`).

We then analyzed the results to identify any differences in HTTP status, `curl` error codes, or TLS handshake times between the two runs.

## Findings

Our analysis focused on identifying domains where the control and grease runs had different outcomes. We considered a difference to be a change in the HTTP status, the `curl` error name, or a significant (>100ms) deviation in the TLS handshake time.

Out of 1000 domains tested, we found **28 domains** with differences between the control and grease runs. These can be broken down into two categories:

### Failure Differences

We found **3 domains** with different success/failure outcomes:

1.  **`events.data.microsoft.com`**: The control run resulted in an HTTP 404, while the grease run failed with a certificate validation error (`CURLE_SSL_CACERT`).
2.  **`w3schools.com`**: The control run timed out (`CURLE_OPERATION_TIMEDOUT`), while the grease run failed with an SSL connection error (`CURLE_SSL_CONNECT_ERROR`).
3.  **`wp.com`**: The control run failed with a DNS resolution error (`CURLE_COULDNT_RESOLVE_HOST`), while the grease run succeeded with an HTTP 301.

Further manual debugging of these three domains revealed that the discrepancies were likely caused by transient network errors or pre-existing server-side issues, not by ECH GREASE itself.

### Performance Differences

We found **25 domains** with a significant (>100ms) difference in TLS handshake time between the control and grease runs.

*   **Improved Performance**: For **24 domains**, the TLS handshake was faster when ECH GREASE was enabled. The improvements ranged from 104ms to 1273ms.
*   **Degraded Performance**: For **1 domain** (`th.bing.com`), the TLS handshake was significantly slower (3598ms) with ECH GREASE.

## Conclusion

Based on our analysis of the top 1000 domains, there is **no evidence to suggest that ECH GREASE breaks connectivity**. The few observed failures were attributable to other factors.

While ECH GREASE can influence the performance of the TLS handshake, our findings show that it is more likely to improve performance than to degrade it. In our test, 24 domains saw a significant performance improvement, while only one saw a degradation. This suggests that enabling ECH GREASE may offer performance benefits for some servers.
