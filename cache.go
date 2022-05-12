package dbc

import (
	"github.com/smartwalle/queue/delay"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Cache interface {
	Set(key string, value interface{}) bool

	SetEx(key string, value interface{}, seconds int64) bool

	SetNx(key string, value interface{}) bool

	Expire(key string, seconds int64)

	Exists(key string) bool

	Get(key string) (interface{}, bool)

	Del(key string)

	Close()

	OnEvicted(func(key string, value interface{}))
}

type cacheWrapper struct {
	*cache
}

type Option func(opt *option)

type option struct {
	hitTTL int64
	now    func() int64
}

// WithHitTTL 设置访问命中延长过期时间
// 当调用 Get 方法有获取到数据，该数据有设置过期时间并且过期时间小于本方法指定的值，则在原有过期时间上加上本方法指定的值
func WithHitTTL(seconds int64) Option {
	return func(opt *option) {
		if seconds < 0 {
			seconds = 0
		}
		opt.hitTTL = seconds
	}
}

type cache struct {
	*option
	seed       uint32
	shard      uint32
	shards     []*shardCache
	delayQueue delay.Queue[string]
	closed     int32
}

func New(opts ...Option) Cache {
	var nCache = &cache{}
	nCache.option = &option{
		now: func() int64 {
			return time.Now().Unix()
		},
	}
	nCache.seed = rand.New(rand.NewSource(time.Now().Unix())).Uint32()
	nCache.shard = 32

	for _, opt := range opts {
		if opt != nil {
			opt(nCache.option)
		}
	}
	nCache.delayQueue = delay.New[string](
		delay.WithTimeUnit(time.Second),
		delay.WithTimeProvider(nCache.option.now),
	)

	nCache.shards = make([]*shardCache, nCache.shard)
	for i := uint32(0); i < nCache.shard; i++ {
		nCache.shards[i] = &shardCache{
			option:     nCache.option,
			RWMutex:    &sync.RWMutex{},
			elements:   make(map[string]*Element),
			delayQueue: nCache.delayQueue,
		}
	}

	go nCache.run()

	var wrapper = &cacheWrapper{}
	wrapper.cache = nCache
	runtime.SetFinalizer(wrapper, stop)

	return wrapper
}

func stop(c *cacheWrapper) {
	c.cache.Close()
}

func (this *cache) getShard(key string) *shardCache {
	var index = djb(this.seed, key) % this.shard
	return this.shards[index]
}

func (this *cache) run() {
	for {
		var key, expiration = this.delayQueue.Dequeue()

		if expiration < 0 {
			return
		}

		this.getShard(key).tick(key)
	}
}

func (this *cache) Set(key string, value interface{}) bool {
	return this.getShard(key).SetEx(key, value, 0)
}

func (this *cache) SetEx(key string, value interface{}, seconds int64) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}
	return this.getShard(key).SetEx(key, value, seconds)
}

func (this *cache) SetNx(key string, value interface{}) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}
	return this.getShard(key).SetNx(key, value)
}

func (this *cache) Expire(key string, seconds int64) {
	if atomic.LoadInt32(&this.closed) == 1 {
		return
	}
	this.getShard(key).Expire(key, seconds)
}

func (this *cache) Exists(key string) bool {
	return this.getShard(key).Exists(key)
}

func (this *cache) Get(key string) (interface{}, bool) {
	return this.getShard(key).Get(key)
}

func (this *cache) Del(key string) {
	this.getShard(key).Del(key)
}

func (this *cache) Close() {
	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		this.delayQueue.Close()
		for _, shard := range this.shards {
			go shard.close()
		}
	}
}

func (this *cache) OnEvicted(f func(key string, value interface{})) {
	for _, shard := range this.shards {
		shard.onEvicted = f
	}
}
