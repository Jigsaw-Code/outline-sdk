How to find and query the authoritative nameserver for a record.

1. Query SOA for the target_domain.
1. Parse the ANSWER Section (Find the Destination):
  * Look at the last CNAME record in the list.
  * The record it points to is your Final Destination (e.g., app-site-association.g.aaplimg.com).
  * (If there are no CNAMEs, your Final Destination is just the target_domain).
1. Parse the AUTHORITY Section (Find the Boss):
  * Grab the Owner Name of the SOA record provided here.
  * This is your Authoritative Zone (e.g., g.aaplimg.com).
1. Get the Nameservers:
  * Query NS for that Authoritative Zone.
1. Execute the Final Query:
  * Query one of those nameservers for the HTTPS record of the Final Destination.

Example:

```console
% dig @8.8.8.8 +noall +answer +authority app-site-association.cdn-apple.com SOA
app-site-association.cdn-apple.com. 455 IN CNAME app-site-association.cdn-apple.com.akadns.net.
app-site-association.cdn-apple.com.akadns.net. 1492 IN CNAME app-site-association.g.aaplimg.com.
g.aaplimg.com.          300     IN      SOA     a.gslb.aaplimg.com. hostmaster.apple.com. 1734464414 1800 300 60480 300
```

```console
% dig @8.8.8.8 +noall +answer g.aaplimg.com. NS                             
g.aaplimg.com.          14400   IN      NS      b.gslb.aaplimg.com.
g.aaplimg.com.          14400   IN      NS      a.gslb.aaplimg.com.
g.aaplimg.com.          14400   IN      NS      ns4.g.aaplimg.com.
g.aaplimg.com.          14400   IN      NS      ns1.g.aaplimg.com.
g.aaplimg.com.          14400   IN      NS      ns3.g.aaplimg.com.
g.aaplimg.com.          14400   IN      NS      ns2.g.aaplimg.com.
```

```console
% dig @b.gslb.aaplimg.com. +noall +answer app-site-association.g.aaplimg.com. HTTPS     
app-site-association.g.aaplimg.com. 300 IN HTTPS 1 . key32768="https://doh.dns.apple.com/dns-query"
```
