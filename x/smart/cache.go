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

package smart

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"golang.org/x/net/dns/dnsmessage"
)

// canonicalName returns the domain name in canonical form. A name in canonical
// form is lowercase and fully qualified. Only US-ASCII letters are affected. See
// Section 6.2 in RFC 4034.
func canonicalName(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r
	}, s)
}

type cacheEntry struct {
	key    string
	msg    *dnsmessage.Message
	expire time.Time
}

// simpleLRUCacheResolver is a very simple caching [dns.Resolver].
// It doesn't use the response TTL.
// It also doesn't dedup duplicate in-flight requests.
type simpleLRUCacheResolver struct {
	resolver dns.Resolver
	cache    []cacheEntry
	mux      sync.Mutex
}

var _ dns.Resolver = (*simpleLRUCacheResolver)(nil)

func newSimpleLRUCacheResolver(resolver dns.Resolver, numEntries int) dns.Resolver {
	return &simpleLRUCacheResolver{resolver: resolver, cache: make([]cacheEntry, numEntries)}
}

func (r *simpleLRUCacheResolver) RemoveExpired() {
	now := time.Now()
	last := 0
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, entry := range r.cache {
		if entry.expire.After(now) {
			r.cache[last] = entry
			last++
		}
	}
	r.cache = r.cache[:last]
}

func (r *simpleLRUCacheResolver) moveToFront(index int) {
	entry := r.cache[index]
	copy(r.cache[1:], r.cache[:index])
	r.cache[0] = entry
}

func makeCacheKey(q dnsmessage.Question) string {
	domainKey := canonicalName(q.Name.String())
	return strings.Join([]string{domainKey, q.Type.String(), q.Class.String()}, "|")
}

func (r *simpleLRUCacheResolver) SearchCache(key string) *dnsmessage.Message {
	r.mux.Lock()
	defer r.mux.Unlock()
	for ei, entry := range r.cache {
		if entry.key == key {
			r.moveToFront(ei)
			// TODO: update TTLs
			// TODO: make names match
			return entry.msg
		}
	}
	return nil
}

func (r *simpleLRUCacheResolver) AddToCache(key string, msg *dnsmessage.Message) {
	r.mux.Lock()
	defer r.mux.Unlock()
	newSize := len(r.cache) + 1
	if newSize > cap(r.cache) {
		newSize = cap(r.cache)
	}
	r.cache = r.cache[:newSize]
	copy(r.cache[1:], r.cache[:newSize-1])
	// TODO: copy and normalize names
	r.cache[0] = cacheEntry{key: key, msg: msg, expire: time.Now().Add(60 * time.Second)}
}

// Query implements [dns.Resolver].
func (r *simpleLRUCacheResolver) Query(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
	r.RemoveExpired()
	cacheKey := makeCacheKey(q)
	if msg := r.SearchCache(cacheKey); msg != nil {
		return msg, nil
	}
	msg, err := r.resolver.Query(ctx, q)
	if err != nil {
		// TODO: cache server failures. See https://datatracker.ietf.org/doc/html/rfc2308.
		return nil, err
	}
	if msg.RCode == dnsmessage.RCodeSuccess || msg.RCode == dnsmessage.RCodeNameError {
		r.AddToCache(cacheKey, msg)
	}
	return msg, nil
}
