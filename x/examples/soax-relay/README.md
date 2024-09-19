# SOCKS5 Proxy Relay for SOAX

This program creates a SOCKS5 proxy relay that allows users to share access to SOAX without sharing their own credentials. It's designed for connectivity testing purposes.

## Design

The proxy relay works as follows:

1. It listens for incoming SOCKS5 connections on a specified local address and port.
2. It authenticates incoming connections using a custom authentication mechanism.
3. For authenticated connections, it relays requests to the SOAX proxy server.
4. It supports both TCP and UDP traffic.

#### Key components:

- **CustomAuthenticator**: Handles authentication based on username prefix and extracts the suffix which includes connection parameters such as country, city, ISP, etc.
- **StaticCredentialStore**: Stores and validates user credentials.
- **UDP Associate Handler**: Manages UDP connections by relaying the SOAX bind address back to the client.

## Usage

### Configuration

The program uses a YAML configuration file (`config.yaml`) for settings. Example configuration:

```yaml
server:
  address: "127.0.0.1"
  port: "1080"

upstream:
  prefix: "package-xxxxx-"
  password: "YourSOAXPassword"
  address: "proxy.soax.com:5000"

credentials:
  foo: "bar"
  jane: "password123"

udp_timeout: 1m
```

### Authentication
Users connect to the relay using the following format:

<b>Username:</b> 

username-country-{cc_code}-sessionid-{session_id}-sessionlength-{time_in_sec}

where any part that goes after username-country-{cc_code} is optional. For more information on the format refer to [this document](https://helpcenter.soax.com/en/articles/6723733-sticky-sessions) on soax support page.

<b>Password:</b> (as defined in the credentials section of the config file)

<b>Example Username:</b>

outline-country-ru-sessionid-12-sessionlength-600

The relay will use the SOAX package ID and password to connect to the SOAX server, appending the country, sessionid, and sessionlength from the user's input.

### Setup and Running

Ensure you have Go installed on your system.
Clone the repository:

```bash
git clone <repository-url>
cd <repository-directory>
```

Install dependencies:
```bash
go mod tidy
```

Create a `config.yaml` file in the project directory with your SOAX credentials and desired settings.

Build the program:

```bash
go build -o socks5relay
```

Run the program:

```bash
./socks5relay
```


The relay will start and listen for incoming connections on the specified address and port.

SOAX Sticky Sessions:

This relay supports SOAX sticky sessions. Users can specify:

1. country

Two-letter country code (e.g., "us" for United States)

2. sessionid

A random string to maintain the same IP across requests

3. sessionlength

Duration of the session in seconds (10 to 3600)

4. city

City name. Example: new+york

5. region

Region name. Example: new+york

6. isp

ISP name. Please use this parameter together with the country information. 

For more information on the format for values please check out [this document](https://helpcenter.soax.com/en/articles/6723733-sticky-sessions) on soax support page.

The relay will forward these parameters to SOAX, allowing users to benefit from sticky sessions without direct access to SOAX credentials.

### Security Considerations

This relay is intended for testing purposes only.
Ensure the relay is run in a secure environment, as it handles sensitive SOAX credentials.
Regularly update the credentials in the config file to maintain security.



