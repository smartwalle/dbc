package dbc

import (
	"github.com/smartwalle/queue/delay"
	"sync"
)

type shardCache struct {
	*option
	*sync.RWMutex
	elements   map[string]*Element
	delayQueue delay.Queue[string]
	onEvicted  func(string, interface{})
}

func (this *shardCache) tick(key string) {
	this.Lock()

	var ele, ok = this.elements[key]
	if ok {
		if this.checkExpired(ele) {
			var value = ele.value

			delete(this.elements, key)
			ele.value = nil
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

func (this *shardCache) Set(key string, value interface{}) bool {
	return this.SetEx(key, value, 0)
}

func (this *shardCache) SetEx(key string, value interface{}, seconds int64) bool {
	var expiration = int64(0)
	if seconds > 0 {
		expiration = this.now() + seconds
	}

	this.Lock()
	var ele, _ = this.elements[key]

	if ele == nil {
		ele = &Element{}
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

func (this *shardCache) SetNx(key string, value interface{}) bool {
	var ele = &Element{}
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

func (this *shardCache) Expire(key string, seconds int64) {
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

func (this *shardCache) Exists(key string) bool {
	this.RLock()
	var _, found = this.elements[key]
	this.RUnlock()
	return found
}

func (this *shardCache) Get(key string) (interface{}, bool) {
	this.Lock()
	var ele, found = this.elements[key]
	if found == false {
		this.Unlock()
		return nil, false
	}

	if this.checkExpired(ele) {
		this.Unlock()
		return nil, false
	}

	if this.hitTTL > 0 && ele.expiration > 0 && ele.expiration-this.now() < this.hitTTL {
		ele.expiration = ele.expiration + this.hitTTL
	}
	this.Unlock()

	return ele.value, true
}

func (this *shardCache) Del(key string) {
	this.Lock()
	var ele, found = this.elements[key]
	if found {
		delete(this.elements, key)

		if this.onEvicted != nil {
			this.onEvicted(key, ele.value)
		}

		if ele.expiration <= 0 {
			ele.value = nil
			ele.expiration = 0
		}
	}
	this.Unlock()
}

func (this *shardCache) close() {
	this.RLock()
	for key := range this.elements {
		this.RUnlock()
		this.Del(key)
		this.RLock()
	}
	this.RUnlock()
}

// checkExpired 检测是否过期
func (this *shardCache) checkExpired(ele *Element) bool {
	return ele.expiration > 0 && this.now() >= ele.expiration
}
