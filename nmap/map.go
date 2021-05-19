package nmap

// 参考 https://github.com/orcaman/concurrent-map

import (
	"sync"
)

const (
	kShardCount = uint32(32)
)

type Option func(p *Map)

func WithShardCount(count uint32) Option {
	return func(m *Map) {
		m.shard = count
	}
}

func WithFNVHash() Option {
	return func(m *Map) {
		m.hash = FNV1
	}
}

func WithBKDRHash() Option {
	return func(m *Map) {
		m.hash = BKDR
	}
}

func New(opts ...Option) *Map {
	var m = &Map{}
	for _, opt := range opts {
		opt(m)
	}
	if m.hash == nil {
		m.hash = DJB
	}
	if m.shard == 0 {
		m.shard = kShardCount
	}

	m.shards = make([]*shardMap, m.shard)
	for i := uint32(0); i < m.shard; i++ {
		m.shards[i] = &shardMap{RWMutex: &sync.RWMutex{}, items: make(map[string]Item)}
	}
	return m
}

type Map struct {
	hash   Hash
	shard  uint32
	shards []*shardMap
}

type shardMap struct {
	*sync.RWMutex
	items map[string]Item
}

func (this *Map) getShard(key string) *shardMap {
	var index = this.hash(key) % this.shard
	return this.shards[index]
}

func (this *Map) Set(key string, value Item) {
	var shard = this.getShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

func (this *Map) SetNx(key string, value Item) {
	var shard = this.getShard(key)
	shard.Lock()
	var _, ok = shard.items[key]
	if ok == false {
		shard.items[key] = value
	}
	shard.Unlock()
}

func (this *Map) Get(key string) (Item, bool) {
	var shard = this.getShard(key)
	shard.RLock()
	var value, ok = shard.items[key]
	shard.RUnlock()
	return value, ok
}

func (this *Map) Exists(key string) bool {
	var shard = this.getShard(key)
	shard.RLock()
	var _, ok = shard.items[key]
	shard.RUnlock()
	return ok
}

func (this *Map) RemoveAll() {
	for i := uint32(0); i < this.shard; i++ {
		var shard = this.shards[i]
		shard.Lock()
		shard.items = make(map[string]Item)
		shard.Unlock()
	}
}

func (this *Map) Remove(key string) {
	var shard = this.getShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

func (this *Map) Pop(key string) (Item, bool) {
	var shard = this.getShard(key)
	shard.Lock()
	var value, ok = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return value, ok
}

func (this *Map) Len() int {
	var count = 0
	for i := uint32(0); i < this.shard; i++ {
		var shard = this.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

func (this *Map) Range(f func(key string, value Item) bool) {
	if f == nil {
		return
	}
	for i := uint32(0); i < this.shard; i++ {
		var shard = this.shards[i]
		shard.RLock()
		for k, v := range shard.items {
			shard.RUnlock()
			if f(k, v) == false {
				return
			}
			shard.RLock()
		}
		shard.RUnlock()
	}
}

func (this *Map) Items() map[string]Item {
	var nMap = make(map[string]Item, this.Len())
	for i := uint32(0); i < this.shard; i++ {
		var shard = this.shards[i]
		shard.RLock()
		for k, v := range shard.items {
			nMap[k] = v
		}
		shard.RUnlock()
	}
	return nMap
}

func (this *Map) Keys() []string {
	var nKeys = make([]string, 0, this.Len())
	for i := uint32(0); i < this.shard; i++ {
		var shard = this.shards[i]
		shard.RLock()
		for k := range shard.items {
			nKeys = append(nKeys, k)
		}
		shard.RUnlock()
	}
	return nKeys
}