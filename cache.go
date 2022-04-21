package dbc

import (
	"github.com/smartwalle/dbc/internal/nmap"
	"github.com/smartwalle/queue/delay"
	"runtime"
	"sync"
	"time"
)

type Cache interface {
	Set(key string, value interface{})

	SetEx(key string, value interface{}, expiration int64)

	SetNx(key string, value interface{}) bool

	Expire(key string, expiration int64)

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

type option struct {
}

type cache struct {
	*option
	maps       *nmap.Map
	delayQueue delay.Queue
	onEvicted  func(string, interface{})
}

func New(opts ...Option) Cache {
	var nCache = &cache{}
	nCache.option = &option{}
	nCache.maps = nmap.New()

	for _, opt := range opts {
		if opt != nil {
			opt(nCache.option)
		}
	}

	nCache.delayQueue = delay.New(
		delay.WithTimeUnit(time.Second),
		delay.WithTimeProvider(func() int64 {
			return time.Now().Unix()
		}),
	)
	go nCache.run()

	var wrapper = &cacheWrapper{}
	wrapper.cache = nCache
	runtime.SetFinalizer(wrapper, stop)

	return wrapper
}

func stop(c *cacheWrapper) {
	c.cache.close()
	c.cache.maps = nil
	c.cache = nil
}

func (this *cache) run() {
	for {
		var value, expiration = this.delayQueue.Dequeue()

		if value == nil || expiration < 0 {
			return
		}

		var item, _ = value.(*nmap.Item)
		if item == nil {
			return
		}

		if this.checkExpired(item) {
			this.Del(item.Key())
		} else {
			if item.Expiration() > 0 {
				this.delayQueue.Enqueue(item, item.Expiration())
			}
		}
	}
}

func (this *cache) close() {
	this.delayQueue.Close()
}

func (this *cache) Set(key string, value interface{}) {
	this.SetEx(key, value, 0)
}

func (this *cache) SetEx(key string, value interface{}, expiration int64) {
	this.maps.GetShard(key).Do(func(mu *sync.RWMutex, items map[string]*nmap.Item) {
		mu.Lock()
		var item, _ = items[key]

		if item == nil {
			item = nmap.NewItem(key, value, expiration)
			items[key] = item
			if expiration > 0 {
				this.delayQueue.Enqueue(item, expiration)
			}
		} else {
			var remain = item.Expiration() - this.delayQueue.Now()

			item.UpdateValue(value)
			item.UpdateExpiration(expiration)

			if expiration > 0 && remain < 3 {
				this.delayQueue.Enqueue(item, expiration)
			}
		}
		mu.Unlock()
	})

	//var item, _ = this.maps.Get(key)
	//
	//if item == nil {
	//	item = nmap.NewItem(key, value, expiration)
	//	this.maps.Set(key, item)
	//
	//	if expiration > 0 {
	//		this.delayQueue.Enqueue(item, expiration)
	//	}
	//} else {
	//	var remain = item.Expiration() - this.delayQueue.Now()
	//
	//	item.UpdateValue(value)
	//	item.UpdateExpiration(expiration)
	//
	//	if expiration > 0 && remain < 3 {
	//		this.maps.Set(key, item)
	//		this.delayQueue.Enqueue(item, expiration)
	//	}
	//}
}

func (this *cache) SetNx(key string, value interface{}) bool {
	var nItem = nmap.NewItem(key, value, 0)
	return this.maps.SetNx(key, nItem)
}

func (this *cache) Expire(key string, expiration int64) {
	this.maps.GetShard(key).Do(func(mu *sync.RWMutex, items map[string]*nmap.Item) {
		mu.Lock()
		var item, ok = items[key]
		if ok {
			var remain = item.Expiration() - this.delayQueue.Now()

			item.UpdateExpiration(expiration)
			if expiration > 0 && remain < 3 {
				this.delayQueue.Enqueue(item, expiration)
			}
		}
		mu.Unlock()
	})

	//var item, ok = this.maps.Get(key)
	//if ok {
	//	var remain = item.Expiration() - this.delayQueue.Now()
	//
	//	item.UpdateExpiration(expiration)
	//
	//	if expiration > 0 && remain < 3 {
	//		this.maps.Set(key, item)
	//		this.delayQueue.Enqueue(item, expiration)
	//	}
	//}
}

func (this *cache) Exists(key string) bool {
	return this.maps.Exists(key)
}

func (this *cache) Get(key string) (interface{}, bool) {
	var item, ok = this.maps.Get(key)
	if ok == false {
		return nil, false
	}
	if this.checkExpired(item) {
		return nil, false
	}

	return item.Value(), true
}

func (this *cache) Del(key string) {
	var item, ok = this.maps.Pop(key)
	if this.onEvicted != nil && ok {
		this.onEvicted(key, item.Value())
	}
}

func (this *cache) Range(f func(key string, value interface{}) bool) {
	this.maps.Range(func(key string, item *nmap.Item) bool {
		if this.checkExpired(item) {
			return true
		}
		f(key, item.Value())
		return true
	})
}

func (this *cache) Clear() {
	this.maps.Range(func(key string, item *nmap.Item) bool {
		this.Del(key)
		return true
	})
}

func (this *cache) OnEvicted(f func(key string, value interface{})) {
	this.onEvicted = f
}

// checkExpired 检测是否过期
func (this *cache) checkExpired(item *nmap.Item) bool {
	var expiration = item.Expiration()
	if expiration == 0 {
		return false
	}
	return this.delayQueue.Now() >= expiration
}
