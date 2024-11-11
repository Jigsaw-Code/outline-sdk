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

package configurl

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newOverrideFromURL(t *testing.T) {
	t.Run("Host Override", func(t *testing.T) {
		cfgUrl, err := url.Parse("override:host=www.google.com")
		require.NoError(t, err)
		override, err := newOverrideFromURL(*cfgUrl)
		require.NoError(t, err)
		addr, err := override("www.youtube.com:443")
		require.NoError(t, err)
		require.Equal(t, "www.google.com:443", addr)
	})
	t.Run("Port Override", func(t *testing.T) {
		cfgUrl, err := url.Parse("override:port=853")
		require.NoError(t, err)
		override, err := newOverrideFromURL(*cfgUrl)
		require.NoError(t, err)
		addr, err := override("8.8.8.8:53")
		require.NoError(t, err)
		require.Equal(t, "8.8.8.8:853", addr)
	})
	t.Run("Full Override", func(t *testing.T) {
		cfgUrl, err := url.Parse("override:host=8.8.8.8&port=853")
		require.NoError(t, err)
		override, err := newOverrideFromURL(*cfgUrl)
		require.NoError(t, err)
		addr, err := override("dns.google:53")
		require.NoError(t, err)
		require.Equal(t, "8.8.8.8:853", addr)
	})
	t.Run("Invalid address", func(t *testing.T) {
		t.Run("Host Override", func(t *testing.T) {
			cfgUrl, err := url.Parse("override:host=www.google.com")
			require.NoError(t, err)
			override, err := newOverrideFromURL(*cfgUrl)
			require.NoError(t, err)
			_, err = override("foo bar")
			require.Error(t, err)
		})
		t.Run("Full Override", func(t *testing.T) {
			cfgUrl, err := url.Parse("override:host=8.8.8.8&port=853")
			require.NoError(t, err)
			override, err := newOverrideFromURL(*cfgUrl)
			require.NoError(t, err)
			addr, err := override("foo bar")
			require.NoError(t, err)
			require.Equal(t, "8.8.8.8:853", addr)
		})
	})
}
