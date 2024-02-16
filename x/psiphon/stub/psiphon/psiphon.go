package psiphon

import (
	"context"
	"errors"
	"net"
)

type Config struct{}

type Controller struct{}

func LoadConfig(configJSON []byte) (*Config, error) {
	return nil, errors.New("not available")
}

func NewController(config *Config) (controller *Controller, err error) {
	return nil, errors.New("not available")
}

func (controller *Controller) Run(ctx context.Context) {}

func (controller *Controller) Dial(remoteAddr string, downstreamConn net.Conn) (conn net.Conn, err error) {
	return nil, errors.New("not available")
}
