// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"sync"

	"github.com/0verkilll/i18n/internal/core"
)

// Compile-time assertion that MapCache implements Cacher.
var _ core.Cacher = (*MapCache)(nil)

// lruEntry is a node in the doubly-linked list used for LRU eviction tracking.
// This is a custom implementation (not container/list) for TinyGo/WASM safety.
type lruEntry struct {
	prev  *lruEntry
	next  *lruEntry
	key   string
	value string
}

// MapCache is a thread-safe, in-memory translation cache backed by a map.
// It supports an optional LRU eviction policy when a maximum entry count is
// configured. When no limit is set (maxEntries == 0), the cache grows without
// bound. All methods are safe for concurrent use by multiple goroutines.
type MapCache struct {
	entries    map[string]*lruEntry
	head       *lruEntry
	tail       *lruEntry
	mu         sync.RWMutex
	maxEntries int
}

// NewMapCache creates a new MapCache with no size limit (unlimited mode).
// Entries are never evicted; call Invalidate to clear the cache.
func NewMapCache() *MapCache {
	return &MapCache{
		entries: make(map[string]*lruEntry),
	}
}

// NewMapCacheWithLimit creates a new MapCache with LRU eviction enabled.
// When the entry count exceeds maxEntries, the least-recently-used entry is
// evicted. If maxEntries is zero or negative, the cache behaves as unlimited.
func NewMapCacheWithLimit(maxEntries int) *MapCache {
	limit := maxEntries
	if limit < 0 {
		limit = 0
	}
	return &MapCache{
		entries:    make(map[string]*lruEntry),
		maxEntries: limit,
	}
}

// Get retrieves a cached translation by its cache key.
// Returns the cached value and true on a hit, or an empty string and false
// on a miss. When LRU eviction is enabled, a hit promotes the entry to the
// head of the access list (requires a write lock).
func (c *MapCache) Get(key string) (string, bool) {
	if c.maxEntries > 0 {
		return c.getWithPromotion(key)
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	return entry.value, true
}

// getWithPromotion handles Get when LRU mode is active, acquiring a write lock
// so the accessed entry can be promoted to the head of the list.
func (c *MapCache) getWithPromotion(key string) (string, bool) {
	c.mu.Lock()
	entry, ok := c.entries[key]
	if !ok {
		c.mu.Unlock()
		return "", false
	}
	c.promoteToHead(entry)
	value := entry.value
	c.mu.Unlock()
	return value, true
}

// Set stores a resolved translation under the given cache key.
// If the key already exists, its value is updated and the entry is promoted
// to the head of the LRU list. When at capacity, the least-recently-used
// entry (tail) is evicted.
func (c *MapCache) Set(key, value string) {
	c.mu.Lock()
	if existing, ok := c.entries[key]; ok {
		existing.value = value
		c.promoteToHead(existing)
		c.mu.Unlock()
		return
	}

	entry := &lruEntry{key: key, value: value}
	c.entries[key] = entry
	c.pushToHead(entry)

	if c.maxEntries > 0 && len(c.entries) > c.maxEntries {
		c.evictTail()
	}
	c.mu.Unlock()
}

// Invalidate discards all cached entries, resetting the cache to an empty state.
func (c *MapCache) Invalidate() {
	c.mu.Lock()
	c.entries = make(map[string]*lruEntry)
	c.head = nil
	c.tail = nil
	c.mu.Unlock()
}

// pushToHead inserts an entry at the head of the doubly-linked list.
// Must be called with the write lock held.
func (c *MapCache) pushToHead(entry *lruEntry) {
	entry.prev = nil
	entry.next = c.head
	if c.head != nil {
		c.head.prev = entry
	}
	c.head = entry
	if c.tail == nil {
		c.tail = entry
	}
}

// detach removes an entry from the doubly-linked list without deleting it
// from the map. Must be called with the write lock held.
func (c *MapCache) detach(entry *lruEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		c.head = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		c.tail = entry.prev
	}
	entry.prev = nil
	entry.next = nil
}

// promoteToHead moves an existing entry to the head of the list, marking it
// as the most recently used. Must be called with the write lock held.
func (c *MapCache) promoteToHead(entry *lruEntry) {
	if c.head == entry {
		return
	}
	c.detach(entry)
	c.pushToHead(entry)
}

// evictTail removes the least-recently-used entry (the tail) from both the
// linked list and the map. Must be called with the write lock held.
func (c *MapCache) evictTail() {
	if c.tail == nil {
		return
	}
	evicted := c.tail
	c.detach(evicted)
	delete(c.entries, evicted.key)
}
