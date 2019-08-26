package dbc

import (
	"sync"
	"time"
)

const (
	kDefaultSize = 1024
)

type Option interface {
	Apply(*Cache)
}

type OptionFunc func(*Cache)

func (f OptionFunc) Apply(c *Cache) {
	f(c)
}

func WithDefaultSize(size int) Option {
	return OptionFunc(func(c *Cache) {
		c.defaultSize = size
	})
}

type HandlerFunc func(key string, value interface{})

func NewCache(opts ...Option) *Cache {
	var c = &Cache{}
	for _, opt := range opts {
		opt.Apply(c)
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

	removedHandler HandlerFunc
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
	if this.removedHandler != nil {
		this.removedHandler(key, item.value)
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
			if this.removedHandler != nil {
				this.mu.Unlock()
				this.removedHandler(key, item.value)
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
	var items = make([]*cacheItem, 0, len(this.items))

	if this.ttlTimer != nil {
		this.ttlTimer.Stop()
	}

	for key, item := range this.items {
		items = append(items, item)
		delete(this.items, key)
	}
	this.items = nil
	this.mu.Unlock()

	for _, item := range items {
		this.removedHandler(item.key, item.value)
	}
}

func (this *Cache) OnRemovedItem(h HandlerFunc) {
	this.mu.Lock()
	this.removedHandler = h
	this.mu.Unlock()
}
