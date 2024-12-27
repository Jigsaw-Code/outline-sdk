// Copyright 2024 The Outline Authors
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
	"flag"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
	"github.com/coder/websocket"
)

type natConn struct {
	net.Conn
	mappingTimeout time.Duration
}

// Consider ReadFrom/WriteTo
func (c *natConn) Write(b []byte) (int, error) {
	c.Conn.SetDeadline(time.Now().Add(c.mappingTimeout))
	return c.Conn.Write(b)
}

func main() {
	listenFlag := flag.String("listen", "localhost:8080", "Local proxy address to listen on")
	transportFlag := flag.String("transport", "", "Transport config")
	backendFlag := flag.String("backend", "", "Address of the endpoint to forward traffic to")
	tcpPathFlag := flag.String("tcp_path", "/tcp", "Path where to run the WebSocket TCP forwarder")
	udpPathFlag := flag.String("udp_path", "/udp", "Path where to run the WebSocket UDP forwarder")
	flag.Parse()

	if *backendFlag == "" {
		slog.Error("Must specify flag -backend")
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", *listenFlag)
	if err != nil {
		slog.Error("Could not listen", "address", *listenFlag, "error", err)
		os.Exit(1)
	}
	defer listener.Close()
	slog.Info("Proxy listening ", "address", listener.Addr().String())

	providers := configurl.NewDefaultProviders()
	mux := http.NewServeMux()
	if *tcpPathFlag != "" {
		dialer, err := providers.NewStreamDialer(context.Background(), *transportFlag)
		if err != nil {
			slog.Error("Could not create stream dialer", "error", err)
			os.Exit(1)
		}
		endpoint := transport.StreamDialerEndpoint{Dialer: dialer, Address: *backendFlag}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Debug("Got stream request", "request", r)
			defer slog.Debug("Request done")
			clientConn, err := websocket.Accept(w, r, nil)
			if err != nil {
				slog.Info("Failed to accept Websocket connection", "error", err)
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			defer clientConn.CloseNow()

			targetConn, err := endpoint.ConnectStream(r.Context())
			if err != nil {
				slog.Info("Failed connect to target endpoint", "error", err)
				clientConn.Close(websocket.StatusBadGateway, "")
				return
			}
			defer targetConn.Close()

			go func() {
				defer targetConn.CloseWrite()
				for {
					msgType, buf, err := clientConn.Read(r.Context())
					if err != nil {
						slog.Info("Read failed", "error", err)
						if !errors.Is(err, io.EOF) {
							clientConn.Close(websocket.StatusInternalError, "failed to read message from client")
						}
						break
					}
					if msgType != websocket.MessageBinary {
						slog.Info("Bad message type", "type", msgType)
						clientConn.Close(websocket.StatusUnsupportedData, "client message is not binary type")
						break
					}
					if _, err := targetConn.Write(buf); err != nil {
						slog.Info("Failed to write to target", "error", err)
						clientConn.Close(websocket.StatusInternalError, "failed to write message to target")
						break
					}
				}
			}()
			// About 2 MTUs
			buf := make([]byte, 3000)
			for {
				n, err := targetConn.Read(buf)
				if err != nil {
					slog.Info("Failed to read from target", "error", err)
					if !errors.Is(err, io.EOF) {
						clientConn.Close(websocket.StatusInternalError, "failed to read message from target")
					}
					break
				}
				msg := buf[:n]
				if err := clientConn.Write(r.Context(), websocket.MessageBinary, msg); err != nil {
					slog.Info("Failed to write to client", "error", err)
					clientConn.Close(websocket.StatusInternalError, "failed to write message to client")
					break
				}
			}
			clientConn.Close(websocket.StatusNormalClosure, "")
		})
		mux.Handle(*tcpPathFlag, http.StripPrefix(*tcpPathFlag, handler))
	}
	if *udpPathFlag != "" {
		dialer, err := providers.NewPacketDialer(context.Background(), *transportFlag)
		if err != nil {
			slog.Error("Could not create packet dialer", "error", err)
			os.Exit(1)
		}
		endpoint := transport.PacketDialerEndpoint{Dialer: dialer, Address: *backendFlag}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// log.Printf("Got packet request: %v\n", r)
			clientConn, err := websocket.Accept(w, r, nil)
			if err != nil {
				slog.Info("Failed to accept Websocket connection", "error", err)
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			defer clientConn.CloseNow()

			targetConn, err := endpoint.ConnectPacket(r.Context())
			if err != nil {
				slog.Info("Failed connect to Endpoint", "error", err)
				clientConn.Close(websocket.StatusBadGateway, "")
				return
			}
			// Expire connetion after 5 minutes of idle time, as per
			// https://datatracker.ietf.org/doc/html/rfc4787#section-4.3
			targetConn = &natConn{targetConn, 5 * time.Minute}
			defer targetConn.Close()

			go func() {
				defer targetConn.Close()
				for {
					msgType, buf, err := clientConn.Read(r.Context())
					if err != nil {
						if !errors.Is(err, io.EOF) {
							log.Printf("Failed to read from client: %v\n", err)
							clientConn.Close(websocket.StatusInternalError, "failed to read message from client")
						}
						break
					}
					if msgType != websocket.MessageBinary {
						log.Printf("Bad message type: %v\n", msgType)
						clientConn.Close(websocket.StatusUnsupportedData, "client message is not binary type")
						break
					}
					if _, err := targetConn.Write(buf); err != nil {
						log.Printf("Failed to write to target: %v\n", err)
						continue
						// clientConn.Close(websocket.StatusInternalError, "failed to write message to target")
						// break
					}
				}
			}()
			// About 2 MTUs
			buf := make([]byte, 3000)
			for {
				n, err := targetConn.Read(buf)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						log.Printf("Failed to read from target: %v\n", err)
						clientConn.Close(websocket.StatusInternalError, "failed to read message from target")
					}
					break
				}
				msg := buf[:n]
				if err := clientConn.Write(r.Context(), websocket.MessageBinary, msg); err != nil {
					log.Printf("Failed to write to client: %v\n", err)
					clientConn.Close(websocket.StatusInternalError, "failed to write message to client")
					break
				}
			}
			clientConn.Close(websocket.StatusNormalClosure, "")
		})
		mux.Handle(*udpPathFlag, http.StripPrefix(*udpPathFlag, handler))
	}
	server := http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("Error running web server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to stop the proxy.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	slog.Info("Shutting down")
	// Gracefully shut down the server, with a 5s timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Failed to shutdown gracefully: %v", err)
		os.Exit(1)
	}
}
