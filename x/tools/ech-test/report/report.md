# Performance Analysis of HTTPS Resource Records for ECH Implementations

**Date:** 2025-10-29

**Author:** [Vinicius Fortuna](https://github.com/fortuna)

## 1. Executive Summary

This report analyzes the performance characteristics of DNS HTTPS resource records (RRs) to inform the implementation strategy for Encrypted ClientHello (ECH) on various libraries and platforms. Our analysis of the top 1,000 domains from the Tranco list, based on five separate runs to ensure robustness, reveals that while the majority of HTTPS queries are performant, a significant minority of domains exhibit high latency, which could negatively impact user experience if the system strictly waits for the HTTPS RR.

**Key Findings:**

*   **Majority of queries are fast:** The median latency for HTTPS queries is low and comparable to A and AAAA records.
*   **A long tail of slow queries:** A small percentage of HTTPS queries are significantly slower, with some taking several seconds or timing out.
*   **Geographic patterns:** Domains with Russian (`.ru`) and Chinese (`.cn`) ccTLDs are disproportionately represented in the set of slow domains.
*   **Root causes of slowness:** The primary causes for the high latency are server-side issues, including timeouts and `SERVFAIL` errors, rather than the inherent size or complexity of HTTPS RRs.

**Recommendation:**

We recommend a hybrid approach for the ECH implementation. Instead of always waiting for the HTTPS RR, the system should race the HTTPS query against the A/AAAA queries with a short timeout (e.g., 50-100ms). If the HTTPS query is not resolved within the timeout, the system should proceed with the standard TLS handshake without ECH. This approach balances the security and privacy benefits of ECH with the need for a fast and reliable user experience.

## 2. Introduction

The introduction of the HTTPS resource record is a critical step for the wide-scale deployment of Encrypted ClientHello (ECH). ECH is a new TLS extension that encrypts the ClientHello message, preventing network observers from snooping on the server name indication (SNI) and other sensitive information.

For ECH to work, the client needs to fetch the ECH configuration from the DNS, which is contained in the HTTPS RR. This introduces a potential performance trade-off: should the client always wait for the HTTPS RR, as mandated by the ECH standard, or should it proceed without it if it's too slow, to avoid harming the user experience?

This report analyzes the performance of HTTPS RR queries based on a dataset of the top 1,000 domains from the Tranco list, collected over five separate runs to ensure the robustness of the results. The goal is to provide data-driven recommendations for ECH implementations.

## 3. Performance Analysis of HTTPS Resource Records

### 3.1. Overall Latency Comparison

The following chart shows a quantile plot of the DNS query durations for A, AAAA, and HTTPS records. This chart illustrates the cumulative distribution of query latencies based on the combined data from three runs.

![Quantile Plot of DNS Query Durations by Type (5 runs)](./quantile_plot.png)

As we can see, for the vast majority of queries (up to the ~0.85 quantile), the latency of HTTPS queries is very close to that of A and AAAA queries. However, beyond this point, the latency of HTTPS queries starts to increase significantly, forming a long tail of slow queries.

### 3.2. HTTPS Query Performance Deep Dive

To better understand the performance of HTTPS queries, let's look at the distribution of their durations.

![Distribution of HTTPS Query Durations (5 runs)](./https_duration_histogram.png)

The histogram shows that the vast majority of HTTPS queries are resolved in under 200ms, with a large concentration in the 0-100ms range. This confirms that in the common case, HTTPS queries are fast. The long tail of the distribution, however, confirms the presence of a significant number of slow queries.

### 3.3. Impact of Geographic Location

We analyzed the difference in duration between HTTPS and A queries, broken down by country-code top-level domain (ccTLD). TLDs that are not country-specific are grouped as "other".

![Distribution of Duration Difference (HTTPS - A) by TLD Category (5 runs)](./duration_diff_by_tld.png)

This chart reveals a few interesting patterns. While the "other" category, containing gTLDs, has the widest distribution due to major outliers, some ccTLDs also show significant variation. Notably, the `.kz` (Kazakhstan) and `.nz` (New Zealand) ccTLDs show a significantly higher median duration difference compared to other ccTLDs. The `.jp` (Japan) and `.su` (Soviet Union) also have a higher median. While `.ru` (Russia) and `.cn` (China) have a wide distribution of duration differences, their median values are closer to the bulk of other ccTLDs, suggesting that while there are slow domains in those regions, the typical performance is not as poor as the outliers might suggest.

### 3.4. Analysis of Slow Queries

A closer look at the slowest HTTPS queries reveals that the high latency is often not due to the size or complexity of the HTTPS record itself, but rather to server-side issues. The most common causes for extreme latency are:

*   **Timeouts:** The DNS query simply times out after several seconds.
*   **`SERVFAIL` errors:** The authoritative DNS server is unable to process the query and returns a `SERVFAIL` error, but only after a long delay.

These issues point to a lack of proper support for the HTTPS RR on some authoritative DNS servers, rather than a fundamental performance problem with the HTTPS RR itself.

### 3.5. Latency vs. Answer Presence

![Latency vs. Answer Presence](./latency_vs_answer.png)

The box plot above compares the duration of HTTPS queries that received an answer against those that did not. Interestingly, the median latency for queries *with* an answer is slightly higher than for those without. However, the distribution for queries with an answer is much tighter, with fewer extreme outliers. This suggests that while there's a small, consistent cost to retrieving the HTTPS record, the major latency issues are more strongly associated with servers that fail to respond correctly to HTTPS queries (i.e., those that don't return an answer).

