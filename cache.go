package dbc

import (
	"sync"
	"time"
)

func NewCache() *Cache {
	var c = &Cache{}
	c.items = make(map[string]*cacheItem)
	c.ttl = 0
	return c
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem

	ttlTimer *time.Timer
	ttl      time.Duration
}

func (this *Cache) Get(key string) (value interface{}) {
	this.mu.RLock()
	item, exists := this.items[key]
	this.mu.RUnlock()

	if exists == false {
		return nil
	}

	if expired, _ := item.expired(time.Now()); expired {
		return nil
	}

	item.touch()
	return item.value
}

func (this *Cache) Exists(key string) (exists bool) {
	this.mu.RLock()
	_, exists = this.items[key]
	this.mu.RUnlock()
	return exists
}

func (this *Cache) Set(key string, value interface{}, ttl time.Duration) {
	var item = newCacheItem(value, ttl)

	this.mu.Lock()
	this.items[key] = item
	this.mu.Unlock()

	if item.ttl > 0 && (this.ttl == 0 || item.ttl < this.ttl) {
		this.ttlCheck()
	}
}

func (this *Cache) Del(key string) {
	this.mu.Lock()
	delete(this.items, key)
	this.mu.Unlock()
}

func (this *Cache) Count() int {
	this.mu.RLock()
	var c = len(this.items)
	this.mu.RUnlock()
	return c
}

func (this *Cache) ttlCheck() {
	this.mu.Lock()

	if this.ttlTimer != nil {
		this.ttlTimer.Stop()
	}

	var now = time.Now()
	var nextTTL = time.Second * 0

	for key, item := range this.items {
		var expired, ttl = item.expired(now)
		if ttl == 0 {
			continue
		}

		if expired {
			delete(this.items, key)
		} else {
			if nextTTL == 0 || ttl < nextTTL {
				nextTTL = ttl
			}
		}
	}

	this.ttl = nextTTL
	if this.ttl > 0 {
		this.ttlTimer = time.AfterFunc(this.ttl, this.ttlCheck)
	}
	this.mu.Unlock()
}
