package nmap

// 参考 https://github.com/orcaman/concurrent-map

import (
	"math/rand"
	"sync"
)

const (
	kShardCount = uint32(32)
)

type Option func(opt *option)

func WithShardCount(count uint32) Option {
	return func(opt *option) {
		opt.shard = count
	}
}

func WithFNVHash() Option {
	return func(opt *option) {
		opt.hashSeed = kFNVSeed
		opt.hash = FNV1
	}
}

func WithBKDRHash() Option {
	return func(opt *option) {
		opt.hashSeed = kBKDRSeed
		opt.hash = BKDR
	}
}

func WithDJBHash() Option {
	return func(opt *option) {
		opt.hashSeed = rand.Uint32()
		opt.hash = DJB
	}
}

type option struct {
	hashSeed uint32
	hash     Hash
	shard    uint32
}

type Map struct {
	*option
	shards []*shardMap
}

type shardMap struct {
	*sync.RWMutex
	items map[string]*Item
}

func New(opts ...Option) *Map {
	var m = &Map{}
	m.option = &option{}

	for _, opt := range opts {
		opt(m.option)
	}
	if m.hash == nil {
		WithDJBHash()(m.option)
	}
	if m.shard == 0 {
		m.shard = kShardCount
	}

	m.shards = make([]*shardMap, m.shard)
	for i := uint32(0); i < m.shard; i++ {
		m.shards[i] = &shardMap{RWMutex: &sync.RWMutex{}, items: make(map[string]*Item)}
	}
	return m
}

func (this *Map) Lock(key string) {
	var shard = this.getShard(key)
	shard.Lock()
}

func (this *Map) Unlock(key string) {
	var shard = this.getShard(key)
	shard.Unlock()
}

func (this *Map) getShard(key string) *shardMap {
	var index = this.hash(this.hashSeed, key) % this.shard
	return this.shards[index]
}

func (this *Map) Set(key string, item *Item) {
	var shard = this.getShard(key)
	shard.Lock()
	shard.items[key] = item
	shard.Unlock()
}

func (this *Map) SetNx(key string, item *Item) bool {
	var shard = this.getShard(key)
	shard.Lock()
	var _, ok = shard.items[key]
	if ok == false {
		shard.items[key] = item
		shard.Unlock()
		return true
	}
	shard.Unlock()
	return false
}

func (this *Map) Get(key string) (*Item, bool) {
	var shard = this.getShard(key)
	shard.RLock()
	var item, ok = shard.items[key]
	shard.RUnlock()
	return item, ok
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
		shard.items = make(map[string]*Item)
		shard.Unlock()
	}
}

func (this *Map) Remove(key string) {
	var shard = this.getShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

func (this *Map) Pop(key string) (*Item, bool) {
	var shard = this.getShard(key)
	shard.Lock()
	var item, ok = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return item, ok
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

func (this *Map) Range(f func(key string, item *Item) bool) {
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

func (this *Map) Items() map[string]*Item {
	var nMap = make(map[string]*Item, this.Len())
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
