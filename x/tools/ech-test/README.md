# ECH Analysis Tools

This project contains tools to better understand the impact of deploying the HTTPS Resource Record (RR) and
Encrypted ClientHello (ECH) in order to inform various tradeoffs.

Some research questions:

* **DNS Latency** - How long does it take to receive the HTTPS RR, and how does that compare to the A and AAAA records?
* **Service Support** - Do services implement the TLS standard correctly, or do they fail with the ECH GREASE extension?
  Do the services supporting ECH correctly implement it? What HTTPS RR features do services support?
* **Network Support** - Do networks block or interfere with ECH?

This project contains two main tools to help answer these questions:

1.  [`dnstest`](./dnstest): A Go program to perform large-scale DNS analysis.
2.  [`greasetest`](./greasetest): A Go program to test ECH GREASE compatibility with top websites.

See the `README.md` file in each tool's directory for more information.
