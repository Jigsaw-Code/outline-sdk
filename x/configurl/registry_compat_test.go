// Copyright 2026 The Outline Authors
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

//lint:file-ignore SA1019 Verify source compatibility with the deprecated registry API.

package configurl_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.getoutline.org/sdk/transport"
	"golang.getoutline.org/sdk/x/configurl"
)

// Verify compatibility from outside the package, including composite literals,
// constructor function values, registration functions, and method expressions.
var (
	_ *configurl.Registry          = &configurl.ProviderContainer{}
	_ *configurl.ProviderContainer = &configurl.Registry{}

	_ func() *configurl.Registry          = configurl.NewProviderContainer
	_ func() *configurl.ProviderContainer = configurl.NewRegistry
	_ func() *configurl.Registry          = configurl.NewDefaultProviders
	_ func() *configurl.ProviderContainer = configurl.NewDefaultRegistry

	_ func(*configurl.Registry) *configurl.Registry                   = configurl.RegisterDefaultProviders
	_ func(*configurl.ProviderContainer) *configurl.ProviderContainer = configurl.RegisterDefaultTypes

	_ func(*configurl.Registry, context.Context, string) (transport.StreamDialer, error)   = (*configurl.ProviderContainer).NewStreamDialer
	_ func(*configurl.Registry, context.Context, string) (transport.PacketDialer, error)   = (*configurl.ProviderContainer).NewPacketDialer
	_ func(*configurl.Registry, context.Context, string) (transport.PacketListener, error) = (*configurl.ProviderContainer).NewPacketListener
)

func TestRegistryConstructors(t *testing.T) {
	for _, tc := range []struct {
		name        string
		newRegistry func() *configurl.Registry
		defaults    bool
	}{
		{"NewRegistry", configurl.NewRegistry, false},
		{"NewProviderContainer", configurl.NewProviderContainer, false},
		{"NewDefaultRegistry", configurl.NewDefaultRegistry, true},
		{"NewDefaultProviders", configurl.NewDefaultProviders, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := tc.newRegistry()
			require.IsType(t, &transport.TCPDialer{}, registry.StreamDialers.BaseInstance)
			require.IsType(t, &transport.UDPDialer{}, registry.PacketDialers.BaseInstance)
			require.IsType(t, &transport.UDPListener{}, registry.PacketListeners.BaseInstance)

			ctx := context.Background()
			sd, err := registry.NewStreamDialer(ctx, "")
			require.NoError(t, err)
			require.Same(t, registry.StreamDialers.BaseInstance, sd)
			pd, err := registry.NewPacketDialer(ctx, "")
			require.NoError(t, err)
			require.Same(t, registry.PacketDialers.BaseInstance, pd)
			pl, err := registry.NewPacketListener(ctx, "")
			require.NoError(t, err)
			require.Same(t, registry.PacketListeners.BaseInstance, pl)

			_, streamErr := registry.NewStreamDialer(ctx, "split:1")
			_, packetErr := registry.NewPacketDialer(ctx, "override:port=53")
			_, listenerErr := registry.NewPacketListener(ctx, "quicprelude:count=0")
			for _, err := range []error{streamErr, packetErr, listenerErr} {
				if tc.defaults {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, "is not registered")
				}
			}
		})
	}
}

func TestRegisterDefaultTypesPreservesRegistry(t *testing.T) {
	for _, tc := range []struct {
		name     string
		register func(*configurl.Registry) *configurl.Registry
	}{
		{"RegisterDefaultTypes", configurl.RegisterDefaultTypes},
		{"RegisterDefaultProviders", configurl.RegisterDefaultProviders},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := configurl.NewRegistry()
			var legacy *configurl.ProviderContainer = registry
			baseErr := errors.New("configured base")
			dialStream := func(context.Context, string) (transport.StreamConn, error) {
				return nil, baseErr
			}
			registry.StreamDialers.BaseInstance = transport.FuncStreamDialer(dialStream)
			newInstance := func(context.Context, *configurl.Config) (transport.StreamDialer, error) {
				return legacy.StreamDialers.BaseInstance, nil
			}
			legacy.StreamDialers.RegisterType("custom", newInstance)
			require.Same(t, registry, tc.register(registry))

			for _, config := range []string{"split:1", "custom|split:1"} {
				dialer, err := registry.NewStreamDialer(context.Background(), config)
				require.NoError(t, err)
				_, err = dialer.DialStream(context.Background(), "unused.invalid:443")
				require.ErrorIs(t, err, baseErr)
			}
		})
	}
}
