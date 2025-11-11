# Broken Domains Analysis Report

This report details domains where HTTPS queries consistently timed out (> 2s).

## Domain: nih.gov

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS nih.gov
ns.nih.gov.
ns2.nih.gov.
ns3.nih.gov.
```

#### Querying dig +timeout=5 +tries=1 +short NS nih.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS nih.gov nih.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS nih.gov', 'nih.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS nih.gov': not found
```

#### Querying ns.nih.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns.nih.gov. nih.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns.nih.gov.', 'nih.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.nih.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.nih.gov. nih.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.nih.gov.', 'nih.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.nih.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.nih.gov. nih.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.nih.gov.', 'nih.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: pubmed.ncbi.nlm.nih.gov

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS pubmed.ncbi.nlm.nih.gov

```

#### Querying dig +timeout=5 +tries=1 +short NS pubmed.ncbi.nlm.nih.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS pubmed.ncbi.nlm.nih.gov pubmed.ncbi.nlm.nih.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS pubmed.ncbi.nlm.nih.gov', 'pubmed.ncbi.nlm.nih.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS pubmed.ncbi.nlm.nih.gov': not found
```

## Domain: usda.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS usda.gov
ns1.usda.gov.
ns2.usda.gov.
```

#### Querying dig +timeout=5 +tries=1 +short NS usda.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS usda.gov usda.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS usda.gov', 'usda.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS usda.gov': not found
```

#### Querying ns1.usda.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.usda.gov. usda.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.usda.gov.', 'usda.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.usda.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.usda.gov. usda.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.usda.gov.', 'usda.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: absher.sa

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS absher.sa
ns1.absher.gov.sa.
ns2.absher.gov.sa.
```

#### Querying dig +timeout=5 +tries=1 +short NS absher.sa for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS absher.sa absher.sa HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS absher.sa', 'absher.sa', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS absher.sa': not found
```

#### Querying ns1.absher.gov.sa. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.absher.gov.sa. absher.sa HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.absher.gov.sa.', 'absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.absher.gov.sa. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.absher.gov.sa. absher.sa HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.absher.gov.sa.', 'absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: www.absher.sa

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS www.absher.sa
absher.sa.
ns2.absher.gov.sa.
ns1.absher.gov.sa.
```

#### Querying dig +timeout=5 +tries=1 +short NS www.absher.sa for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS www.absher.sa www.absher.sa HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS www.absher.sa', 'www.absher.sa', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS www.absher.sa': not found
```

#### Querying absher.sa. for HTTPS record
```bash
dig +timeout=5 +tries=1 @absher.sa. www.absher.sa HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@absher.sa.', 'www.absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.absher.gov.sa. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.absher.gov.sa. www.absher.sa HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.absher.gov.sa.', 'www.absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.absher.gov.sa. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.absher.gov.sa. www.absher.sa HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.absher.gov.sa.', 'www.absher.sa', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: jino.ru

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS jino.ru
ns4.jino.ru.
ns2.jino.ru.
ns3.jino.ru.
ns1.jino.ru.
```

#### Querying dig +timeout=5 +tries=1 +short NS jino.ru for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS jino.ru jino.ru HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS jino.ru', 'jino.ru', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS jino.ru': not found
```

#### Querying ns4.jino.ru. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns4.jino.ru. jino.ru HTTPS
; <<>> DiG 9.20.9 <<>> +timeout=5 +tries=1 @ns4.jino.ru. jino.ru HTTPS
; (2 servers found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 14808
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 121 msec
;; SERVER: 2001:1bb0:e000:1e::1cd#53(ns4.jino.ru.) (UDP)
;; WHEN: Tue Nov 11 00:26:48 EST 2025
;; MSG SIZE  rcvd: 87
```

