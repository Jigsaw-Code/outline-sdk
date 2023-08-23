package shared_backend

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"

	_ "golang.org/x/mobile/bind"
)

type ConnectivityTestProtocolConfig struct {
	Tcp bool `json:"tcp"`
	Udp bool `json:"udp"`
}

type ConnectivityTestResult struct {
	// Inputs
	Proxy    string `json:"proxy"`
	Resolver string `json:"resolver"`
	Proto    string `json:"proto"`
	Prefix   string `json:"prefix"`
	// Observations
	Time       time.Time              `json:"time"`
	DurationMs int64                  `json:"durationMs"`
	Error      *ConnectivityTestError `json:"error"`
}

type ConnectivityTestError struct {
	// TODO: add Shadowsocks/Transport error
	Op string `json:"operation"`
	// Posix error, when available
	PosixError string `json:"posixError"`
	// TODO: remove IP addresses
	Msg string `json:"message"`
}

type ConnectivityTestInput struct {
	AccessKey string                         `json:"accessKey"`
	Domain    string                         `json:"domain"`
	Resolvers []string                       `json:"resolvers"`
	Protocols ConnectivityTestProtocolConfig `json:"protocols"`
}

type sessionConfig struct {
	Hostname  string
	Port      int
	CryptoKey *shadowsocks.EncryptionKey
	Prefix    Prefix
}

type Prefix []byte

