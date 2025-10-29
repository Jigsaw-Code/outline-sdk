# Performance Analysis of HTTPS Resource Records for ECH Implementations

**Date:** 2025-10-29

**Author:** Vinicius Fortuna

## 1. Executive Summary

This report analyzes the performance characteristics of DNS HTTPS resource records (RRs) to inform the implementation strategy for Encrypted ClientHello (ECH) on various libraries and platforms. Our analysis of the top 1,000 domains from the Tranco list, based on three separate runs to ensure robustness, reveals that while the majority of HTTPS queries are performant, a significant minority of domains exhibit high latency, which could negatively impact user experience if the system strictly waits for the HTTPS RR.

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

This report analyzes the performance of HTTPS RR queries based on a dataset of the top 1,000 domains from the Tranco list, collected over three separate runs to ensure the robustness of the results. The goal is to provide data-driven recommendations for ECH implementations.

## 3. Performance Analysis of HTTPS Resource Records

### 3.1. Overall Latency Comparison

The following chart shows a quantile plot of the DNS query durations for A, AAAA, and HTTPS records. This chart illustrates the cumulative distribution of query latencies based on the combined data from three runs.

![Quantile Plot of DNS Query Durations by Type (3 runs)](./quantile_plot.png)

As we can see, for the vast majority of queries (up to the ~0.85 quantile), the latency of HTTPS queries is very close to that of A and AAAA queries. However, beyond this point, the latency of HTTPS queries starts to increase significantly, forming a long tail of slow queries.

### 3.2. HTTPS Query Performance Deep Dive

To better understand the performance of HTTPS queries, let's look at the distribution of their durations.

![Distribution of HTTPS Query Durations (3 runs)](./https_duration_histogram.png)

The histogram shows that the vast majority of HTTPS queries are resolved in under 200ms, with a large concentration in the 0-100ms range. This confirms that in the common case, HTTPS queries are fast. The long tail of the distribution, however, confirms the presence of a significant number of slow queries.

### 3.3. Impact of Geographic Location

We analyzed the difference in duration between HTTPS and A queries, broken down by country-code top-level domain (ccTLD). TLDs that are not country-specific are grouped as "other".

![Distribution of Duration Difference (HTTPS - A) by TLD Category (3 runs)](./duration_diff_by_tld.png)

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
| alpn | 78 |
| ipv4hint | 60 |
| ipv6hint | 44 |
| ech | 4 |

This analysis shows that:

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

## 6. Appendix: Top 100 Slowest Domains by HTTPS/A Ratio (3 runs, averaged)

