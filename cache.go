package dbc

import (
	"github.com/smartwalle/dbc/internal/nmap"
	"github.com/smartwalle/queue/delay"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Cache interface {
	Set(key string, value interface{}) bool

	SetEx(key string, value interface{}, expiration int64) bool

	SetNx(key string, value interface{}) bool

	Expire(key string, expiration int64)

	Exists(key string) bool

	Get(key string) (interface{}, bool)

	Del(key string)

	Range(f func(key string, value interface{}) bool)

	Close()

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
	delayQueue delay.Queue[string]
	onEvicted  func(string, interface{})
	closed     int32
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

	nCache.delayQueue = delay.New[string](
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
	c.cache.Close()
	//c.cache.maps = nil
	//c.cache = nil
}

func (this *cache) run() {
	for {
		var key, expiration = this.delayQueue.Dequeue()

		if expiration < 0 {
			return
		}

		//var key, _ = obj.(string)
		this.maps.GetShard(key).Do(func(mu *sync.RWMutex, elements map[string]*nmap.Element) {
			mu.Lock()

			var ele, ok = elements[key]
			if ok {
				if this.checkExpired(ele) {
					var value = ele.Value()

					delete(elements, key)
					ele.Reset()
					this.pool.Put(ele)
					mu.Unlock()

					if this.onEvicted != nil {
						this.onEvicted(key, value)
					}
				} else {
					if ele.Expiration() > 0 {
						this.delayQueue.Enqueue(key, ele.Expiration())
					}
					mu.Unlock()
				}
			} else {
				mu.Unlock()
			}
		})
	}
}

func (this *cache) Set(key string, value interface{}) bool {
	return this.SetEx(key, value, 0)
}

func (this *cache) SetEx(key string, value interface{}, expiration int64) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}

	this.maps.GetShard(key).Do(func(mu *sync.RWMutex, elements map[string]*nmap.Element) {
		mu.Lock()
		var ele, _ = elements[key]

		if ele == nil {

			ele = this.pool.Get().(*nmap.Element)
			ele.Init(key, value, expiration)

			elements[key] = ele
			if expiration > 0 {
				this.delayQueue.Enqueue(key, expiration)
			}
		} else {
			var remain = ele.Expiration() - now()

			ele.UpdateValue(value)
			ele.UpdateExpiration(expiration)

			if expiration > 0 && remain < 3 {
				this.delayQueue.Enqueue(key, expiration)
			}
		}
		mu.Unlock()
	})
	return true
}

func (this *cache) SetNx(key string, value interface{}) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}
	var ele = this.pool.Get().(*nmap.Element)
	ele.Init(key, value, 0)
	return this.maps.SetNx(key, ele)
}

func (this *cache) Expire(key string, expiration int64) {
	if atomic.LoadInt32(&this.closed) == 1 {
		return
	}
	this.maps.GetShard(key).Do(func(mu *sync.RWMutex, elements map[string]*nmap.Element) {
		mu.Lock()
		var ele, ok = elements[key]
		if ok {
			var remain = ele.Expiration() - now()

			ele.UpdateExpiration(expiration)
			if expiration > 0 && remain < 3 {
				this.delayQueue.Enqueue(key, expiration)
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
		return nil, false
	}

	return ele.Value(), true
}

func (this *cache) Del(key string) {
	var ele, ok = this.maps.Pop(key)
	if ok && this.onEvicted != nil {
		this.onEvicted(key, ele.Value())

		if ele.Expiration() <= 0 {
			ele.Reset()
			this.pool.Put(ele)
		}
	}
}

func (this *cache) Range(f func(key string, value interface{}) bool) {
	this.maps.Range(func(key string, ele *nmap.Element) bool {
		if this.checkExpired(ele) {
			return true
		}
		f(key, ele.Value())
		return true
	})
}

func (this *cache) Close() {
	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		this.delayQueue.Close()
		this.maps.Range(func(key string, ele *nmap.Element) bool {
			this.Del(key)
			return true
		})
	}
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