func ConnectivityTest(input ConnectivityTestInput) ([]ConnectivityTestResult, error) {
	config, err := parseAccessKey(input.AccessKey)
	if err != nil {
		return nil, err
	}

	proxyIPs, err := net.DefaultResolver.LookupIP(context.Background(), "ip", config.Hostname)
	if err != nil {
		return nil, err
	}

	// TODO: limit number of IPs. Or force an input IP?
	var results []ConnectivityTestResult
	for _, hostIP := range proxyIPs {
		proxyAddress := net.JoinHostPort(hostIP.String(), fmt.Sprint(config.Port))

		for _, resolverHost := range input.Resolvers {
			resolverHost := strings.TrimSpace(resolverHost)
			resolverAddress := net.JoinHostPort(resolverHost, "53")

			if input.Protocols.Tcp {
				testTime := time.Now()
				var testErr error
				var testDuration time.Duration

				dialer, err := makeStreamDialer(proxyAddress, config.CryptoKey, config.Prefix)
				if err != nil {
					return nil, err
				}

				resolver := &transport.StreamDialerEndpoint{Dialer: dialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverStreamConnectivity(context.Background(), resolver, input.Domain)

				results = append(results, ConnectivityTestResult{
					Proxy:      proxyAddress,
					Resolver:   resolverAddress,
					Proto:      "tcp",
					Prefix:     config.Prefix.String(),
					Time:       testTime.UTC().Truncate(time.Second),
					DurationMs: testDuration.Milliseconds(),
					Error:      makeErrorRecord(testErr),
				})
			}

			if input.Protocols.Udp {
				testTime := time.Now()
				var testErr error
				var testDuration time.Duration

				listener, err := makePacketListener(proxyAddress, config.CryptoKey)
				if err != nil {
					return nil, err
				}

				dialer := transport.PacketListenerDialer{Listener: listener}
				resolver := &transport.PacketDialerEndpoint{Dialer: dialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverPacketConnectivity(context.Background(), resolver, input.Domain)

				results = append(results, ConnectivityTestResult{
					Proxy:      proxyAddress,
					Resolver:   resolverAddress,
					Proto:      "udp",
					Prefix:     config.Prefix.String(),
					Time:       testTime.UTC().Truncate(time.Second),
					DurationMs: testDuration.Milliseconds(),
					Error:      makeErrorRecord(testErr),
				})
			}
		}
	}

	return results, nil
}

func makeStreamDialer(proxyAddress string, cryptoKey *shadowsocks.EncryptionKey, prefix []byte) (transport.StreamDialer, error) {
	proxyDialer, err := shadowsocks.NewStreamDialer(&transport.TCPEndpoint{Address: proxyAddress}, cryptoKey)
	if err != nil {
		return nil, err
	}
	if len(prefix) > 0 {
		proxyDialer.SaltGenerator = shadowsocks.NewPrefixSaltGenerator(prefix)
	}
	return proxyDialer, nil
}

func makePacketListener(proxyAddress string, cryptoKey *shadowsocks.EncryptionKey) (transport.PacketListener, error) {
	return shadowsocks.NewPacketListener(&transport.UDPEndpoint{Address: proxyAddress}, cryptoKey)
}

func makeErrorRecord(err error) *ConnectivityTestError {
	if err == nil {
		return nil
	}
	var record = new(ConnectivityTestError)
	var testErr *connectivity.TestError
	if errors.As(err, &testErr) {
		record.Op = testErr.Op
		record.PosixError = testErr.PosixError
		record.Msg = unwrapAll(testErr).Error()
	} else {
		record.Msg = err.Error()
	}
	return record
}

func unwrapAll(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

func (p Prefix) String() string {
	runes := make([]rune, len(p))
	for i, b := range p {
		runes[i] = rune(b)
	}
	return string(runes)
}

func parseAccessKey(accessKey string) (*sessionConfig, error) {
	var config sessionConfig
	accessKeyURL, err := url.Parse(accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access key: %w", err)
	}
	var portString string
	// Host is a <host>:<port> string
	config.Hostname, portString, err = net.SplitHostPort(accessKeyURL.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint address: %w", err)
	}
	config.Port, err = strconv.Atoi(portString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse port number: %w", err)
	}
	cipherInfoBytes, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(accessKeyURL.User.String())
	if err != nil {
		return nil, fmt.Errorf("failed to decode cipher info [%v]: %v", accessKeyURL.User.String(), err)
	}
	cipherName, secret, found := strings.Cut(string(cipherInfoBytes), ":")
	if !found {
		return nil, fmt.Errorf("invalid cipher info: no ':' separator")
	}
	config.CryptoKey, err = shadowsocks.NewEncryptionKey(cipherName, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	prefixStr := accessKeyURL.Query().Get("prefix")
	if len(prefixStr) > 0 {
		config.Prefix, err = ParseStringPrefix(prefixStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse prefix: %w", err)
		}
	}
	return &config, nil
}

func ParseStringPrefix(utf8Str string) (Prefix, error) {
	runes := []rune(utf8Str)
	rawBytes := make([]byte, len(runes))
	for i, r := range runes {
		if (r & 0xFF) != r {
			return nil, fmt.Errorf("character out of range: %d", r)
		}
		rawBytes[i] = byte(r)
	}
	return rawBytes, nil
}

/* INFRASTRUCTURE (to be generalized via reflection/generics and moved) */
type CallInputMessage struct {
	Method string `json:"method"`
	Input  string `json:"input"`
}

type CallOutputMessage struct {
	Result string   `json:"result"`
	Errors []string `json:"errors"`
}

func SendRawCall(rawInputMessage []byte) []byte {
	var inputMessage CallInputMessage

	parseInputError := json.Unmarshal(rawInputMessage, &inputMessage)

	outputMessage := CallOutputMessage{Result: "", Errors: []string{}}

	if parseInputError != nil {
		outputMessage.Errors = append(outputMessage.Errors, "SendRawCall: error parsing raw input string")
	}

	if inputMessage.Method != "ConnectivityTest" {
		outputMessage.Errors = append(outputMessage.Errors, "SendRawCall: method to call not found")
	}

	var methodInput ConnectivityTestInput

	unmarshallingInputError := json.Unmarshal([]byte(inputMessage.Input), &methodInput)

	if unmarshallingInputError != nil {
		outputMessage.Errors = append(outputMessage.Errors, "SendRawCall: error parsing method input")
	}

	result, testError := ConnectivityTest(methodInput)

	if testError != nil {
		outputMessage.Errors = append(outputMessage.Errors, testError.Error())
	}

	rawResult, marshallingResultError := json.Marshal(result)

	if marshallingResultError != nil {
		outputMessage.Errors = append(outputMessage.Errors, "SendRawCall: error serializing method result")
	}

	outputMessage.Result = string(rawResult)

	rawOutputMessage, marshallingOutputError := json.Marshal(outputMessage)

	if marshallingOutputError != nil {
		fmt.Println("[ERROR] failed to properly marshal SendRawCall output")
	}

	return rawOutputMessage
}
