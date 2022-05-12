package dbc

import (
	"github.com/smartwalle/queue/delay"
	"math/rand"
	"sync/atomic"
	"time"
)

type Cache[T any] interface {
	Set(key string, value T) bool

	SetEx(key string, value T, seconds int64) bool

	SetNx(key string, value T) bool

	Expire(key string, seconds int64)

	Exists(key string) bool

	Get(key string) (T, bool)

	Del(key string)

	Close()

	OnEvicted(func(key string, value T))
}

type Option func(opt *option)

type option struct {
	now    func() int64
	hitTTL int64
	seed   uint32
	shard  uint32
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

type cache[T any] struct {
	*option
	shards     []*shardCache[T]
	delayQueue delay.Queue[string]
	closed     int32
}

func New[T any](opts ...Option) Cache[T] {
	var nCache = &cache[T]{}
	nCache.option = &option{
		seed:  rand.New(rand.NewSource(time.Now().Unix())).Uint32(),
		shard: 32,
		now: func() int64 {
			return time.Now().Unix()
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(nCache.option)
		}
	}
	nCache.delayQueue = delay.New[string](
		delay.WithTimeUnit(time.Second),
		delay.WithTimeProvider(nCache.now),
	)

	nCache.shards = make([]*shardCache[T], nCache.shard)
	for i := uint32(0); i < nCache.shard; i++ {
		nCache.shards[i] = newShard[T](nCache.delayQueue, nCache.option)
	}

	go nCache.run()

	return nCache
}

func (this *cache[T]) getShard(key string) *shardCache[T] {
	var index = djb(this.seed, key) % this.shard
	return this.shards[index]
}

func (this *cache[T]) run() {
	for {
		var key, expiration = this.delayQueue.Dequeue()

		if expiration < 0 {
			return
		}

		this.getShard(key).tick(key)
	}
}

func (this *cache[T]) Set(key string, value T) bool {
	return this.getShard(key).SetEx(key, value, 0)
}

func (this *cache[T]) SetEx(key string, value T, seconds int64) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}
	return this.getShard(key).SetEx(key, value, seconds)
}

func (this *cache[T]) SetNx(key string, value T) bool {
	if atomic.LoadInt32(&this.closed) == 1 {
		return false
	}
	return this.getShard(key).SetNx(key, value)
}

func (this *cache[T]) Expire(key string, seconds int64) {
	if atomic.LoadInt32(&this.closed) == 1 {
		return
	}
	this.getShard(key).Expire(key, seconds)
}

func (this *cache[T]) Exists(key string) bool {
	return this.getShard(key).Exists(key)
}

func (this *cache[T]) Get(key string) (T, bool) {
	return this.getShard(key).Get(key)
}

func (this *cache[T]) Del(key string) {
	this.getShard(key).Del(key)
}

func (this *cache[T]) Close() {
	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		this.delayQueue.Close()
		for _, shard := range this.shards {
			go shard.close()
		}
	}
}

func (this *cache[T]) OnEvicted(f func(key string, value T)) {
	for _, shard := range this.shards {
		shard.onEvicted = f
	}
}
