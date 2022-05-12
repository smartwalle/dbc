package dbc

import (
	"github.com/smartwalle/queue/delay"
	"sync"
)

type shardCache[T any] struct {
	*option
	*sync.RWMutex
	elements   map[string]*Element[T]
	delayQueue delay.Queue[string]
	onEvicted  func(string, T)
	empty      T
}

func newShard[T any](delayQueue delay.Queue[string], opt *option) *shardCache[T] {
	var shard = &shardCache[T]{}
	shard.option = opt
	shard.RWMutex = &sync.RWMutex{}
	shard.elements = make(map[string]*Element[T])
	shard.delayQueue = delayQueue
	return shard
}

func (this *shardCache[T]) tick(key string) {
	this.Lock()

	var ele, ok = this.elements[key]
	if ok {
		if this.checkExpired(ele) {
			var value = ele.value

			delete(this.elements, key)
			ele.value = this.empty
			ele.expiration = 0

			this.Unlock()

			if this.onEvicted != nil {
				this.onEvicted(key, value)
			}
		} else {
			if ele.expiration > 0 {
				this.delayQueue.Enqueue(key, ele.expiration)
			}
			this.Unlock()
		}
	} else {
		this.Unlock()
	}
}

func (this *shardCache[T]) Set(key string, value T) bool {
	return this.SetEx(key, value, 0)
}

func (this *shardCache[T]) SetEx(key string, value T, seconds int64) bool {
	var expiration = int64(0)
	if seconds > 0 {
		expiration = this.now() + seconds
	}

	this.Lock()
	var ele, _ = this.elements[key]

	if ele == nil {
		ele = &Element[T]{}
		ele.value = value
		ele.expiration = expiration
		this.elements[key] = ele

		if expiration > 0 {
			this.delayQueue.Enqueue(key, expiration)
		}
	} else {
		var remain = ele.expiration - this.now()

		ele.value = value
		ele.expiration = expiration

		if expiration > 0 && remain < 3 {
			this.delayQueue.Enqueue(key, expiration)
		}
	}
	this.Unlock()
	return true
}

func (this *shardCache[T]) SetNx(key string, value T) bool {
	var ele = &Element[T]{}
	ele.value = value
	ele.expiration = 0

	this.Lock()
	var _, found = this.elements[key]
	if found == false {
		this.elements[key] = ele
	}
	this.Unlock()
	return found == false
}

func (this *shardCache[T]) Expire(key string, seconds int64) {
	var expiration = int64(0)
	if seconds > 0 {
		expiration = this.now() + seconds
	}

	this.Lock()
	var ele, ok = this.elements[key]
	if ok {
		var remain = ele.expiration - this.now()

		ele.expiration = expiration

		if expiration > 0 && remain < 3 {
			this.delayQueue.Enqueue(key, expiration)
		}
	}
	this.Unlock()
}

func (this *shardCache[T]) Exists(key string) bool {
	this.RLock()
	var _, found = this.elements[key]
	this.RUnlock()
	return found
}

func (this *shardCache[T]) Get(key string) (T, bool) {
	this.Lock()
	var ele, found = this.elements[key]
	if found == false {
		this.Unlock()
		return this.empty, false
	}

	if this.checkExpired(ele) {
		this.Unlock()
		return this.empty, false
	}

	if this.hitTTL > 0 && ele.expiration > 0 && ele.expiration-this.now() < this.hitTTL {
		ele.expiration = ele.expiration + this.hitTTL
	}
	this.Unlock()

	return ele.value, true
}

func (this *shardCache[T]) Del(key string) {
	this.Lock()
	var ele, found = this.elements[key]
	if found {
		delete(this.elements, key)

		if this.onEvicted != nil {
			this.onEvicted(key, ele.value)
		}

		if ele.expiration <= 0 {
			ele.value = this.empty
			ele.expiration = 0
		}
	}
	this.Unlock()
}

func (this *shardCache[T]) close() {
	this.RLock()
	for key := range this.elements {
		this.RUnlock()
		this.Del(key)
		this.RLock()
	}
	this.RUnlock()
}

// checkExpired 检测是否过期
func (this *shardCache[T]) checkExpired(ele *Element[T]) bool {
	return ele.expiration > 0 && this.now() >= ele.expiration
}
