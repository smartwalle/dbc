package dbc

import (
	"github.com/smartwalle/dbc/internal"
	"github.com/smartwalle/nmap"
	"runtime"
	"time"
)

type Cache[T any] interface {
	Set(key string, value T)

	SetEx(key string, value T, expiration time.Duration)

	SetNx(key string, value T) bool

	Expire(key string, expiration time.Duration)

	Exists(key string) bool

	Get(key string) (T, bool)

	Del(key string)

	Range(f func(key string, value T) bool)

	Clear()

	OnEvicted(func(key string, value T))
}

type cacheWrapper[T any] struct {
	*cache[T]
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

type cache[T any] struct {
	*option
	items     *nmap.Map[internal.Item[T]]
	janitor   *internal.Janitor
	onEvicted func(string, T)
}

func New[T any](opts ...Option) Cache[T] {
	var nCache = &cache[T]{}
	nCache.option = &option{
		cleanupInterval: 0,
		hitTTL:          0,
	}
	nCache.items = nmap.New[internal.Item[T]]()

	for _, opt := range opts {
		opt(nCache.option)
	}

	var janitor = internal.NewJanitor(nCache.cleanupInterval)
	go janitor.Run(nCache)
	nCache.janitor = janitor

	var c = &cacheWrapper[T]{}
	c.cache = nCache
	runtime.SetFinalizer(c, stopJanitor[T])

	return c
}

func stopJanitor[T any](c *cacheWrapper[T]) {
	c.cache.close()
	c.cache.items = nil
	c.cache.janitor = nil
	c.cache = nil
}

func (this *cache[T]) OnTick() {
	this.items.Range(func(key string, item internal.Item[T]) bool {
		if item.Expired() {
			this.Del(key)
		}
		return true
	})
}

func (this *cache[T]) close() {
	this.janitor.Close()
}

func (this *cache[T]) Set(key string, value T) {
	this.SetEx(key, value, 0)
}

func (this *cache[T]) SetEx(key string, value T, expiration time.Duration) {
	var t int64
	if expiration > 0 {
		t = time.Now().Add(expiration).UnixNano()
	}
	var nItem = internal.NewItem[T](value, t)
	this.items.Set(key, nItem)
}

func (this *cache[T]) SetNx(key string, value T) bool {
	var nItem = internal.NewItem[T](value, 0)
	return this.items.SetNx(key, nItem)
}

func (this *cache[T]) Expire(key string, expiration time.Duration) {
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

func (this *cache[T]) Exists(key string) bool {
	return this.items.Exists(key)
}

func (this *cache[T]) Get(key string) (T, bool) {
	var item, ok = this.items.Get(key)

	var value T

	if ok == false {
		return value, false
	}
	if item.Expired() {
		return value, false
	}

	if this.hitTTL > 0 {
		item.Extend(this.hitTTL.Nanoseconds())
		this.items.Set(key, item)
	}
	value = item.Value()
	return value, true
}

func (this *cache[T]) Del(key string) {
	var item, ok = this.items.Pop(key)
	if this.onEvicted != nil && ok {
		this.onEvicted(key, item.Value())
	}
}

func (this *cache[T]) Range(f func(key string, value T) bool) {
	this.items.Range(func(key string, item internal.Item[T]) bool {
		f(key, item.Value())
		return true
	})
}

func (this *cache[T]) Clear() {
	this.items.Range(func(key string, item internal.Item[T]) bool {
		this.Del(key)
		return true
	})
}

func (this *cache[T]) OnEvicted(f func(key string, value T)) {
	this.onEvicted = f
}
