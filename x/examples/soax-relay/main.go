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
	"os"
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
	PrefixCredentials CredentialStore // Store for validating the prefix part of the username
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

	log.Printf("Attempting authentication with prefix: %s and suffix: %s", usernamePrefix, usernameSuffix)

	// Validate the prefix and password against stored credentials
	if !a.PrefixCredentials.Valid(usernamePrefix, password, userAddr) {
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

// CredentialStore is an interface to validate prefix-based credentials
type CredentialStore interface {
	Valid(username, password, userAddr string) bool
}

// StaticCredentialStore is a simple credential store that holds a map of valid prefixes and passwords
type StaticCredentialStore struct {
	credentials map[string]string // Maps username/prefix to password
}

// Valid checks if the provided prefix and password are valid
func (s *StaticCredentialStore) Valid(prefix, password, userAddr string) bool {
	storedPassword, exists := s.credentials[prefix]
	return exists && storedPassword == password
}

func udpAssociateHandler(ctx context.Context, writer io.Writer, request *socks5.Request) error {
	// Extract the suffix from the auth context
	suffix, ok := request.AuthContext.Payload["suffix"]
	if !ok {
		return errors.New("no suffix found in auth context")
	}

	soaxPackage := viper.GetString("soax.package")
	soaxPassword := viper.GetString("soax.password")
	soaxAddress := viper.GetString("soax.address")

	// Construct the upstream soax username by concatenating
	// the soax package name with the suffix
	soaxUsername := soaxPackage + suffix
	log.Printf("Sending associate command with username: %s\n", soaxUsername)

	streamEndpoint := transport.StreamDialerEndpoint{Dialer: &transport.TCPDialer{}, Address: soaxAddress}
	client, err := csocks5.NewClient(&streamEndpoint)
	if err != nil {
		return err
	}

	err = client.SetCredentials([]byte(soaxUsername), []byte(soaxPassword))
	if err != nil {
		return err
	}

	client.EnablePacket(&transport.UDPDialer{})

	conn, bindAddr, err := client.ConnectAndRequest(ctx, csocks5.CmdUDPAssociate, "0.0.0.0:0")
	if err != nil {
		return err
	}

	// Start a goroutine to close the connection after a timeout
	go func() {
		timeout := viper.GetDuration("udp_timeout")
		<-time.After(timeout)
		log.Println("Closing UDP associate connection after timeout")
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

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	return nil
}

func main() {

	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	creds := &StaticCredentialStore{
		credentials: viper.GetStringMapString("credentials"),
	}

	// Create the custom authenticator
	customAuth := CustomAuthenticator{
		PrefixCredentials: creds,
	}

	server := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{customAuth}),
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
		socks5.WithDialAndRequest(func(ctx context.Context, network, addr string, req *socks5.Request) (net.Conn, error) {
			authContext := req.AuthContext
			if authContext == nil {
				return nil, errors.New("no auth context available")
			}

			suffix, ok := authContext.Payload["suffix"]
			if !ok {
				return nil, errors.New("no suffix found in auth context")
			}

			soaxpackage := viper.GetString("soax.package")
			soaxPassword := viper.GetString("soax.password")
			soaxAddress := viper.GetString("soax.address")

			// Construct the upstream username by concatenating the base username with the suffix
			soaxUsername := soaxpackage + suffix // e.g., "package-189365-country-ir-seesionid-..."

			switch network {
			case "tcp", "tcp4", "tcp6":
				streamEndpoint := transport.StreamDialerEndpoint{Dialer: &transport.TCPDialer{}, Address: soaxAddress}
				client, err := csocks5.NewClient(&streamEndpoint)
				if err != nil {
					return nil, err
				}

				err = client.SetCredentials([]byte(soaxUsername), []byte(soaxPassword))
				if err != nil {
					return nil, err
				}
				return client.DialStream(ctx, addr)
			default:
				return nil, fmt.Errorf("unsupported network: %s", network)
			}
		}),
		socks5.WithAssociateHandle(udpAssociateHandler),
	)

	// Run SOCKS5 proxy on the specified address and port
	listeningAddr := fmt.Sprintf("%s:%s", viper.GetString("server.address"), viper.GetString("server.port"))
	if err := server.ListenAndServe("tcp", listeningAddr); err != nil {
		panic(err)
	}

}
