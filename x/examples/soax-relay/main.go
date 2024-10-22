// Copyright 2023 Jigsaw Operations LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	csocks5 "github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/spf13/viper"
	"github.com/things-go/go-socks5"
	"github.com/things-go/go-socks5/statute"
)

// CustomAuthenticator handles authentication based on username prefix and extracts the suffix
type CustomAuthenticator struct {
	credentials map[string]string
}

// GetCode implements the `GetCode` method for the `Authenticator` interface
func (a CustomAuthenticator) GetCode() uint8 {
	return statute.MethodUserPassAuth
}

// Authenticate checks if the username matches a required prefix and extracts the suffix
func (a CustomAuthenticator) Authenticate(reader io.Reader, writer io.Writer, userAddr string) (*socks5.AuthContext, error) {
	// Respond to the client to use username/password authentication
	if _, err := writer.Write([]byte{statute.VersionSocks5, statute.MethodUserPassAuth}); err != nil {
		return nil, err
	}

	// Parse the username and password from the client request
	nup, err := statute.ParseUserPassRequest(reader)
	if err != nil {
		return nil, err
	}

	fullUsername := string(nup.User)
	password := string(nup.Pass)

	// Split the username into prefix and suffix parts
	parts := strings.SplitN(fullUsername, "-", 2)
	if len(parts) < 2 {
		if _, err := writer.Write([]byte{statute.UserPassAuthVersion, statute.AuthFailure}); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("username does not contain a valid prefix-suffix format")
	}

	usernamePrefix := parts[0] // Prefix (e.g., "jane")
	usernameSuffix := parts[1] // Suffix (e.g., "residential-country-ir")

	storedPassword, exists := a.credentials[usernamePrefix]
	// Validate the prefix and password against stored credentials
	if !exists || storedPassword != password {
		// Authentication failed
		if _, err := writer.Write([]byte{statute.UserPassAuthVersion, statute.AuthFailure}); err != nil {
			return nil, err
		}
		return nil, statute.ErrUserAuthFailed
	}

	// Authentication successful
	if _, err := writer.Write([]byte{statute.UserPassAuthVersion, statute.AuthSuccess}); err != nil {
		return nil, err
	}

	// Return the context with the extracted suffix for later use
	return &socks5.AuthContext{
		Method: statute.MethodUserPassAuth,
		Payload: map[string]string{
			"suffix": usernameSuffix, // Store the suffix for later use in the dialer
			"prefix": usernamePrefix, // Store the prefix as well, if needed
		},
	}, nil
}

func udpAssociateHandler(ctx context.Context, writer io.Writer, request *socks5.Request, config *Config) error {

	// Extract the suffix from the auth context
	suffix, ok := request.AuthContext.Payload["suffix"]
	if !ok {
		return errors.New("no suffix found in auth context")
	}

	var soaxUsername string
	var soaxKey string
	// Construct the upstream soax username by concatenating
	// the soax package name with the suffix
	if strings.HasPrefix(suffix, "residential") {
		// remove the "residential-" prefix
		soaxUsername = config.SOAX.ResidentialPackageID + strings.TrimPrefix(suffix, "residential-")
		soaxKey = config.SOAX.ResidentialPackageKey
	} else if strings.HasPrefix(suffix, "mobile") {
		// remove the "mobile-" prefix
		soaxUsername = config.SOAX.MobilePackageID + strings.TrimPrefix(suffix, "mobile-")
		soaxKey = config.SOAX.MobilePackageKey
	} else {
		return errors.New("invalid package type: it must start with residential or mobile")
	}

	streamEndpoint := transport.StreamDialerEndpoint{Dialer: &transport.TCPDialer{}, Address: config.SOAX.Address}
	upstreamClient, err := csocks5.NewClient(&streamEndpoint)
	if err != nil {
		return err
	}
	err = upstreamClient.SetCredentials([]byte(soaxUsername), []byte(soaxKey))
	if err != nil {
		return err
	}

	upstreamClient.EnablePacket(&transport.UDPDialer{})
	upstreamConn, upstreamBindAddr, err := upstreamClient.ConnectAndRequest(ctx, csocks5.CmdUDPAssociate, "0.0.0.0:0")
	if err != nil {
		return err
	}
	defer upstreamConn.Close()

	// Start listening on the client UDP address
	clientListenAddr, err := transport.MakeNetAddr("udp", config.Server.UDPListenAddress)
	if err != nil {
		return err
	}
	clientListener, err := net.ListenUDP("udp", clientListenAddr.(*net.UDPAddr))
	if err != nil {
		return err
	}
	defer clientListener.Close()

	// Send the local UDP address back to the client
	if err = socks5.SendReply(writer, statute.RepSuccess, clientListener.LocalAddr()); err != nil {
		return err
	}

	// Create a connection to the upstream server
	upstreamUDPConn, err := net.DialUDP("udp", nil, convertToNetAddr(upstreamBindAddr.IP, upstreamBindAddr.Port))
	if err != nil {
		return err
	}
	defer upstreamUDPConn.Close()

	// Start UDP relay
	errCh := make(chan error, 3)

	// Relay packets between client and upstream
	go relayUDP(clientListener, upstreamUDPConn, errCh)

	// Monitor client TCP connection
	go func() {
		_, _ = io.Copy(io.Discard, request.Reader)
		errCh <- errors.New("client TCP connection closed")
	}()

	// Wait for any error or connection close
	err = <-errCh
	log.Printf("UDP Associate ended: %v", err)
	return nil
}

