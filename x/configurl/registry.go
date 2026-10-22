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
	"context"
	"fmt"
	"reflect"
	"sync"

	"golang.getoutline.org/sdk/transport"
)

// Parser constructs a value from a config. A nil config requests the default base.
// Forward ctx to dependency parsers; returned objects should use the context of
// each later operation rather than retain the construction context.
// Parser and the existing BuildFunc are interchangeable.
type Parser[T any] = BuildFunc[T]

// Registry looks up parsers by their exact Go result type and config URL scheme.
// Use [Lookup], [New], or [NewFromConfig] for typed access. Implementations may store
// parsers however they choose; the interface does not expose storage or base instances.
type Registry interface {
	// LookupParser returns the parser registered for resultType and name, with ok
	// reporting whether the entry exists. The empty name denotes the default base
	// parser, invoked with a nil Config. A registered nil parser disables the entry.
	// A non-nil parser must have the unnamed function type
	// func(context.Context, *Config) (T, error), where T is exactly resultType.
	LookupParser(resultType reflect.Type, name string) (parser any, ok bool)
}

// MutableRegistry adds registration to [Registry]. Use [Register] or [RegisterBase]
// for type safety.
// Configure registries before using components built from them; changing a parser
// does not change objects already constructed. Implementations used concurrently
// must synchronize lookup and registration.
type MutableRegistry interface {
	Registry
	// RegisterParser registers parser for resultType and name, replacing any existing
	// registration. The empty name selects the default base parser; a nil parser
	// disables the entry.
	// It accepts named or unnamed functions with signature
	// func(context.Context, *Config) (T, error), where T is exactly resultType,
	// and converts them to the unnamed function type returned by LookupParser.
	// It panics if resultType is nil or parser has an incompatible type.
	RegisterParser(resultType reflect.Type, name string, parser any)
}

type parserKey struct {
	resultType reflect.Type
	name       string
}

type registry struct {
	mu      sync.RWMutex
	parsers map[parserKey]any
}

// LookupParser implements [Registry].
func (r *registry) LookupParser(resultType reflect.Type, name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	parser, ok := r.parsers[parserKey{resultType, name}]
	return parser, ok
}

// RegisterParser implements [MutableRegistry].
func (r *registry) RegisterParser(resultType reflect.Type, name string, parser any) {
	parser = normalizeParser(resultType, parser)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers[parserKey{resultType, name}] = parser
}

// Register registers a parser for a config URL scheme and exact result type T.
// Use an interface type argument when registering an implementation of it.
// The empty name registers the default base parser. A nil parser disables the entry.
func Register[T any](r MutableRegistry, name string, parser Parser[T]) {
	r.RegisterParser(reflect.TypeFor[T](), name, parser)
}

// RegisterBase registers the parser used when a config is nil.
// It is a convenience function equivalent to calling [Register] with an empty name:
// Register(r, "", parser).
func RegisterBase[T any](r MutableRegistry, parser Parser[T]) {
	Register(r, "", parser)
}

// Lookup returns a typed parser without invoking it. It checks the function's
// signature before returning it. Capturing it before replacing a registration
// lets a new parser delegate to the previous parser.
func Lookup[T any](r Registry, name string) (Parser[T], error) {
	resultType := reflect.TypeFor[T]()
	if r == nil {
		return nil, fmt.Errorf("cannot look up %v parser %q in a nil registry", resultType, name)
	}
	parser, ok := r.LookupParser(resultType, name)
	if !ok {
		return nil, fmt.Errorf("%v config type %q is not registered", resultType, name)
	}
	if parser == nil {
		return nil, fmt.Errorf("%v config type %q parser is nil", resultType, name)
	}
	newInstance, ok := parser.(func(context.Context, *Config) (T, error))
	if !ok {
		return nil, fmt.Errorf("%v config type %q has incompatible parser type %T", resultType, name, parser)
	}
	if newInstance == nil {
		return nil, fmt.Errorf("%v config type %q parser is nil", resultType, name)
	}
	return Parser[T](newInstance), nil
}

