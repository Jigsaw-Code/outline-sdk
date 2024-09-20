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
	"time"

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
	usernameSuffix := parts[1] // Suffix (e.g., "-country-ir")

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

	// Construct the upstream soax username by concatenating
	// the soax package name with the suffix
	soaxUsername := config.SOAX.PackageID + suffix
	streamEndpoint := transport.StreamDialerEndpoint{Dialer: &transport.TCPDialer{}, Address: config.SOAX.Address}
	client, err := csocks5.NewClient(&streamEndpoint)
	if err != nil {
		return err
	}

	err = client.SetCredentials([]byte(soaxUsername), []byte(config.SOAX.PackageKey))
	if err != nil {
		return err
	}

	client.EnablePacket(&transport.UDPDialer{})

	conn, bindAddr, err := client.ConnectAndRequest(ctx, csocks5.CmdUDPAssociate, "0.0.0.0:0")
	if err != nil {
		return err
	}

	// Start a goroutine to monitor the client's TCP connection.
	// When the client's TCP connection is closed,
	// close the upstream TCP connection as well.
	go func() {
		_, _ = io.Copy(io.Discard, request.Reader)
		// let's see if this fixes the udp issue on remote server
		time.Sleep(5 * time.Second)
		conn.Close()
	}()

	if err = socks5.SendReply(writer, statute.RepSuccess, convertToNetAddr("udp", bindAddr.IP, bindAddr.Port)); err != nil {
		return err
	}

	return nil
}

func convertToNetAddr(network string, ip netip.Addr, port uint16) net.Addr {
	// Convert netip.Addr to net.IP
	netIP := ip.AsSlice()
	switch network {
	case "tcp", "tcp4", "tcp6":
		// Create a net.TCPAddr
		tcpAddr := &net.TCPAddr{
			IP:   netIP,
			Port: int(port),
		}
		return tcpAddr
	case "udp", "udp4", "udp6":
		// Create a net.UDPAddr
		udpAddr := &net.UDPAddr{
			IP:   netIP,
			Port: int(port),
		}
		return udpAddr
	}
	return nil
}

type Config struct {
	Server struct {
		ListenAddress string `mapstructure:"listen_address"`
	}
	SOAX struct {
		PackageID  string `mapstructure:"package_id"`
		PackageKey string `mapstructure:"package_key"`
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

	config.SOAX.PackageID = "package-" + config.SOAX.PackageID + "-"

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
	client, err := csocks5.NewClient(&streamEndpoint)
	if err != nil {
		return
	}

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
			soaxUsername := config.SOAX.PackageID + suffix // e.g., "package-123456-country-ir-seesionid-..."
			err = client.SetCredentials([]byte(soaxUsername), []byte(config.SOAX.PackageKey))
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
	if err := server.ListenAndServe("tcp", config.Server.ListenAddress); err != nil {
		panic(err)
	}

}