func relayUDP(clientConn *net.UDPConn, upstreamConn *net.UDPConn, errCh chan<- error) {
	var clientAddr *net.UDPAddr
	var clientAddrMu sync.Mutex

	// Client to upstream
	go func() {
		buf := make([]byte, 64*1024)
		for {
			n, addr, err := clientConn.ReadFromUDP(buf)
			if err != nil {
				errCh <- fmt.Errorf("error reading from client UDP: %v", err)
				return
			}

			clientAddrMu.Lock()
			if clientAddr == nil {
				clientAddr = addr
			}
			clientAddrMu.Unlock()

			_, err = upstreamConn.Write(buf[:n])
			if err != nil {
				errCh <- fmt.Errorf("error writing to upstream UDP: %v", err)
				return
			}
		}
	}()

	// Upstream to client
	go func() {
		buf := make([]byte, 64*1024)
		for {
			n, err := upstreamConn.Read(buf)
			if err != nil {
				errCh <- fmt.Errorf("error reading from upstream UDP: %v", err)
				return
			}

			clientAddrMu.Lock()
			if clientAddr != nil {
				_, err = clientConn.WriteToUDP(buf[:n], clientAddr)
				if err != nil {
					errCh <- fmt.Errorf("error writing to client UDP: %v", err)
					return
				}
			}
			clientAddrMu.Unlock()
		}
	}()
}
func convertToNetAddr(ip netip.Addr, port uint16) *net.UDPAddr {
	return &net.UDPAddr{
		IP:   ip.AsSlice(),
		Port: int(port),
	}
}

type Config struct {
	Server struct {
		TCPListenAddress string `mapstructure:"tcp_listen_address"`
		UDPListenAddress string `mapstructure:"udp_listen_address"`
	}
	SOAX struct {
		MobilePackageID  string `mapstructure:"mobile_package_id"`
		MobilePackageKey string `mapstructure:"mobile_package_key"`
		ResidentialPackageID  string `mapstructure:"residential_package_id"`
		ResidentialPackageKey string `mapstructure:"residential_package_key"`
		Address    string `mapstructure:"address"`
	}
	Credentials map[string]string `mapstructure:"credentials"`
}

func loadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.SOAX.MobilePackageID = "package-" + config.SOAX.MobilePackageID + "-"
	config.SOAX.ResidentialPackageID = "package-" + config.SOAX.ResidentialPackageID + "-"

	if config.SOAX.Address == "" {
		config.SOAX.Address = "proxy.soax.com:5000"
	}

	return &config, nil
}

func main() {

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create the custom authenticator
	customAuth := CustomAuthenticator{
		credentials: config.Credentials,
	}
	streamEndpoint := transport.StreamDialerEndpoint{Dialer: &transport.TCPDialer{}, Address: config.SOAX.Address}

	server := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{customAuth}),
		socks5.WithDialAndRequest(func(ctx context.Context, network, addr string, req *socks5.Request) (net.Conn, error) {
			authContext := req.AuthContext
			if authContext == nil {
				return nil, errors.New("no auth context available")
			}

			suffix, ok := authContext.Payload["suffix"]
			if !ok {
				return nil, errors.New("no suffix found in auth context")
			}

			// Construct the upstream username by concatenating the base username with the suffix
			// if the suffix starts with "residential", use the residential package credentials
			// otherwise, use the mobile package credentials	
			var soaxUsername string
			var soaxKey string
			if strings.HasPrefix(suffix, "residential") {
				// remove the "residential-" prefix
				soaxUsername = config.SOAX.ResidentialPackageID + strings.TrimPrefix(suffix, "residential-")
				soaxKey = config.SOAX.ResidentialPackageKey
			} else if strings.HasPrefix(suffix, "mobile") {
				// remove the "mobile-" prefix
				soaxUsername = config.SOAX.MobilePackageID + strings.TrimPrefix(suffix, "mobile-")
				soaxKey = config.SOAX.MobilePackageKey
			} else {
				return nil, errors.New("invalid package type. it must start with residential or mobile")
			}
			client, err := csocks5.NewClient(&streamEndpoint)
			if err != nil {
				return nil, err
			}
			err = client.SetCredentials([]byte(soaxUsername), []byte(soaxKey))
			if err != nil {
				return nil, err
			}
			return client.DialStream(ctx, addr)
		}),
		socks5.WithAssociateHandle(func(ctx context.Context, writer io.Writer, request *socks5.Request) error {
			return udpAssociateHandler(ctx, writer, request, config)
		}),
	)

	// Run SOCKS5 proxy on the specified address and port
	if err := server.ListenAndServe("tcp", config.Server.TCPListenAddress); err != nil {
		panic(err)
	}

}
