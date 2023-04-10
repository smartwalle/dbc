package dbc

import (
	"github.com/smartwalle/queue/delay"
	"sync"
)

type shardCache[K Key, V any] struct {
	delayQueue delay.Queue[K]
	empty      V
	options    *options
	elements   map[K]Element[V]
	onEvicted  func(K, V)
	mu         sync.RWMutex
}

func newShard[K Key, V any](delayQueue delay.Queue[K], opts *options) *shardCache[K, V] {
	var shard = &shardCache[K, V]{}
	shard.options = opts
	shard.elements = make(map[K]Element[V])
	shard.delayQueue = delayQueue
	return shard
}

func (this *shardCache[K, V]) expireTick(key K) bool {
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

func (this *shardCache[K, V]) Set(key K, value V) bool {
	return this.SetEx(key, value, 0)
}

func (this *shardCache[K, V]) SetEx(key K, value V, seconds int64) bool {
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
		ele.value = value
		this.elements[key] = ele

		if expiration > 0 && remain < 3 {
			this.delayQueue.Enqueue(key, expiration)
		}
	} else {
		ele.expiration = expiration
		ele.value = value
		this.elements[key] = ele

		if expiration > 0 {
			this.delayQueue.Enqueue(key, expiration)
		}
	}

	this.mu.Unlock()
	return true
}

func (this *shardCache[K, V]) SetNx(key K, value V) bool {
	this.mu.Lock()
	var _, found = this.elements[key]
	if found == false {
		var ele = Element[V]{}
		ele.value = value
		ele.expiration = 0
		this.elements[key] = ele
	}
	this.mu.Unlock()
	return found == false
}

func (this *shardCache[K, V]) Expire(key K, seconds int64) {
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
		this.elements[key] = ele

		if expiration > 0 && remain < 3 {
			this.delayQueue.Enqueue(key, expiration)
		}
	}
	this.mu.Unlock()
}

func (this *shardCache[K, V]) Exists(key K) bool {
	this.mu.RLock()
	var _, found = this.elements[key]
	this.mu.RUnlock()
	return found
}

func (this *shardCache[K, V]) Get(key K) (V, bool) {
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
			this.elements[key] = ele
		}
	}

	this.mu.RUnlock()

	return ele.value, true
}

func (this *shardCache[K, V]) Del(key K) {
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

func (this *shardCache[K, V]) del(ele Element[V], key K) {
	delete(this.elements, key)

	ele.expiration = 0
	ele.value = this.empty
}

func (this *shardCache[K, V]) close() {
	this.mu.RLock()
	for key := range this.elements {
		this.mu.RUnlock()
		this.Del(key)
		this.mu.RLock()
	}
	this.mu.RUnlock()
}
