package dbc

import (
	"sync"
	"time"
)

const (
	kDefaultSize = 1024
)

type option func(*Cache)

type removeHandler func(key string, value interface{})

func WithDefaultSize(size int) option {
	return func(c *Cache) {
		c.defaultSize = size
	}
}

func WithRemoveHandler(handler removeHandler) option {
	return func(c *Cache) {
		c.removedItem = handler
	}
}

func NewCache(opts ...option) *Cache {
	var c = &Cache{}
	for _, opt := range opts {
		opt(c)
	}
	if c.defaultSize <= 0 {
		c.defaultSize = kDefaultSize
	}
	c.items = make(map[string]*cacheItem, c.defaultSize)
	c.ttl = 0
	return c
}

type Cache struct {
	mu sync.RWMutex

	defaultSize int
	items       map[string]*cacheItem

	ttlTimer *time.Timer
	ttl      time.Duration

	removedItem removeHandler
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
	this.mu.Lock()
	var item = this.items[key]
	if item != nil {
		item.update(value, ttl)
	} else {
		item = newCacheItem(key, value, ttl)
	}
	this.items[key] = item
	this.mu.Unlock()

	if item.ttl > 0 && (this.ttl == 0 || item.ttl < this.ttl) {
		this.ttlCheck()
	}
}

func (this *Cache) SetNx(key string, value interface{}, ttl time.Duration) bool {
	this.mu.Lock()
	var _, exists = this.items[key]
	if exists {
		this.mu.Unlock()
		return false
	}

	var item = newCacheItem(key, value, ttl)
	this.items[key] = item
	this.mu.Unlock()

	if item.ttl > 0 && (this.ttl == 0 || item.ttl < this.ttl) {
		this.ttlCheck()
	}
	return true
}

func (this *Cache) Del(key string) {
	this.mu.Lock()
	var item = this.items[key]
	if item == nil {
		this.mu.Unlock()
		return
	}
	delete(this.items, key)
	if this.removedItem != nil {
		this.removedItem(key, item.value)
	}
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
			if this.removedItem != nil {
				this.mu.Unlock()
				this.removedItem(key, item.value)
				this.mu.Lock()
			}
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

func (this *Cache) Close() {
	this.mu.Lock()
	if len(this.items) == 0 {
		this.mu.Unlock()
		return
	}

	var items = make([]*cacheItem, 0, len(this.items))

	if this.ttlTimer != nil {
		this.ttlTimer.Stop()
	}

	for key, item := range this.items {
		items = append(items, item)
		delete(this.items, key)
	}
	this.items = make(map[string]*cacheItem)
	this.mu.Unlock()

	for _, item := range items {
		this.removedItem(item.key, item.value)
	}
}