| Domain | Avg A Duration (ms) | Avg HTTPS Duration (ms) | Avg Ratio (HTTPS/A) |
|:---|:---|:---|:---|
| nih.gov | 12.67 | 4341.67 | 342.76 |
| pubmed.ncbi.nlm.nih.gov | 11.00 | 3024.00 | 274.91 |
| duckdns.org | 37.33 | 1701.67 | 45.58 |
| samsungapps.com | 13.00 | 211.67 | 16.28 |
| one.one | 11.33 | 171.67 | 15.15 |
| intel.com | 12.67 | 141.00 | 11.13 |
| reg.ru | 12.33 | 137.00 | 11.11 |
| kaspi.kz | 12.00 | 131.33 | 10.94 |
| hp.com | 9.33 | 89.00 | 9.54 |
| userapi.com | 13.00 | 123.67 | 9.51 |
| ozon.ru | 17.00 | 140.67 | 8.27 |
| rakuten.co.jp | 21.00 | 168.00 | 8.00 |
| line.me | 15.00 | 115.33 | 7.69 |
| wildberries.ru | 17.67 | 135.00 | 7.64 |
| vkontakte.ru | 12.33 | 91.67 | 7.43 |
| sberbank.ru | 18.00 | 128.33 | 7.13 |
| mikrotik.com | 13.67 | 97.33 | 7.12 |
| myfritz.net | 17.67 | 123.00 | 6.96 |
| unesco.org | 12.67 | 85.67 | 6.76 |
| baidu.com | 13.00 | 86.67 | 6.67 |
| kaspersky.com | 20.67 | 135.00 | 6.53 |
| timeweb.ru | 20.67 | 131.67 | 6.37 |
| mega.co.nz | 12.33 | 77.00 | 6.24 |
| mts.ru | 18.67 | 116.33 | 6.23 |
| uol.com.br | 17.33 | 101.33 | 5.85 |
| free.fr | 17.33 | 100.00 | 5.77 |
| launchpad.net | 18.00 | 96.67 | 5.37 |
| vedcdnlb.com | 21.00 | 110.00 | 5.24 |
| dailymotion.com | 13.33 | 69.67 | 5.23 |
| nrdp-ipv6.prod.ftl.netflix.com | 10.00 | 46.67 | 4.67 |
| nflxso.net | 10.67 | 47.00 | 4.41 |
| alidns.com | 24.33 | 101.67 | 4.18 |
| ieee.org | 10.33 | 43.00 | 4.16 |
| mi.com | 20.33 | 83.00 | 4.08 |
| liftoff.io | 10.67 | 43.00 | 4.03 |
| globo.com | 14.33 | 56.33 | 3.93 |
| cpanel.net | 11.33 | 44.33 | 3.91 |
| grammarly.io | 11.33 | 43.67 | 3.85 |
| dns.cn | 30.00 | 112.33 | 3.74 |
| akamaiedge.net | 14.00 | 49.67 | 3.55 |
| kaspersky-labs.com | 11.00 | 38.33 | 3.48 |
| bidr.io | 12.33 | 41.00 | 3.32 |
| arubanetworks.com | 13.00 | 43.00 | 3.31 |
| jsdelivr.net | 16.33 | 49.67 | 3.04 |
| mediatek.com | 40.33 | 122.33 | 3.03 |
| rambler.ru | 56.33 | 164.33 | 2.92 |
| ks-cdn.com | 163.67 | 475.67 | 2.91 |
| netease.com | 60.00 | 169.00 | 2.82 |
| tiktokcdn.com | 10.33 | 29.00 | 2.81 |
| us-v20.events.data.microsoft.com | 11.33 | 31.00 | 2.74 |
| wiley.com | 19.33 | 52.67 | 2.72 |
| inner-active.mobi | 11.33 | 30.67 | 2.71 |
| dnspod.net | 94.00 | 251.67 | 2.68 |
| gandi.net | 16.67 | 44.00 | 2.64 |
| node.e2ro.com | 16.33 | 42.67 | 2.61 |
| adobe.io | 12.67 | 33.00 | 2.61 |
| nbcnews.com | 14.67 | 38.00 | 2.59 |
| netangels.ru | 64.67 | 165.00 | 2.55 |
| everesttech.net | 12.33 | 31.33 | 2.54 |
| gamepass.com | 9.67 | 24.33 | 2.52 |
| pv-cdn.net | 9.67 | 24.00 | 2.48 |
| anydesk.com | 18.00 | 44.67 | 2.48 |
| att.net | 16.33 | 40.33 | 2.47 |
| bamgrid.com | 18.67 | 46.00 | 2.46 |
| amazon.dev | 9.00 | 22.00 | 2.44 |
| configservice.wyzecam.com | 19.67 | 48.00 | 2.44 |
| telekom.de | 48.33 | 116.00 | 2.40 |
| firefox.com | 9.33 | 22.33 | 2.39 |
| debian.org | 13.00 | 31.00 | 2.38 |
| data.microsoft.com | 11.33 | 27.00 | 2.38 |
| www.gov.uk | 15.33 | 36.33 | 2.37 |
| samsungcloud.com | 8.67 | 20.33 | 2.35 |
| twimg.com | 12.67 | 29.67 | 2.34 |
| open.spotify.com | 11.67 | 27.00 | 2.31 |
| crwdcntrl.net | 8.67 | 20.00 | 2.31 |
| capcutapi.com | 13.00 | 30.00 | 2.31 |
| rbc.ru | 56.33 | 129.67 | 2.30 |
| samsungqbe.com | 12.33 | 28.33 | 2.30 |
| tiktokcdn-us.com | 11.33 | 26.00 | 2.29 |
| sharepoint.com | 12.33 | 28.00 | 2.27 |
| iso.org | 7.67 | 17.33 | 2.26 |
| ttvnw.net | 11.67 | 26.33 | 2.26 |
| sc-cdn.net | 11.67 | 26.33 | 2.26 |
| scribd.com | 12.00 | 27.00 | 2.25 |
| betweendigital.com | 41.33 | 93.00 | 2.25 |
| rzone.de | 18.33 | 41.00 | 2.24 |
| appcenter.ms | 8.67 | 19.33 | 2.23 |
| cloud.microsoft | 9.00 | 20.00 | 2.22 |
| cloudapp.azure.com | 13.67 | 30.33 | 2.22 |
| registrar-servers.com | 21.33 | 47.33 | 2.22 |
| visualstudio.com | 11.67 | 25.67 | 2.20 |
| t-online.de | 56.67 | 124.67 | 2.20 |
| windows.net | 10.33 | 22.67 | 2.19 |
| ksyuncdn.com | 173.00 | 377.67 | 2.18 |
| avcdn.net | 9.33 | 20.00 | 2.14 |
| who.int | 59.33 | 127.00 | 2.14 |
| berkeley.edu | 20.00 | 42.67 | 2.13 |
| lemonde.fr | 17.67 | 37.67 | 2.13 |
| cloudfront.net | 9.00 | 19.00 | 2.11 |
| live.net | 9.33 | 19.67 | 2.11 |
