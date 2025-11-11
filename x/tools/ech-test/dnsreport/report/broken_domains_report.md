# Broken Domains Analysis Report

This report details domains where HTTPS queries consistently timed out (> 2s).

## Domain: absher.sa

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns1.absher.gov.sa.
ns2.absher.gov.sa.
```

#### Querying ns1.absher.gov.sa. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.absher.gov.sa.', 'absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.absher.gov.sa. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.absher.gov.sa.', 'absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: caf.fr

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns.caf.fr.
ns3.caf.fr.
ns1.caf.fr.
```

#### Querying ns.caf.fr. for HTTPS record
```
Error executing command: Command '['dig', '@ns.caf.fr.', 'caf.fr', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.caf.fr. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.caf.fr.', 'caf.fr', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.caf.fr. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.caf.fr.', 'caf.fr', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: cancer.gov

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns.nih.gov.
ns2.nih.gov.
ns3.nih.gov.
```

#### Querying ns.nih.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns.nih.gov.', 'cancer.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.nih.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.nih.gov.', 'cancer.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.nih.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.nih.gov.', 'cancer.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: census.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns2e.census.gov.
ns1e.census.gov.
```

#### Querying ns2e.census.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns2e.census.gov.', 'census.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1e.census.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns1e.census.gov.', 'census.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: ct.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns2.cen.ct.gov.
ns1.cen.ct.gov.
```

#### Querying ns2.cen.ct.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.cen.ct.gov.', 'ct.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.cen.ct.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.cen.ct.gov.', 'ct.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: dtvce.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns3.dtvce.com.
ns1.dtvce.com.
```

#### Querying ns3.dtvce.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.dtvce.com.', 'dtvce.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.dtvce.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.dtvce.com.', 'dtvce.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: globe.com.ph

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
dns1.globenet.com.ph.
g-net.globe.com.ph.
sec.globe.com.ph.
g-net1.globe.com.ph.
```

#### Querying dns1.globenet.com.ph. for HTTPS record
```
Error executing command: Command '['dig', '@dns1.globenet.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying g-net.globe.com.ph. for HTTPS record
```
Error executing command: Command '['dig', '@g-net.globe.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying sec.globe.com.ph. for HTTPS record
```
Error executing command: Command '['dig', '@sec.globe.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying g-net1.globe.com.ph. for HTTPS record
```
Error executing command: Command '['dig', '@g-net1.globe.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: golux.com

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
Error executing command: Command '['dig', '+short', 'NS', 'golux.com']' returned non-zero exit status 9.

```

#### Querying Error executing command: Command '['dig', '+short', 'NS', 'golux.com']' returned non-zero exit status 9. for HTTPS record
```
Error executing command: Command '['dig', "@Error executing command: Command '['dig', '+short', 'NS', 'golux.com']' returned non-zero exit status 9.", 'golux.com', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'Error executing command: Command '['dig', '+short', 'NS', 'golux.com']' returned non-zero exit status 9.': not found
```

## Domain: iastate.edu

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
dns-1.iastate.edu.
dns-2.iastate.edu.
dns-3.iastate.edu.
```

#### Querying dns-1.iastate.edu. for HTTPS record
```
Error executing command: Command '['dig', '@dns-1.iastate.edu.', 'iastate.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying dns-2.iastate.edu. for HTTPS record
```
Error executing command: Command '['dig', '@dns-2.iastate.edu.', 'iastate.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying dns-3.iastate.edu. for HTTPS record
```
Error executing command: Command '['dig', '@dns-3.iastate.edu.', 'iastate.edu', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: jino.ru

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns4.jino.ru.
ns1.jino.ru.
ns2.jino.ru.
ns3.jino.ru.
```

#### Querying ns4.jino.ru. for HTTPS record
```
; <<>> DiG 9.20.9 <<>> @ns4.jino.ru. jino.ru HTTPS
; (2 servers found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 46982
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 116 msec
;; SERVER: 2001:1bb0:e000:1e::1cd#53(ns4.jino.ru.) (UDP)
;; WHEN: Mon Nov 10 20:44:43 EST 2025
;; MSG SIZE  rcvd: 87
```

#### Querying ns1.jino.ru. for HTTPS record
```
; <<>> DiG 9.20.9 <<>> @ns1.jino.ru. jino.ru HTTPS
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 5347
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 134 msec
;; SERVER: 217.107.34.200#53(ns1.jino.ru.) (UDP)
;; WHEN: Mon Nov 10 20:44:46 EST 2025
;; MSG SIZE  rcvd: 87
```

#### Querying ns2.jino.ru. for HTTPS record
```
; <<>> DiG 9.20.9 <<>> @ns2.jino.ru. jino.ru HTTPS
; (2 servers found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 10347
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 141 msec
;; SERVER: 2001:1bb0:e000:1e::917#53(ns2.jino.ru.) (UDP)
;; WHEN: Mon Nov 10 20:44:47 EST 2025
;; MSG SIZE  rcvd: 87
```

#### Querying ns3.jino.ru. for HTTPS record
```
; <<>> DiG 9.20.9 <<>> @ns3.jino.ru. jino.ru HTTPS
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 47118
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 136 msec
;; SERVER: 217.107.219.170#53(ns3.jino.ru.) (UDP)
;; WHEN: Mon Nov 10 20:44:47 EST 2025
;; MSG SIZE  rcvd: 87
```