#### Querying ns2.jino.ru. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.jino.ru. jino.ru HTTPS
; <<>> DiG 9.20.9 <<>> +timeout=5 +tries=1 @ns2.jino.ru. jino.ru HTTPS
; (2 servers found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 30178
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 121 msec
;; SERVER: 2001:1bb0:e000:1e::917#53(ns2.jino.ru.) (UDP)
;; WHEN: Tue Nov 11 00:26:49 EST 2025
;; MSG SIZE  rcvd: 87
```

#### Querying ns3.jino.ru. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.jino.ru. jino.ru HTTPS
; <<>> DiG 9.20.9 <<>> +timeout=5 +tries=1 @ns3.jino.ru. jino.ru HTTPS
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 12888
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 131 msec
;; SERVER: 217.107.219.170#53(ns3.jino.ru.) (UDP)
;; WHEN: Tue Nov 11 00:26:53 EST 2025
;; MSG SIZE  rcvd: 87
```

#### Querying ns1.jino.ru. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.jino.ru. jino.ru HTTPS
; <<>> DiG 9.20.9 <<>> +timeout=5 +tries=1 @ns1.jino.ru. jino.ru HTTPS
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 45428
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;jino.ru.			IN	HTTPS

;; AUTHORITY SECTION:
jino.ru.		86400	IN	SOA	ns1.jino.ru. postmaster.jino.ru. 2014043157 28800 7200 604800 86400

;; Query time: 113 msec
;; SERVER: 217.107.34.200#53(ns1.jino.ru.) (UDP)
;; WHEN: Tue Nov 11 00:26:53 EST 2025
;; MSG SIZE  rcvd: 87
```

## Domain: census.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS census.gov
ns2e.census.gov.
ns1e.census.gov.
```

#### Querying dig +timeout=5 +tries=1 +short NS census.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS census.gov census.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS census.gov', 'census.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS census.gov': not found
```

#### Querying ns2e.census.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2e.census.gov. census.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2e.census.gov.', 'census.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1e.census.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1e.census.gov. census.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1e.census.gov.', 'census.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: pushy.io

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS pushy.io
ns2.pushy.me.
ns1.pushy.me.
ns3.pushy.me.
ns4.pushy.me.
```

#### Querying dig +timeout=5 +tries=1 +short NS pushy.io for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS pushy.io pushy.io HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS pushy.io', 'pushy.io', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS pushy.io': not found
```

#### Querying ns2.pushy.me. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.pushy.me. pushy.io HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.pushy.me. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.pushy.me. pushy.io HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.pushy.me. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.pushy.me. pushy.io HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns4.pushy.me. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns4.pushy.me. pushy.io HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns4.pushy.me.', 'pushy.io', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: cancer.gov

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR, SERVFAIL
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS cancer.gov
ns3.nih.gov.
ns.nih.gov.
ns2.nih.gov.
```

#### Querying dig +timeout=5 +tries=1 +short NS cancer.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS cancer.gov cancer.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS cancer.gov', 'cancer.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS cancer.gov': not found
```

#### Querying ns3.nih.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.nih.gov. cancer.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.nih.gov.', 'cancer.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns.nih.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns.nih.gov. cancer.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns.nih.gov.', 'cancer.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.nih.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.nih.gov. cancer.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.nih.gov.', 'cancer.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: novell.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS novell.com
ns2.softwaregrp.com.
ns1.softwaregrp.com.
ns3.softwaregrp.com.
```

#### Querying dig +timeout=5 +tries=1 +short NS novell.com for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS novell.com novell.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS novell.com', 'novell.com', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS novell.com': not found
```

#### Querying ns2.softwaregrp.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.softwaregrp.com. novell.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.softwaregrp.com.', 'novell.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.softwaregrp.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.softwaregrp.com. novell.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.softwaregrp.com.', 'novell.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.softwaregrp.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.softwaregrp.com. novell.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.softwaregrp.com.', 'novell.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: nyc.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS nyc.gov
ns4.nyc.gov.
ns3.nyc.gov.
ns2.nyc.gov.
ns1.nyc.gov.
```

#### Querying dig +timeout=5 +tries=1 +short NS nyc.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS nyc.gov nyc.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS nyc.gov', 'nyc.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS nyc.gov': not found
```

#### Querying ns4.nyc.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns4.nyc.gov. nyc.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns4.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.nyc.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.nyc.gov. nyc.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.nyc.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.nyc.gov. nyc.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.nyc.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.nyc.gov. nyc.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.nyc.gov.', 'nyc.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: iastate.edu

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS iastate.edu
dns-1.iastate.edu.
dns-3.iastate.edu.
dns-2.iastate.edu.
```

#### Querying dig +timeout=5 +tries=1 +short NS iastate.edu for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS iastate.edu iastate.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS iastate.edu', 'iastate.edu', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS iastate.edu': not found
```

#### Querying dns-1.iastate.edu. for HTTPS record
```bash
dig +timeout=5 +tries=1 @dns-1.iastate.edu. iastate.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dns-1.iastate.edu.', 'iastate.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying dns-3.iastate.edu. for HTTPS record
```bash
dig +timeout=5 +tries=1 @dns-3.iastate.edu. iastate.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dns-3.iastate.edu.', 'iastate.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying dns-2.iastate.edu. for HTTPS record
```bash
dig +timeout=5 +tries=1 @dns-2.iastate.edu. iastate.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dns-2.iastate.edu.', 'iastate.edu', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: southwest.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS southwest.com
ns-sdc.southwest.com.
ns-w11.southwest.com.
```

#### Querying dig +timeout=5 +tries=1 +short NS southwest.com for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS southwest.com southwest.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS southwest.com', 'southwest.com', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS southwest.com': not found
```

#### Querying ns-sdc.southwest.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns-sdc.southwest.com. southwest.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns-sdc.southwest.com.', 'southwest.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns-w11.southwest.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns-w11.southwest.com. southwest.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns-w11.southwest.com.', 'southwest.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: microfocus.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS microfocus.com
ns1.softwaregrp.com.
ns2.softwaregrp.com.
ns3.softwaregrp.com.
```

#### Querying dig +timeout=5 +tries=1 +short NS microfocus.com for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS microfocus.com microfocus.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS microfocus.com', 'microfocus.com', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS microfocus.com': not found
```

#### Querying ns1.softwaregrp.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.softwaregrp.com. microfocus.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.softwaregrp.com.', 'microfocus.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.softwaregrp.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.softwaregrp.com. microfocus.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.softwaregrp.com.', 'microfocus.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.softwaregrp.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.softwaregrp.com. microfocus.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.softwaregrp.com.', 'microfocus.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: miibeian.gov.cn

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS miibeian.gov.cn
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '+short', 'NS', 'miibeian.gov.cn']' returned non-zero exit status 9.

