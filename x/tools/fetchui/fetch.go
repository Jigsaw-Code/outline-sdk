package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
	tea "github.com/charmbracelet/bubbletea"
)

func doFetch(req *request) tea.Cmd {
	return func() tea.Msg {
		providers := configurl.NewDefaultProviders()
		streamDialer, err := providers.NewStreamDialer(context.Background(), req.transport)
		if err != nil {
			return fetchResultMsg{req: req, status: fmt.Sprintf("failed to create dialer for %s: %v", req.transport, err)}
		}

		client := http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return streamDialer.DialStream(ctx, addr)
				},
			},
		}

		ctx, dismiss := context.WithTimeout(context.Background(), 10*time.Second)
		defer dismiss()
		httpReq, err := http.NewRequestWithContext(ctx, "GET", req.url, nil)
		if err != nil {
			return fetchResultMsg{req: req, status: fmt.Sprintf("failed to create request for %s: %v", req.url, err)}
		}
		resp, err := client.Do(httpReq)
		if err != nil {
			if errors.Is(context.Cause(ctx), context.DeadlineExceeded) {
				return fetchResultMsg{req: req, status: "TIMEOUT"}
			}
			return fetchResultMsg{req: req, status: err.Error()}
		}
		defer resp.Body.Close()

		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return fetchResultMsg{req: req, status: fmt.Sprintf("failed to read response body from %s with %s: %v", req.url, req.transport, err)}
		}

		return fetchResultMsg{req: req, status: "SUCCESS"}
	}
}