// NewFromConfig constructs a T from a parsed config. A nil config invokes the base parser.
func NewFromConfig[T any](ctx context.Context, r Registry, config *Config) (T, error) {
	name := ""
	if config != nil {
		name = config.URL.Scheme
		if name == "" {
			var zero T
			return zero, fmt.Errorf("config URL has no scheme")
		}
	}
	parser, err := Lookup[T](r, name)
	if err != nil {
		var zero T
		return zero, err
	}
	return parser(ctx, config)
}

// New constructs a T from a config URL string. An empty or whitespace-only string
// invokes the base parser. Use NewFromConfig when the config is already parsed.
func New[T any](ctx context.Context, r Registry, configText string) (T, error) {
	config, err := ParseConfig(configText)
	if err != nil {
		var zero T
		return zero, err
	}
	return NewFromConfig[T](ctx, r, config)
}

// NewRegistry creates a registry with default base implementations of
// [transport.StreamDialer], [transport.PacketDialer], and [transport.PacketListener].
// Other result types and config URL schemes can be added with Register.
func NewRegistry() MutableRegistry {
	r := &registry{parsers: make(map[parserKey]any)}
	streamDialer := &transport.TCPDialer{}
	newStreamDialer := func(context.Context, *Config) (transport.StreamDialer, error) {
		return streamDialer, nil
	}
	RegisterBase(r, newStreamDialer)
	packetDialer := &transport.UDPDialer{}
	newPacketDialer := func(context.Context, *Config) (transport.PacketDialer, error) {
		return packetDialer, nil
	}
	RegisterBase(r, newPacketDialer)
	packetListener := &transport.UDPListener{}
	newPacketListener := func(context.Context, *Config) (transport.PacketListener, error) {
		return packetListener, nil
	}
	RegisterBase(r, newPacketListener)
	return r
}

// NewDefaultRegistry creates a registry with default bases and built-in parsers.
func NewDefaultRegistry() MutableRegistry { return RegisterDefaultTypes(NewRegistry()) }

// RegisterDefaultTypes registers the built-in config URL parsers, preserving bases.
func RegisterDefaultTypes(registry MutableRegistry) MutableRegistry {
	// Please keep the list in alphabetical order.
	registerDisorderDialer(registry, "disorder")
	registerDO53StreamDialer(registry, "do53")
	registerDOHStreamDialer(registry, "doh")

	registerH2ConnectStreamDialer(registry, "h2connect")
	registerH3ConnectStreamDialer(registry, "h3connect")
	registerHTTPConnectStreamDialer(registry, "httpconnect")

	registerOverrideStreamDialer(registry, "override")
	registerOverridePacketDialer(registry, "override")

	registerQUICPreludePacketListener(registry, "quicprelude")

	registerSOCKS5StreamDialer(registry, "socks5")
	registerSOCKS5PacketDialer(registry, "socks5")
	registerSOCKS5PacketListener(registry, "socks5")

	registerSplitStreamDialer(registry, "split")

	registerShadowsocksStreamDialer(registry, "ss")
	registerShadowsocksPacketDialer(registry, "ss")
	registerShadowsocksPacketListener(registry, "ss")

	registerTLSStreamDialer(registry, "tls")

	registerTLSFragStreamDialer(registry, "tlsfrag")

	registerWebsocketStreamDialer(registry, "ws")
	registerWebsocketPacketDialer(registry, "ws")
	return registry
}

// normalizeParser validates a registration and converts named functions to the
// unnamed function type required by Registry, without wrapping or invoking them.
func normalizeParser(resultType reflect.Type, parser any) any {
	if resultType == nil {
		panic("configurl: parser result type is nil")
	}
	if parser == nil {
		return nil
	}
	parserType := reflect.FuncOf(
		[]reflect.Type{reflect.TypeFor[context.Context](), reflect.TypeFor[*Config]()},
		[]reflect.Type{resultType, reflect.TypeFor[error]()}, false,
	)
	value := reflect.ValueOf(parser)
	if !value.Type().AssignableTo(parserType) {
		panic(fmt.Sprintf("configurl: parser for %v has type %T, want %v", resultType, parser, parserType))
	}
	if value.IsNil() {
		return nil
	}
	return value.Convert(parserType).Interface()
}
