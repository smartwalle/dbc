package dbc

import (
	"github.com/smartwalle/dbc/nmap"
	"runtime"
	"time"
)

type Cache interface {
	Set(key string, value interface{})

	SetEx(key string, value interface{}, expiration time.Duration)

	Exists(key string) bool

	Get(key string) interface{}

	Del(key string)

	OnEvicted(func(key string, value interface{}))
}

type cacheWrapper struct {
	*sharedCache
}

func stopJanitor(c *cacheWrapper) {
	c.close()
}

type Option func(c *sharedCache)

func WithCleanupInterval(interval time.Duration) Option {
	return func(c *sharedCache) {
		c.cleanupInterval = interval
	}
}

type sharedCache struct {
	cleanupInterval time.Duration
	items           *nmap.Map
	janitor         *Janitor
	onEvicted       func(string, interface{})
}

func New(opts ...Option) Cache {
	var sc = &sharedCache{}
	sc.items = nmap.New()
	sc.cleanupInterval = 0

	for _, opt := range opts {
		opt(sc)
	}

	var janitor = NewJanitor(sc.cleanupInterval)
	go janitor.run(sc)
	sc.janitor = janitor

	var c = &cacheWrapper{}
	c.sharedCache = sc
	runtime.SetFinalizer(c, stopJanitor)

	return c
}

func (this *sharedCache) Tick() {
	this.items.Range(func(key string, item nmap.Item) bool {
		if item.Expired() {
			this.Del(key)
		}
		return true
	})
}

func (this *sharedCache) close() {
	this.janitor.close()
}

func (this *sharedCache) Set(key string, value interface{}) {
	this.SetEx(key, value, 0)
}

func (this *sharedCache) SetEx(key string, value interface{}, expiration time.Duration) {
	var t int64
	if expiration > 0 {
		t = time.Now().Add(expiration).UnixNano()
	}
	var nItem = nmap.NewItem(value, t)
	this.items.Set(key, nItem)
}

func (this *sharedCache) Exists(key string) bool {
	return this.items.Exists(key)
}

func (this *sharedCache) Get(key string) interface{} {
	var item, ok = this.items.Get(key)
	if ok == false {
		return nil
	}
	if item.Expired() {
		return nil
	}
	return item.Data()
}

func (this *sharedCache) Del(key string) {
	if this.onEvicted != nil {
		var item, ok = this.items.Get(key)
		if ok {
			this.onEvicted(key, item.Data())
		}
	}
	this.items.Remove(key)
}

func (this *sharedCache) OnEvicted(f func(key string, value interface{})) {
	this.onEvicted = f
}
