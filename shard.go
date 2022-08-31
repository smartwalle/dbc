package dbc

import (
	"github.com/smartwalle/queue/delay"
	"sync"
)

type shardCache[T any] struct {
	delayQueue delay.Queue[string]
	empty      T
	options    *options
	elements   map[string]*Element[T]
	onEvicted  func(string, T)
	mu         sync.RWMutex
}

func newShard[T any](delayQueue delay.Queue[string], opts *options) *shardCache[T] {
	var shard = &shardCache[T]{}
	shard.options = opts
	shard.elements = make(map[string]*Element[T])
	shard.delayQueue = delayQueue
	return shard
}

func (this *shardCache[T]) expireTick(key string) bool {
	this.mu.Lock()

	var ele, found = this.elements[key]
	if found && ele.expiration > 0 {
		if ele.expired(this.options.timeProvider()) {
			var value = ele.value

			this.del(ele, key)

			this.mu.Unlock()

			if this.onEvicted != nil {
				this.onEvicted(key, value)
			}
			return true
		} else {
			this.delayQueue.Enqueue(key, ele.expiration)
			this.mu.Unlock()
			return false
		}
	} else {
		this.mu.Unlock()
		return false
	}
}

func (this *shardCache[T]) Set(key string, value T) bool {
	return this.SetEx(key, value, 0)
}

func (this *shardCache[T]) SetEx(key string, value T, seconds int64) bool {
	var now = int64(0)
	var expiration = int64(0)
	if seconds > 0 {
		now = this.options.timeProvider()
		expiration = now + seconds
	}

	this.mu.Lock()
	var ele, _ = this.elements[key]

	if ele == nil {
		ele = &Element[T]{}
		ele.expiration = expiration
		ele.value = value
		this.elements[key] = ele

		if expiration > 0 {
			this.delayQueue.Enqueue(key, expiration)
		}
	} else {
		var remain = ele.expiration - now

		ele.expiration = expiration
		ele.value = value

		if expiration > 0 && remain < 3 {
			this.delayQueue.Enqueue(key, expiration)
		}
	}
	this.mu.Unlock()
	return true
}

func (this *shardCache[T]) SetNx(key string, value T) bool {
	var ele = &Element[T]{}
	ele.value = value
	ele.expiration = 0

	this.mu.Lock()
	var _, found = this.elements[key]
	if found == false {
		this.elements[key] = ele
	}
	this.mu.Unlock()
	return found == false
}

func (this *shardCache[T]) Expire(key string, seconds int64) {
	var now = int64(0)
	var expiration = int64(0)
	if seconds > 0 {
		now = this.options.timeProvider()
		expiration = now + seconds
	}

	this.mu.Lock()
	var ele, found = this.elements[key]
	if found {
		var remain = ele.expiration - now

		ele.expiration = expiration

		if expiration > 0 && remain < 3 {
			this.delayQueue.Enqueue(key, expiration)
		}
	}
	this.mu.Unlock()
}

func (this *shardCache[T]) Exists(key string) bool {
	this.mu.RLock()
	var _, found = this.elements[key]
	this.mu.RUnlock()
	return found
}

func (this *shardCache[T]) Get(key string) (T, bool) {
	this.mu.RLock()
	var ele, found = this.elements[key]
	if found == false {
		this.mu.RUnlock()
		return this.empty, false
	}

	if ele.expiration > 0 {
		var now = this.options.timeProvider()

		if ele.expired(now) {
			this.mu.RUnlock()
			return this.empty, false
		}

		if this.options.hitTTL > 0 && ele.expiration-now < this.options.hitTTL {
			// 忽略多个读操作同步进行可能引起的偏差
			ele.expiration = ele.expiration + this.options.hitTTL
		}
	}

	this.mu.RUnlock()

	return ele.value, true
}

func (this *shardCache[T]) Del(key string) {
	this.mu.Lock()
	var ele, found = this.elements[key]
	if found {
		var value = ele.value

		this.del(ele, key)

		this.mu.Unlock()

		if this.onEvicted != nil {
			this.onEvicted(key, value)
		}
	} else {
		this.mu.Unlock()
	}
}

func (this *shardCache[T]) del(ele *Element[T], key string) {
	delete(this.elements, key)

	ele.expiration = 0
	ele.value = this.empty
}

func (this *shardCache[T]) close() {
	this.mu.RLock()
	for key := range this.elements {
		this.mu.RUnlock()
		this.Del(key)
		this.mu.RLock()
	}
	this.mu.RUnlock()
}
