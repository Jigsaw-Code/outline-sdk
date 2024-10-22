# SOCKS5 Proxy Relay for SOAX

This program creates a SOCKS5 proxy relay that allows users to share access to SOAX without sharing their own credentials. It's designed for connectivity testing purposes.

## Design

The proxy relay works as follows:

1. It listens for incoming SOCKS5 connections on a specified local address and port.
2. It authenticates incoming connections using a custom authentication mechanism.
3. For authenticated connections, it relays requests to the SOAX proxy server.
4. It supports both TCP and UDP traffic.

## Usage

### Configuration

The program uses a YAML configuration file (`config.yaml`) for settings. Please refer to the example config file.

### Authentication
Users connect to the relay using the following format:

<b>Username:</b> 

username-{residential, mobile}-country-{cc_code}-sessionid-{session_id}-sessionlength-{time_in_sec}

where any part that goes after username-country-{cc_code} is optional. For more information on the format refer to [this document](https://helpcenter.soax.com/en/articles/6723733-sticky-sessions) on soax support page. To get a node on residential or mobile nework, use the residential- or mobile- tags.

<b>Password:</b> (as defined in the credentials section of the config file)

<b>Example Username:</b>

`outline-country-ru-sessionid-12-sessionlength-600` where `outline` is the username and parameters that goes after configured the soax proxy

The relay will use the SOAX package ID and password to connect to the SOAX server, appending the country, sessionid, and sessionlength from the user's input.

### Setup and Running

Ensure you have Go installed on your system.

Create a `config.yaml` file in the project directory with your SOAX credentials and desired settings.

```bash
go run github.com/Jigsaw-Code/outline-sdk/x/examples/soax-relay@latest
```

The relay will start and listen for incoming connections on the specified address and port.

### SOAX Sticky Sessions

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

For more information on the format for values please check out [this document](https://helpcenter.soax.com/en/articles/6723733-sticky-sessions) on soax support page. Here's a list of supported ISPs for Iran and Russia. For ISPs with space in name, please replace space with `%20`:

Russia Mobile ISPs:
```
["pjsc megafon","mts pjsc","bee line cable","tele2 russia","rostelecom","tbank jsc","t-mob","ekaterinburg-2000","edinos","tattelecom","jsc vainah telecom","s.u.e. dpr republic operator of networks","sberbank-telecom","k-telekom","llc sp abaza telecom","mcs","edinos ltd.","jv a-mobile","ozyorsk telecom cjsc.","mobile trend","innovation solutions center ltd.","zao aquafon-gsm","isp balzer-telecom","invest mobile","dom.ru","ooo vtc-mobile","mts ojsc","main radio meteorological centre (mrmc)","novokuznetsk telecom"]
```

Iran Mobile ISPs:
```
["mobile communication company of iran","mtn irancell","rightel communication service company pjs","aria shatel pjsc","rightel","pardis fanvari partak","aria shatel company ltd"]
```

The relay will forward these parameters to SOAX, allowing users to benefit from sticky sessions without direct access to SOAX credentials.

### Security Considerations

- This relay is intended for testing purposes only.
- Ensure the relay is run in a secure environment, as it handles sensitive SOAX credentials.
- Regularly update the credentials in the config file to maintain security.
- SOCKS5 traffic is not encrypted.