## Domain: microfocus.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns3.softwaregrp.com.
ns2.softwaregrp.com.
ns1.softwaregrp.com.
```

#### Querying ns3.softwaregrp.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.softwaregrp.com.', 'microfocus.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.softwaregrp.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.softwaregrp.com.', 'microfocus.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.softwaregrp.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.softwaregrp.com.', 'microfocus.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: miibeian.gov.cn

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
Error executing command: Command '['dig', '+short', 'NS', 'miibeian.gov.cn']' returned non-zero exit status 9.

```

#### Querying Error executing command: Command '['dig', '+short', 'NS', 'miibeian.gov.cn']' returned non-zero exit status 9. for HTTPS record
```
Error executing command: Command '['dig', "@Error executing command: Command '['dig', '+short', 'NS', 'miibeian.gov.cn']' returned non-zero exit status 9.", 'miibeian.gov.cn', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'Error executing command: Command '['dig', '+short', 'NS', 'miibeian.gov.cn']' returned non-zero exit status 9.': not found
```

## Domain: nih.gov

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns3.nih.gov.
ns.nih.gov.
ns2.nih.gov.
```

#### Querying ns3.nih.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.nih.gov.', 'nih.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns.nih.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns.nih.gov.', 'nih.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.nih.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.nih.gov.', 'nih.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: novell.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns1.softwaregrp.com.
ns3.softwaregrp.com.
ns2.softwaregrp.com.
```

#### Querying ns1.softwaregrp.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.softwaregrp.com.', 'novell.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.softwaregrp.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.softwaregrp.com.', 'novell.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.softwaregrp.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.softwaregrp.com.', 'novell.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: nyc.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns1.nyc.gov.
ns4.nyc.gov.
ns3.nyc.gov.
ns2.nyc.gov.
```

#### Querying ns1.nyc.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns4.nyc.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns4.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.nyc.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.nyc.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: pubmed.ncbi.nlm.nih.gov

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```

```

**No authoritative nameservers found.**

## Domain: pushy.io

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns4.pushy.me.
ns1.pushy.me.
ns3.pushy.me.
ns2.pushy.me.
```

#### Querying ns4.pushy.me. for HTTPS record
```
Error executing command: Command '['dig', '@ns4.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.pushy.me. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.pushy.me. for HTTPS record
```
Error executing command: Command '['dig', '@ns3.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.pushy.me. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: southwest.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns-w11.southwest.com.
ns-sdc.southwest.com.
```

#### Querying ns-w11.southwest.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns-w11.southwest.com.', 'southwest.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns-sdc.southwest.com. for HTTPS record
```
Error executing command: Command '['dig', '@ns-sdc.southwest.com.', 'southwest.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: telefonica.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
dns1.movistar.es.
dns2.movistar.es.
```

#### Querying dns1.movistar.es. for HTTPS record
```
Error executing command: Command '['dig', '@dns1.movistar.es.', 'telefonica.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying dns2.movistar.es. for HTTPS record
```
Error executing command: Command '['dig', '@dns2.movistar.es.', 'telefonica.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: unm.edu

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
externaldns0.nmt.edu.
pdnsbox.id.nmt.edu.
ns1.unm.edu.
ns2.unm.edu.
```

#### Querying externaldns0.nmt.edu. for HTTPS record
```
; <<>> DiG 9.20.9 <<>> @externaldns0.nmt.edu. unm.edu HTTPS
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: REFUSED, id: 11601
;; flags: qr rd; QUERY: 1, ANSWER: 0, AUTHORITY: 0, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;unm.edu.			IN	HTTPS

;; Query time: 91 msec
;; SERVER: 129.138.4.63#53(externaldns0.nmt.edu.) (UDP)
;; WHEN: Mon Nov 10 20:51:10 EST 2025
;; MSG SIZE  rcvd: 36
```

#### Querying pdnsbox.id.nmt.edu. for HTTPS record
```
Error executing command: Command '['dig', '@pdnsbox.id.nmt.edu.', 'unm.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.unm.edu. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.unm.edu.', 'unm.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.unm.edu. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.unm.edu.', 'unm.edu', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: usda.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns1.usda.gov.
ns2.usda.gov.
```

#### Querying ns1.usda.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.usda.gov.', 'usda.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.usda.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.usda.gov.', 'usda.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: utah.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
ns2.utah.gov.
ns1.utah.gov.
```

#### Querying ns2.utah.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.utah.gov.', 'utah.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.utah.gov. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.utah.gov.', 'utah.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: www.absher.sa

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```
absher.sa.
ns1.absher.gov.sa.
ns2.absher.gov.sa.
```

#### Querying absher.sa. for HTTPS record
```
Error executing command: Command '['dig', '@absher.sa.', 'www.absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.absher.gov.sa. for HTTPS record
```
Error executing command: Command '['dig', '@ns1.absher.gov.sa.', 'www.absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.absher.gov.sa. for HTTPS record
```
Error executing command: Command '['dig', '@ns2.absher.gov.sa.', 'www.absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

