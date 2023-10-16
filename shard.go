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

func (s *shardCache[K, V]) expireTick(key K) bool {
	s.mu.Lock()

	var ele, found = s.elements[key]
	if found && ele.expiration > 0 {
		if ele.expired(s.options.timeProvider()) {
			var value = ele.value
			s.del(ele, key)
			s.mu.Unlock()

			if s.onEvicted != nil {
				s.onEvicted(key, value)
			}
			return true
		} else {
			s.delayQueue.Enqueue(key, ele.expiration)
			s.mu.Unlock()
			return false
		}
	} else {
		s.mu.Unlock()
		return false
	}
}

func (s *shardCache[K, V]) Set(key K, value V) bool {
	return s.SetEx(key, value, 0)
}

func (s *shardCache[K, V]) SetEx(key K, value V, seconds int64) bool {
	var now = int64(0)
	var expiration = int64(0)
	if seconds > 0 {
		now = s.options.timeProvider()
		expiration = now + seconds
	}

	s.mu.Lock()
	var ele, found = s.elements[key]

	if found {
		var remain = ele.expiration - now

		ele.expiration = expiration
		ele.value = value
		s.elements[key] = ele

		if expiration > 0 && remain < 3 {
			s.delayQueue.Enqueue(key, expiration)
		}
	} else {
		ele.expiration = expiration
		ele.value = value
		s.elements[key] = ele

		if expiration > 0 {
			s.delayQueue.Enqueue(key, expiration)
		}
	}

	s.mu.Unlock()
	return true
}

func (s *shardCache[K, V]) SetNx(key K, value V) bool {
	s.mu.Lock()
	var _, found = s.elements[key]
	if !found {
		var ele = Element[V]{}
		ele.value = value
		ele.expiration = 0
		s.elements[key] = ele
		s.mu.Unlock()
		return true
	}
	s.mu.Unlock()
	return false
}

func (s *shardCache[K, V]) Expire(key K, seconds int64) {
	var now = int64(0)
	var expiration = int64(0)
	if seconds > 0 {
		now = s.options.timeProvider()
		expiration = now + seconds
	}

	s.mu.Lock()
	var ele, found = s.elements[key]
	if found {
		var remain = ele.expiration - now

		ele.expiration = expiration
		s.elements[key] = ele

		if expiration > 0 && remain < 3 {
			s.delayQueue.Enqueue(key, expiration)
		}
	}
	s.mu.Unlock()
}

func (s *shardCache[K, V]) Exists(key K) bool {
	s.mu.RLock()
	var _, found = s.elements[key]
	s.mu.RUnlock()
	return found
}

func (s *shardCache[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	var ele, found = s.elements[key]
	if !found {
		s.mu.RUnlock()
		return s.empty, false
	}

	if ele.expiration > 0 {
		var now = s.options.timeProvider()

		if ele.expired(now) {
			s.mu.RUnlock()
			return s.empty, false
		}

		if s.options.hitTTL > 0 && ele.expiration-now < s.options.hitTTL {
			// 忽略多个读操作同步进行可能引起的偏差
			ele.expiration = ele.expiration + s.options.hitTTL
			s.elements[key] = ele
		}
	}

	s.mu.RUnlock()

	return ele.value, true
}

func (s *shardCache[K, V]) Del(key K) {
	s.mu.Lock()
	var ele, found = s.elements[key]
	if found {
		var value = ele.value
		s.del(ele, key)
		s.mu.Unlock()

		if s.onEvicted != nil {
			s.onEvicted(key, value)
		}
	} else {
		s.mu.Unlock()
	}
}

func (s *shardCache[K, V]) del(ele Element[V], key K) {
	delete(s.elements, key)

	ele.expiration = 0
	ele.value = s.empty
}

func (s *shardCache[K, V]) close() {
	s.mu.RLock()
	for key := range s.elements {
		s.mu.RUnlock()
		s.Del(key)
		s.mu.RLock()
	}
	s.mu.RUnlock()
}
