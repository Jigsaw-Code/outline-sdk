package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-internal-sdk/x/connectivity"
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

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) TestConnectivity(
	accessKey string,
	domain string,
	resolvers []string,
	protocols ConnectivityTestProtocolConfig) ([]ConnectivityTestResult, error) {

	config, err := parseAccessKey(accessKey)
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

		for _, resolverHost := range resolvers {
			resolverHost := strings.TrimSpace(resolverHost)
			resolverAddress := net.JoinHostPort(resolverHost, "53")

			if protocols.Tcp {
				testTime := time.Now()
				var testErr error
				var testDuration time.Duration

				dialer, err := makeStreamDialer(proxyAddress, config.CryptoKey, config.Prefix)
				if err != nil {
					return nil, err
				}

				resolver := &transport.StreamDialerEndpoint{Dialer: dialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverStreamConnectivity(context.Background(), resolver, domain)

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

			if protocols.Udp {
				testTime := time.Now()
				var testErr error
				var testDuration time.Duration

				listener, err := makePacketListener(proxyAddress, config.CryptoKey)
				if err != nil {
					return nil, err
				}

				dialer := transport.PacketListenerDialer{Listener: listener}
				resolver := &transport.PacketDialerEndpoint{Dialer: dialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverPacketConnectivity(context.Background(), resolver, domain)

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
