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

package configurl

import (
	"reflect"

	"golang.getoutline.org/sdk/transport"
)

// providerContainerRegistry lets the released provider API use the same component
// registration helpers. Lookups read the providers themselves so later changes
// to their registrations or BaseInstance fields remain visible to dependencies.
// This adapter is private and is used only to register the built-in named types.
type providerContainerRegistry struct{ *ProviderContainer }

// LookupParser implements [Registry].
func (r providerContainerRegistry) LookupParser(resultType reflect.Type, name string) (Parser[any], bool) {
	switch resultType {
	case reflect.TypeFor[transport.StreamDialer]():
		return lookupProviderParser(&r.StreamDialers, name)
	case reflect.TypeFor[transport.PacketDialer]():
		return lookupProviderParser(&r.PacketDialers, name)
	case reflect.TypeFor[transport.PacketListener]():
		return lookupProviderParser(&r.PacketListeners, name)
	default:
		return nil, false
	}
}

// RegisterParser implements [MutableRegistry].
func (r providerContainerRegistry) RegisterParser(resultType reflect.Type, name string, parser Parser[any]) {
	switch resultType {
	case reflect.TypeFor[transport.StreamDialer]():
		r.StreamDialers.RegisterType(name, typedParser[transport.StreamDialer](name, parser))
	case reflect.TypeFor[transport.PacketDialer]():
		r.PacketDialers.RegisterType(name, typedParser[transport.PacketDialer](name, parser))
	case reflect.TypeFor[transport.PacketListener]():
		r.PacketListeners.RegisterType(name, typedParser[transport.PacketListener](name, parser))
	default:
		panic("unsupported built-in configurl result type: " + resultType.String())
	}
}

func lookupProviderParser[T comparable](provider *ExtensibleProvider[T], name string) (Parser[any], bool) {
	if name == "" {
		return eraseParser(provider.NewInstance), true
	}
	parser, ok := provider.builders[name]
	return eraseParser(parser), ok
}
