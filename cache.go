package dbc

import (
	"sync"
	"time"
)

func NewCache(ttl time.Duration) *Cache {
	var c = &Cache{}
	if ttl < time.Second {
		ttl = time.Second
	}
	c.ttl = ttl
	c.items = make(map[string]*item)
	c.run()
	return c
}

type Cache struct {
	mu     sync.RWMutex
	ttl    time.Duration
	ticker *time.Ticker
	items  map[string]*item
}

func (this *Cache) Get(key string) (value interface{}) {
	this.mu.Lock()
	defer this.mu.Unlock()

	item, exists := this.items[key]
	if exists == false {
		return nil
	}
	if item.expired() {
		delete(this.items, key)
		return nil
	}
	item.touch(this.ttl)
	return item.value
}

func (this *Cache) Set(key string, value interface{}) {
	this.mu.Lock()
	defer this.mu.Unlock()

	var item = newItem(value, this.ttl)
	this.items[key] = item
}

func (this *Cache) Del(key string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	delete(this.items, key)
}

func (this *Cache) Count() int {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return len(this.items)
}

func (this *Cache) Close() error {
	this.ticker.Stop()
	this.items = nil
	return nil
}

func (this *Cache) cleanup() {
	this.mu.Lock()
	defer this.mu.Unlock()

	for key, item := range this.items {
		if item.expired() {
			delete(this.items, key)
		}
	}
}

func (this *Cache) run() {
	this.ticker = time.NewTicker(this.ttl)
	go func() {
		for {
			select {
			case <-this.ticker.C:
				this.cleanup()
			}
		}
	}()
}
