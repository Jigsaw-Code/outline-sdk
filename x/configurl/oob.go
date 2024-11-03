package configurl

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

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
		if len(splitStr) != 4 {
			return nil, fmt.Errorf("oob: config should be in oob:<number>:<char>:<boolean>:<interval> format")
		}

		position, err := strconv.Atoi(splitStr[0])
		if err != nil {
			return nil, fmt.Errorf("oob: oob position is not a number: %v", splitStr[0])
		}

		if len(splitStr[1]) != 1 {
			return nil, fmt.Errorf("oob: oob byte should be a single character: %v", splitStr[1])
		}
		char := splitStr[1][0]

		disOOB, err := strconv.ParseBool(splitStr[2])
		if err != nil {
			return nil, fmt.Errorf("oob: disOOB is not a boolean: %v", splitStr[2])
		}

		delay, err := time.ParseDuration(splitStr[3])
		if err != nil {
			return nil, fmt.Errorf("oob: delay is not a duration: %v", splitStr[3])
		}
		return oob.NewStreamDialer(sd, int64(position), char, disOOB, delay)
	})
}
