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
	pool       *sync.Pool
	delayQueue delay.Queue
	onEvicted  func(string, interface{})
}

func New(opts ...Option) Cache {
	var nCache = &cache{}
	nCache.option = &option{}
	nCache.maps = nmap.New()
	nCache.pool = &sync.Pool{
		New: func() interface{} {
			return &nmap.Element{}
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(nCache.option)
		}
	}

	nCache.delayQueue = delay.New(
		delay.WithTimeUnit(time.Second),
		delay.WithTimeProvider(now),
	)
	go nCache.run()

	var wrapper = &cacheWrapper{}
	wrapper.cache = nCache
	runtime.SetFinalizer(wrapper, stop)

	return wrapper
}

func now() int64 {
	return time.Now().Unix()
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

		var ele, _ = value.(*nmap.Element)
		if ele == nil {
			return
		}

		var key = ele.Key()

		this.maps.GetShard(key).Do(func(mu *sync.RWMutex, elements map[string]*nmap.Element) {
			mu.Lock()

			if this.checkExpired(ele) {
				var _, ok = elements[key]
				if ok {
					delete(elements, key)
				}
				mu.Unlock()

				if ok && this.onEvicted != nil {
					this.onEvicted(key, ele.Value())
				}

				this.release(ele)
			} else {
				if ele.Expiration() > 0 {
					this.delayQueue.Enqueue(ele, ele.Expiration())
				}
				mu.Unlock()
			}
		})
	}
}

func (this *cache) close() {
	this.delayQueue.Close()
}

func (this *cache) Set(key string, value interface{}) {
	this.SetEx(key, value, 0)
}

func (this *cache) SetEx(key string, value interface{}, expiration int64) {
	this.maps.GetShard(key).Do(func(mu *sync.RWMutex, elements map[string]*nmap.Element) {
		mu.Lock()
		var ele, _ = elements[key]

		if ele == nil {

			ele = this.pool.Get().(*nmap.Element)
			ele.Init(key, value, expiration)
			//ele = nmap.NewElement(key, value, expiration)

			elements[key] = ele
			if expiration > 0 {
				this.delayQueue.Enqueue(ele, expiration)
			}
		} else {
			var remain = ele.Expiration() - now()

			ele.UpdateValue(value)
			ele.UpdateExpiration(expiration)

			if expiration > 0 && remain < 3 {
				this.delayQueue.Enqueue(ele, expiration)
			}
		}
		mu.Unlock()
	})
}

func (this *cache) SetNx(key string, value interface{}) bool {
	//var ele = nmap.NewElement(key, value, 0)
	var ele = this.pool.Get().(*nmap.Element)
	ele.Init(key, value, 0)
	return this.maps.SetNx(key, ele)
}

func (this *cache) Expire(key string, expiration int64) {
	this.maps.GetShard(key).Do(func(mu *sync.RWMutex, elements map[string]*nmap.Element) {
		mu.Lock()
		var ele, ok = elements[key]
		if ok {
			var remain = ele.Expiration() - now()

			ele.UpdateExpiration(expiration)
			if expiration > 0 && remain < 3 {
				this.delayQueue.Enqueue(ele, expiration)
			}
		}
		mu.Unlock()
	})
}

func (this *cache) Exists(key string) bool {
	return this.maps.Exists(key)
}

func (this *cache) Get(key string) (interface{}, bool) {
	var ele, ok = this.maps.Get(key)
	if ok == false {
		return nil, false
	}
	if this.checkExpired(ele) {
		this.release(ele)
		return nil, false
	}

	return ele.Value(), true
}

func (this *cache) Del(key string) {
	var ele, ok = this.maps.Pop(key)
	if ok && this.onEvicted != nil {
		this.onEvicted(key, ele.Value())

		this.release(ele)
	}
}

func (this *cache) Range(f func(key string, value interface{}) bool) {
	this.maps.Range(func(key string, ele *nmap.Element) bool {
		if this.checkExpired(ele) {
			this.release(ele)
			return true
		}
		f(key, ele.Value())
		return true
	})
}

func (this *cache) Clear() {
	this.maps.Range(func(key string, ele *nmap.Element) bool {
		this.Del(key)
		return true
	})
}

func (this *cache) OnEvicted(f func(key string, value interface{})) {
	this.onEvicted = f
}

// checkExpired 检测是否过期
func (this *cache) checkExpired(ele *nmap.Element) bool {
	var expiration = ele.Expiration()
	if expiration == 0 {
		return false
	}
	return now() >= expiration
}

func (this *cache) release(ele *nmap.Element) {
	if ele != nil {
		ele.Reset()
		this.pool.Put(ele)
	}
}
