package configurl

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/oob"
)

func registerOOBStreamDialer(r TypeRegistry[transport.StreamDialer], typeID string, newSD BuildFunc[transport.StreamDialer]) {
	r.RegisterType(typeID, func(ctx context.Context, config *Config) (transport.StreamDialer, error) {
		sd, err := newSD(ctx, config.BaseConfig)
		if err != nil {
			return nil, err
		}
		params := config.URL.Opaque

		splitStr := strings.Split(params, ":")
		if len(splitStr) != 3 {
			return nil, fmt.Errorf("split config should be in oob:<number>:<char>:<boolean> format")
		}

		position, err := strconv.Atoi(splitStr[0])
		if err != nil {
			return nil, fmt.Errorf("position is not a number: %v. Split config should be in oob:<number>:<char>:<boolean> format", splitStr[0])
		}

		if len(splitStr[1]) != 1 {
			return nil, fmt.Errorf("char should be a single character: %v. Split config should be in oob:<number>:<char>:<boolean> format", splitStr[1])
		}
		char := splitStr[1][0]

		disOOB, err := strconv.ParseBool(splitStr[2])
		if err != nil {
			return nil, fmt.Errorf("disOOB is not a boolean: %v. Split config should be in oob:<number>:<char>:<boolean> format", splitStr[2])
		}
		return oob.NewStreamDialerWithOOB(sd, int64(position), char, disOOB)
	})
}