### 3.6. HTTPS RR Feature Usage

There are **82 unique domains** in our dataset that have an HTTPS RR.

![HTTPS RR Parameter Usage (Unique Domains)](./param_usage_unique_domains.png)

The bar chart and table below show the frequency of different parameters found in the HTTPS RRs that were successfully retrieved, counted by unique domains.

| Parameter | Unique Domains |
|:---|:---|
| Total HTTPS RR Support | 82 |
| alpn | 78 |
| ipv4hint | 60 |
| ipv6hint | 44 |
| ech | 4 |

This analysis shows that:

*   **82 unique domains** have HTTPS RR support.
*   **78 unique domains** use the `alpn` parameter.
*   **60 unique domains** provide an `ipv4hint`.
*   **44 unique domains** provide an `ipv6hint`.
*   **4 unique domains** have an `ech` parameter in their HTTPS RR.

This corrected data gives a much clearer view of the landscape. The number of domains supporting ECH is still small, as expected for a new standard, but it's now accurately represented as a count of unique domains.


## 4. Recommendations for ECH Implementations

Based on our analysis, we recommend a **hybrid approach** for the implementations. A strict implementation that always waits for the HTTPS RR would lead to a poor user experience for a noticeable minority of domains.

Our recommendation is to **race the HTTPS query against the A/AAAA queries with a short timeout**. Here's how it would work:

1.  When a new connection is initiated, the client sends A, AAAA, and HTTPS queries in parallel.
2.  The client waits for a short period (e.g., 50-100ms) for the HTTPS query to complete.
3.  **If the HTTPS query completes within the timeout:** The client uses the ECH configuration from the HTTPS RR to establish an ECH-enabled connection.
4.  **If the HTTPS query does not complete within the timeout:** The client proceeds with the standard TLS handshake using the IP addresses from the A/AAAA records, without ECH.

This approach has several advantages:

*   **Prioritizes user experience:** It avoids long delays for the user when a server has a slow or broken HTTPS RR implementation.
*   **Enables ECH for the majority:** For the vast majority of domains where the HTTPS RR is fast, ECH will be used, providing its security and privacy benefits.
*   **Graceful degradation:** It allows the system to gracefully fall back to standard TLS when ECH is not available or too slow.

## 5. Conclusion

The HTTPS resource record is a critical component for the future of a more private and secure internet with ECH. While our analysis shows that there is a long tail of slow HTTPS queries, these are caused by a minority of misconfigured or slow servers. By implementing a hybrid approach that races the HTTPS query with a short timeout, the client can reap the benefits of ECH without compromising on user experience.

## 6. Appendix: Slowest Domains by HTTPS/A Ratio (5 runs, median, diff > 50ms)

| Domain | Median A Duration (ms) | Median HTTPS Duration (ms) | Median Ratio (HTTPS/A) |
|:---|:---|:---|:---|
| pubmed.ncbi.nlm.nih.gov | 8 | 3019 | 377.38 |
| nih.gov | 14 | 5000 | 357.14 |
| samsungapps.com | 13 | 214 | 16.46 |
| chinamobile.com | 16 | 255 | 15.94 |
| jomodns.com | 15 | 225 | 15.00 |
| beian.miit.gov.cn | 19 | 247 | 13.00 |
| shifen.com | 19 | 243 | 12.79 |
| ksyuncdn.com | 20 | 243 | 12.15 |
| ks-cdn.com | 20 | 235 | 11.75 |
| yahoo.co.jp | 15 | 167 | 11.13 |
| uol.com.br | 13 | 137 | 10.54 |
| kaspi.kz | 19 | 183 | 9.63 |
| mikrotik.com | 14 | 127 | 9.07 |
| rakuten.co.jp | 19 | 163 | 8.58 |
| wbbasket.ru | 16 | 130 | 8.12 |
| consultant.ru | 17 | 133 | 7.82 |
| reg.ru | 17 | 132 | 7.76 |
| 2gis.com | 19 | 138 | 7.26 |
| wp.pl | 17 | 123 | 7.24 |
| rbc.ru | 19 | 129 | 6.79 |
| rambler.ru | 20 | 135 | 6.75 |
| cdnvideo.ru | 20 | 130 | 6.50 |
| t-online.de | 19 | 123 | 6.47 |
| myfritz.net | 19 | 120 | 6.32 |
| vkuser.net | 21 | 124 | 5.90 |
| pool.ntp.org | 16 | 92 | 5.75 |
| nease.net | 18 | 103 | 5.72 |
| taobao.com | 20 | 108 | 5.40 |
| intel.com | 16 | 85 | 5.31 |
| gandi.net | 19 | 100 | 5.26 |
| betweendigital.com | 18 | 94 | 5.22 |
| mediatek.com | 18 | 88 | 4.89 |
| netease.com | 24 | 104 | 4.33 |
