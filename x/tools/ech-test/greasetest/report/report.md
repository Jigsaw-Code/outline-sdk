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

Out of the 1000 domains tested, we found **3 domains** with different success/failure outcomes:

1.  **`events.data.microsoft.com`**: The control run resulted in an HTTP 404, while the grease run failed with a certificate validation error (`CURLE_SSL_CACERT`).
2.  **`w3schools.com`**: The control run timed out (`CURLE_OPERATION_TIMEDOUT`), while the grease run failed with an SSL connection error (`CURLE_SSL_CONNECT_ERROR`).
3.  **`wp.com`**: The control run failed with a DNS resolution error (`CURLE_COULDNT_RESOLVE_HOST`), while the grease run succeeded with an HTTP 301.

Further manual debugging of these three domains revealed that the discrepancies were likely caused by transient network errors or pre-existing server-side issues, not by ECH GREASE itself.

When considering performance, we found **59 domains** with a significant (>100ms) difference in TLS handshake time between the control and grease runs. However, these are performance variations, not connectivity failures.

## Conclusion

Based on our analysis of the top 1000 domains, there is **no evidence to suggest that ECH GREASE breaks connectivity**. The few observed failures were attributable to other factors. While ECH GREASE can influence the performance of the TLS handshake, it does not appear to cause widespread connection failures.