```

#### Querying dig +timeout=5 +tries=1 +short NS miibeian.gov.cn for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS miibeian.gov.cn miibeian.gov.cn HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS miibeian.gov.cn', 'miibeian.gov.cn', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS miibeian.gov.cn': not found
```

## Domain: dtvce.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS dtvce.com
ns3.dtvce.com.
ns1.dtvce.com.
```

#### Querying dig +timeout=5 +tries=1 +short NS dtvce.com for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS dtvce.com dtvce.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS dtvce.com', 'dtvce.com', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS dtvce.com': not found
```

#### Querying ns3.dtvce.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.dtvce.com. dtvce.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.dtvce.com.', 'dtvce.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.dtvce.com. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.dtvce.com. dtvce.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.dtvce.com.', 'dtvce.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: telefonica.com

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS telefonica.com
dns2.movistar.es.
dns1.movistar.es.
```

#### Querying dig +timeout=5 +tries=1 +short NS telefonica.com for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS telefonica.com telefonica.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS telefonica.com', 'telefonica.com', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS telefonica.com': not found
```

#### Querying dns2.movistar.es. for HTTPS record
```bash
dig +timeout=5 +tries=1 @dns2.movistar.es. telefonica.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dns2.movistar.es.', 'telefonica.com', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying dns1.movistar.es. for HTTPS record
```bash
dig +timeout=5 +tries=1 @dns1.movistar.es. telefonica.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dns1.movistar.es.', 'telefonica.com', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: ct.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS ct.gov
ns1.cen.ct.gov.
ns2.cen.ct.gov.
```

#### Querying dig +timeout=5 +tries=1 +short NS ct.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS ct.gov ct.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS ct.gov', 'ct.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS ct.gov': not found
```

#### Querying ns1.cen.ct.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.cen.ct.gov. ct.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.cen.ct.gov.', 'ct.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.cen.ct.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.cen.ct.gov. ct.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.cen.ct.gov.', 'ct.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: utah.gov

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS utah.gov
ns1.utah.gov.
ns2.utah.gov.
```

#### Querying dig +timeout=5 +tries=1 +short NS utah.gov for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS utah.gov utah.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS utah.gov', 'utah.gov', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS utah.gov': not found
```

#### Querying ns1.utah.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.utah.gov. utah.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.utah.gov.', 'utah.gov', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.utah.gov. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.utah.gov. utah.gov HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.utah.gov.', 'utah.gov', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: golux.com

### Initial Analysis (from CSV data)

- **RCODEs:** NOERROR
- **Errors:** query failed: ETIMEDOUT

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS golux.com
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '+short', 'NS', 'golux.com']' returned non-zero exit status 9.

