package dbc

import (
	"github.com/smartwalle/dbc/internal"
	"github.com/smartwalle/dbc/internal/nmap"
	"runtime"
	"time"
)

type Cache interface {
	Set(key string, value interface{})

	SetEx(key string, value interface{}, expiration time.Duration)

	SetNx(key string, value interface{}) bool

	Expire(key string, expiration time.Duration)

	Exists(key string) bool

	Get(key string) (interface{}, bool)

	Del(key string)

	Range(f func(key string, value interface{}) bool)

	Clear()

	OnEvicted(func(key string, value interface{}))
}

type cacheWrapper struct {
	*cache
}

type Option func(c *option)

func WithCleanup(interval time.Duration) Option {
	return func(opt *option) {
		opt.cleanupInterval = interval
	}
}

func WithHitTTL(ttl time.Duration) Option {
	return func(opt *option) {
		if ttl < 0 {
			ttl = 0
		}
		opt.hitTTL = ttl
	}
}

type option struct {
	cleanupInterval time.Duration
	hitTTL          time.Duration
}

type cache struct {
	*option
	items     *nmap.Map
	janitor   *internal.Janitor
	onEvicted func(string, interface{})
}

func New(opts ...Option) Cache {
	var nCache = &cache{}
	nCache.option = &option{}
	nCache.items = nmap.New()
	nCache.cleanupInterval = 0
	nCache.hitTTL = 0

	for _, opt := range opts {
		opt(nCache.option)
	}

	var janitor = internal.NewJanitor(nCache.cleanupInterval)
	go janitor.Run(nCache)
	nCache.janitor = janitor

	var wrapper = &cacheWrapper{}
	wrapper.cache = nCache
	runtime.SetFinalizer(wrapper, stopJanitor)

	return wrapper
}

func stopJanitor(c *cacheWrapper) {
	c.cache.close()
	c.cache.items = nil
	c.cache.janitor = nil
	c.cache = nil
}

func (this *cache) OnTick() {
	var evictedItems = this.items.DeleteExpired(this.onEvicted != nil)

	for _, item := range evictedItems {
		this.onEvicted(item.Key, item.Value)
	}
}

func (this *cache) close() {
	this.janitor.Close()
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

func (this *cache) Get(key string) (interface{}, bool) {
	var item, ok = this.items.Get(key)
	if ok == false {
		return nil, false
	}
	if item.Expired() {
		return nil, false
	}

	if this.hitTTL > 0 {
		item.Extend(this.hitTTL.Nanoseconds())
		this.items.Set(key, item)
	}

	return item.Value(), true
}

func (this *cache) Del(key string) {
	var item, ok = this.items.Pop(key)
	if this.onEvicted != nil && ok {
		this.onEvicted(key, item.Value())
	}
}

func (this *cache) Range(f func(key string, value interface{}) bool) {
	this.items.Range(func(key string, item *nmap.Item) bool {
		if item.Expired() {
			return true
		}
		f(key, item.Value())
		return true
	})
}

func (this *cache) Clear() {
	this.items.Range(func(key string, item *nmap.Item) bool {
		this.Del(key)
		return true
	})
}

func (this *cache) OnEvicted(f func(key string, value interface{})) {
	this.onEvicted = f
}
