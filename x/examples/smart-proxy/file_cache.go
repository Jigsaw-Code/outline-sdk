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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
)

// JSONFileCache implements a key-value cache using a JSON file.
type JSONFileCache struct {
	path  string
	mu    sync.RWMutex
	cache map[string]string
}

// NewJSONFileCache creates a new JSONFileCache.
func NewJSONFileCache(path string) (*JSONFileCache, error) {
	c := &JSONFileCache{
		path:  path,
		cache: make(map[string]string),
	}
	data, err := os.ReadFile(c.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("failed to read the cache file: %v\n", err)
		return nil, err
	}
	if len(data) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(data, &c.cache); err != nil {
		fmt.Printf("failed to unmarshal the cache file content: %v\n", err)
		return nil, err
	}
	return c, nil
}

// flushNoLock writes the current cache data to the JSON file without holding the lock.
func (c *JSONFileCache) flushNoLock() error {
	data, err := json.MarshalIndent(c.cache, "", "  ")
	if err != nil {
		fmt.Printf("failed to marshal the cache content: %v\n", err)
		return err
	}

	err = os.WriteFile(c.path, data, 0644)
	if err != nil {
		fmt.Printf("failed to write to the cache file: %v\n", err)
	}
	return err
}

// Get retrieves a strategy result string associated with the given key.
func (c *JSONFileCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.cache[key]
	return []byte(val), ok
}

// Put adds the strategy result string to the cache with the given key.
func (c *JSONFileCache) Put(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if val == nil {
		delete(c.cache, key)
	} else {
		c.cache[key] = string(val)
	}
	c.flushNoLock()
}
