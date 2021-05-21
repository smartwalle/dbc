package dbc

import (
	"github.com/smartwalle/dbc/nmap"
	"runtime"
	"time"
)

type Cache interface {
	Set(key string, value interface{})

	SetEx(key string, value interface{}, expiration time.Duration)

	SetNx(key string, value interface{}) bool

	Expire(key string, expiration time.Duration)

	Exists(key string) bool

	Get(key string) interface{}

	Del(key string)

	OnEvicted(func(key string, value interface{}))
}

type cacheWrapper struct {
	*cache
}

type Option func(c *cache)

func WithCleanup(interval time.Duration) Option {
	return func(c *cache) {
		c.cleanupInterval = interval
	}
}

type cache struct {
	cleanupInterval time.Duration
	items           *nmap.Map
	janitor         *Janitor
	onEvicted       func(string, interface{})
}

func New(opts ...Option) Cache {
	var sc = &cache{}
	sc.items = nmap.New()
	sc.cleanupInterval = 0

	for _, opt := range opts {
		opt(sc)
	}

	var janitor = NewJanitor(sc.cleanupInterval)
	go janitor.run(sc)
	sc.janitor = janitor

	var c = &cacheWrapper{}
	c.cache = sc
	runtime.SetFinalizer(c, stopJanitor)

	return c
}

func stopJanitor(c *cacheWrapper) {
	c.close()
}

func (this *cache) Tick() {
	this.items.Range(func(key string, item nmap.Item) bool {
		if item.Expired() {
			this.Del(key)
		}
		return true
	})
}

func (this *cache) close() {
	this.janitor.close()
}

func (this *cache) Set(key string, value interface{}) {
	this.SetEx(key, value, 0)
}

func (this *cache) SetEx(key string, value interface{}, expiration time.Duration) {
	var t int64
	if expiration > 0 {
		t = time.Now().Add(expiration).UnixNano()
	}
	var nItem = nmap.NewItem(value, t)
	this.items.Set(key, nItem)
}

func (this *cache) SetNx(key string, value interface{}) bool {
	var nItem = nmap.NewItem(value, 0)
	return this.items.SetNx(key, nItem)
}

func (this *cache) Expire(key string, expiration time.Duration) {
	var item, ok = this.items.Get(key)
	if ok {
		var t int64
		if expiration > 0 {
			t = time.Now().Add(expiration).UnixNano()
		}
		item.UpdateExpiration(t)
		this.items.Set(key, item)
	}
}

func (this *cache) Exists(key string) bool {
	return this.items.Exists(key)
}

func (this *cache) Get(key string) interface{} {
	var item, ok = this.items.Get(key)
	if ok == false {
		return nil
	}
	if item.Expired() {
		return nil
	}
	return item.Data()
}

func (this *cache) Del(key string) {
	var item, ok = this.items.Pop(key)
	if this.onEvicted != nil && ok {
		this.onEvicted(key, item.Data())
	}
}

func (this *cache) OnEvicted(f func(key string, value interface{})) {
	this.onEvicted = f
}
