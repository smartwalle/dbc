package dbc

import (
	"sync"
	"time"
)

func newCacheItem(key string, value interface{}, ttl time.Duration) *cacheItem {
	var now = time.Now()

	if ttl < 0 {
		ttl = 0
	}

	var item = &cacheItem{}
	item.value = value
	item.ttl = ttl
	item.accessedOn = now
	return item
}

type cacheItem struct {
	mu    sync.RWMutex
	key   string
	value interface{}

	ttl        time.Duration
	accessedOn time.Time
}

func (this *cacheItem) touch() {
	this.mu.Lock()
	defer this.mu.Unlock()

	this.accessedOn = time.Now()
}

func (this *cacheItem) update(value interface{}, ttl time.Duration) {
	this.mu.Lock()
	this.value = value
	this.ttl = ttl
	this.mu.Unlock()
}

func (this *cacheItem) expired(now time.Time) (bool, time.Duration) {
	this.mu.RLock()
	var ttl = this.ttl
	var accessedOn = this.accessedOn
	this.mu.RUnlock()

	if ttl == 0 {
		return false, 0
	}

	if now.Sub(accessedOn) >= ttl {
		return true, -1
	}

	ttl = ttl - now.Sub(accessedOn)
	return false, ttl
}
