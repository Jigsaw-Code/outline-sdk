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

package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/tls"
)

func parseOptions(configURL *url.URL) ([]tls.ClientOption, error) {
	query := configURL.Opaque
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	options := []tls.ClientOption{}
	for key, values := range values {
		switch strings.ToLower(key) {
		case "sni":
			if len(values) != 1 {
				return nil, fmt.Errorf("sni option must has one value, found %v", len(values))
			}
			options = append(options, tls.WithSNI(values[0]))
		case "certname":
			if len(values) != 1 {
				return nil, fmt.Errorf("certName option must has one value, found %v", len(values))
			}
			options = append(options, tls.WithCertificateName(values[0]))
		default:
			return nil, fmt.Errorf("unsupported option %v", key)

		}
	}
	return options, nil
}

func wrapStreamDialerWithTLS(innerDialer transport.StreamDialer, configURL *url.URL) (transport.StreamDialer, error) {
	options, err := parseOptions(configURL)
	if err != nil {
		return nil, err
	}
	return tls.NewStreamDialer(innerDialer, options...)
}
