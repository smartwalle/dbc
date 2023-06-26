package dbc

import (
	"github.com/smartwalle/dbc/internal"
	"github.com/smartwalle/queue/delay"
	"sync/atomic"
	"time"
)

const (
	ShardCount = 32 // 分片数量
)

type Key = internal.Key

type Cache[K Key, V any] interface {
	Set(key K, value V) bool

	SetEx(key K, value V, seconds int64) bool

	SetNx(key K, value V) bool

	Expire(key K, seconds int64)

	Exists(key K) bool

	Get(key K) (V, bool)

	Del(key K)

	Len() int

	Close()

	OnEvicted(func(key K, value V))
}

type Option func(opt *options)

type options struct {
	hitTTL       int64
	timeProvider func() int64
}

// WithHitTTL 设置访问命中延长过期时间
// 当调用 Get 方法有获取到数据，该数据有设置过期时间并且过期时间小于本方法指定的值，则在原有过期时间上加上本方法指定的值
func WithHitTTL(seconds int64) Option {
	return func(opts *options) {
		if seconds < 0 {
			seconds = 0
		}
		opts.hitTTL = seconds
	}
}

func WithTimeProvider(fn func() int64) Option {
	return func(opt *options) {
		opt.timeProvider = fn
	}
}

type cache[K Key, V any] struct {
	delayQueue delay.Queue[K]
	options    *options
	sharding   func(key K) uint32
	shardCount uint32
	shards     []*shardCache[K, V]
	closed     int32
}

func New[V any](opts ...Option) Cache[string, V] {
	return NewCache[string, V](DJBSharding(), opts...)
}

func NewCache[K Key, V any](sharding func(key K) uint32, opts ...Option) Cache[K, V] {
	var nCache = &cache[K, V]{}
	nCache.options = &options{}
	nCache.sharding = sharding
	nCache.shardCount = ShardCount

	for _, opt := range opts {
		if opt != nil {
			opt(nCache.options)
		}
	}
	if nCache.options.timeProvider == nil {
		nCache.options.timeProvider = func() int64 {
			return time.Now().Unix()
		}
	}

	nCache.delayQueue = delay.New[K](
		delay.WithTimeUnit(time.Second),
		delay.WithTimeProvider(nCache.options.timeProvider),
	)

	nCache.shards = make([]*shardCache[K, V], nCache.shardCount)
	for i := uint32(0); i < nCache.shardCount; i++ {
		nCache.shards[i] = newShard[K, V](nCache.delayQueue, nCache.options)
	}

	go nCache.run()

	return nCache
}

func (this *cache[K, V]) getShard(key K) *shardCache[K, V] {
	var index = this.sharding(key)
	return this.shards[index]
}

func (this *cache[K, V]) run() {
	for {
		var key, expiration = this.delayQueue.Dequeue()

		if expiration < 0 {
			return
		}

		this.getShard(key).expireTick(key)
	}
}

func (this *cache[K, V]) Set(key K, value V) bool {
	return this.SetEx(key, value, 0)
}

func (this *cache[K, V]) SetEx(key K, value V, seconds int64) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}
	return this.getShard(key).SetEx(key, value, seconds)
}

func (this *cache[K, V]) SetNx(key K, value V) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}
	return this.getShard(key).SetNx(key, value)
}

func (this *cache[K, V]) Expire(key K, seconds int64) {
	if atomic.LoadInt32(&this.closed) == 1 {
		return
	}
	this.getShard(key).Expire(key, seconds)
}

func (this *cache[K, V]) Exists(key K) bool {
	return this.getShard(key).Exists(key)
}

func (this *cache[K, V]) Get(key K) (V, bool) {
	return this.getShard(key).Get(key)
}

func (this *cache[K, V]) Del(key K) {
	this.getShard(key).Del(key)
}

func (this *cache[K, V]) Len() int {
	var count = 0
	for i := uint32(0); i < this.shardCount; i++ {
		var shard = this.shards[i]
		shard.mu.RLock()
		count += len(shard.elements)
		shard.mu.RUnlock()
	}
	return count
}

func (this *cache[K, V]) Close() {
	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		this.delayQueue.Close()
		for _, shard := range this.shards {
			go shard.close()
		}
	}
}

func (this *cache[K, V]) OnEvicted(fn func(key K, value V)) {
	for _, shard := range this.shards {
		shard.onEvicted = fn
	}
}