```

#### Querying dig +timeout=5 +tries=1 +short NS golux.com for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS golux.com golux.com HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS golux.com', 'golux.com', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS golux.com': not found
```

## Domain: caf.fr

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS caf.fr
ns.caf.fr.
ns3.caf.fr.
ns1.caf.fr.
```

#### Querying dig +timeout=5 +tries=1 +short NS caf.fr for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS caf.fr caf.fr HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS caf.fr', 'caf.fr', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS caf.fr': not found
```

#### Querying ns.caf.fr. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns.caf.fr. caf.fr HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns.caf.fr.', 'caf.fr', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns3.caf.fr. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns3.caf.fr. caf.fr HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns3.caf.fr.', 'caf.fr', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.caf.fr. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.caf.fr. caf.fr HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.caf.fr.', 'caf.fr', 'HTTPS']' returned non-zero exit status 9.

```

## Domain: unm.edu

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS unm.edu
pdnsbox.id.nmt.edu.
ns1.unm.edu.
ns2.unm.edu.
externaldns0.nmt.edu.
```

#### Querying dig +timeout=5 +tries=1 +short NS unm.edu for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS unm.edu unm.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS unm.edu', 'unm.edu', 'HTTPS']' returned non-zero exit status 1.
dig: couldn't get address for 'dig +timeout=5 +tries=1 +short NS unm.edu': not found
```

#### Querying pdnsbox.id.nmt.edu. for HTTPS record
```bash
dig +timeout=5 +tries=1 @pdnsbox.id.nmt.edu. unm.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@pdnsbox.id.nmt.edu.', 'unm.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns1.unm.edu. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns1.unm.edu. unm.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns1.unm.edu.', 'unm.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying ns2.unm.edu. for HTTPS record
```bash
dig +timeout=5 +tries=1 @ns2.unm.edu. unm.edu HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@ns2.unm.edu.', 'unm.edu', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying externaldns0.nmt.edu. for HTTPS record
```bash
dig +timeout=5 +tries=1 @externaldns0.nmt.edu. unm.edu HTTPS
; <<>> DiG 9.20.9 <<>> +timeout=5 +tries=1 @externaldns0.nmt.edu. unm.edu HTTPS
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: REFUSED, id: 23708
;; flags: qr rd; QUERY: 1, ANSWER: 0, AUTHORITY: 0, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;unm.edu.			IN	HTTPS

;; Query time: 61 msec
;; SERVER: 129.138.4.63#53(externaldns0.nmt.edu.) (UDP)
;; WHEN: Tue Nov 11 00:31:53 EST 2025
;; MSG SIZE  rcvd: 36
```

## Domain: globe.com.ph

### Initial Analysis (from CSV data)

- **RCODEs:** SERVFAIL
- **Errors:** N/A

### Authoritative DNS Investigation

#### Authoritative Nameservers (dig +short NS)
```bash
dig +timeout=5 +tries=1 +short NS globe.com.ph
g-net.globe.com.ph.
dns1.globenet.com.ph.
g-net1.globe.com.ph.
sec.globe.com.ph.
```

#### Querying dig +timeout=5 +tries=1 +short NS globe.com.ph for HTTPS record
```bash
dig +timeout=5 +tries=1 @dig +timeout=5 +tries=1 +short NS globe.com.ph globe.com.ph HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dig +timeout=5 +tries=1 +short NS globe.com.ph', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying g-net.globe.com.ph. for HTTPS record
```bash
dig +timeout=5 +tries=1 @g-net.globe.com.ph. globe.com.ph HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@g-net.globe.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying dns1.globenet.com.ph. for HTTPS record
```bash
dig +timeout=5 +tries=1 @dns1.globenet.com.ph. globe.com.ph HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@dns1.globenet.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying g-net1.globe.com.ph. for HTTPS record
```bash
dig +timeout=5 +tries=1 @g-net1.globe.com.ph. globe.com.ph HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@g-net1.globe.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

#### Querying sec.globe.com.ph. for HTTPS record
```bash
dig +timeout=5 +tries=1 @sec.globe.com.ph. globe.com.ph HTTPS
Error executing command: Command '['dig', '+timeout=5', '+tries=1', '@sec.globe.com.ph.', 'globe.com.ph', 'HTTPS']' returned non-zero exit status 9.

```

