// Copyright 2025 The Outline Authors
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

// If the build tag `psiphon` is set, allow importing and calling psiphon

package psiphon

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/mobileproxy"
	"github.com/Jigsaw-Code/outline-sdk/x/smart"
)

// Takes a (potentially very long) psiphon config and outputs
// a short signature string for logging identification purposes
// with only the PropagationChannelId and SponsorId (required fields)
// ex: {PropagationChannelId: FFFFFFFFFFFFFFFF, SponsorId: FFFFFFFFFFFFFFFF, [...]}
// If the config does not contains these fields
// output the whole config as a string
func getPsiphonConfigSignature(yamlNode smart.YAMLNode) string {
	switch yamlMap := yamlNode.(type) {
	case map[string]any:
		propagationChannelId, ok1 := yamlMap["PropagationChannelId"].(string)
		sponsorId, ok2 := yamlMap["SponsorId"].(string)

		if ok1 && ok2 {
			return fmt.Sprintf("{PropagationChannelId: %v, SponsorId: %v, [...]}", propagationChannelId, sponsorId)
		}
	}
	jsonBytes, err := json.Marshal(yamlNode)
	if err != nil {
		return fmt.Sprintf("invalid config: %v", err)
	}
	return string(jsonBytes)
}

// ParseConfig creates the Psiphon StreamDialer from a config.
func ParseConfig(ctx context.Context, yamlNode smart.YAMLNode) (transport.StreamDialer, string, error) {
	dialer, err := parsePsiphon(ctx, yamlNode)
	if err != nil {
		return nil, "", err
	}
	configSignature := getPsiphonConfigSignature(yamlNode)
	return dialer, configSignature, err
}

// RegisterFallbackParser registers the Psiphon config parser as a fallback parser for the given name (usually "psiphon").
// This function is a convenience wrapper around opts.RegisterFallbackParser that is compatible with Go Mobile.
//
// Parameters:
// * opts - the SmartDialerOptions to register the parser with
// * name - the name under which to register the fallback parser
func RegisterFallbackParser(opts *mobileproxy.SmartDialerOptions, name string) {
	opts.RegisterFallbackParser(name, ParseConfig)
}
