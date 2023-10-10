# Outline Connectivity Test

This app illustrates the use of the Shadowsocks transport to resolve a domain name over TCP or UDP. If report collector is specified, it will send a report to the collector if connectivity error occurs.

Example:
```
# From https://www.reddit.com/r/outlinevpn/wiki/index/prefixing/
KEY=ss://ENCRYPTION_KEY@HOST:PORT/
# Example report collector setup: https://github.com/amircybersec/report-collector
COLLECTOR_URL=https://server-address.com/
for PREFIX in POST%20 HTTP%2F1.1%20 %05%C3%9C_%C3%A0%01%20 %16%03%01%40%00%01 %13%03%03%3F %16%03%03%40%00%02; do
  go run github.com/Jigsaw-Code/outline-sdk/x/examples/outline-connectivity@latest -transport="$KEY?prefix=$PREFIX" -proto tcp -resolver 8.8.8.8 -collector $COLLECTOR_URL && echo Prefix "$PREFIX" works!
done
```
