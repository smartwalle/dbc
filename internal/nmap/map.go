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
	elements map[string]*Element
}

func (this *shardMap) Do(f func(mu *sync.RWMutex, elements map[string]*Element)) {
	if f != nil {
		f(this.RWMutex, this.elements)
	}
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
		m.shards[i] = &shardMap{RWMutex: &sync.RWMutex{}, elements: make(map[string]*Element)}
	}
	return m
}

func (this *Map) GetShard(key string) *shardMap {
	var index = this.hash(this.hashSeed, key) % this.shard
	return this.shards[index]
}

func (this *Map) Set(key string, ele *Element) {
	var shard = this.GetShard(key)
	shard.Lock()
	shard.elements[key] = ele
	shard.Unlock()
}

func (this *Map) SetNx(key string, ele *Element) bool {
	var shard = this.GetShard(key)
	shard.Lock()
	var _, ok = shard.elements[key]
	if ok == false {
		shard.elements[key] = ele
		shard.Unlock()
		return true
	}
	shard.Unlock()
	return false
}

func (this *Map) Get(key string) (*Element, bool) {
	var shard = this.GetShard(key)
	shard.RLock()
	var ele, ok = shard.elements[key]
	shard.RUnlock()
	return ele, ok
}

func (this *Map) Exists(key string) bool {
	var shard = this.GetShard(key)
	shard.RLock()
	var _, ok = shard.elements[key]
	shard.RUnlock()
	return ok
}

//func (this *Map) RemoveAll() {
//	for i := uint32(0); i < this.shard; i++ {
//		var shard = this.shards[i]
//		shard.Lock()
//		shard.elements = make(map[string]*Element)
//		shard.Unlock()
//	}
//}
//
//func (this *Map) Remove(key string) {
//	var shard = this.GetShard(key)
//	shard.Lock()
//	delete(shard.elements, key)
//	shard.Unlock()
//}

func (this *Map) Pop(key string) (*Element, bool) {
	var shard = this.GetShard(key)
	shard.Lock()
	var ele, ok = shard.elements[key]
	delete(shard.elements, key)
	shard.Unlock()
	return ele, ok
}

//func (this *Map) Len() int {
//	var count = 0
//	for i := uint32(0); i < this.shard; i++ {
//		var shard = this.shards[i]
//		shard.RLock()
//		count += len(shard.elements)
//		shard.RUnlock()
//	}
//	return count
//}

func (this *Map) Range(f func(key string, ele *Element) bool) {
	if f == nil {
		return
	}
	for i := uint32(0); i < this.shard; i++ {
		var shard = this.shards[i]
		shard.RLock()
		for k, v := range shard.elements {
			shard.RUnlock()
			if f(k, v) == false {
				return
			}
			shard.RLock()
		}
		shard.RUnlock()
	}
}

//func (this *Map) Elements() map[string]*Element {
//	var nMap = make(map[string]*Element, this.Len())
//	for i := uint32(0); i < this.shard; i++ {
//		var shard = this.shards[i]
//		shard.RLock()
//		for k, v := range shard.elements {
//			nMap[k] = v
//		}
//		shard.RUnlock()
//	}
//	return nMap
//}

//func (this *Map) Elements() map[string]*Element {
//	var nMap = make(map[string]*Element, this.Len())
//	for i := uint32(0); i < this.shard; i++ {
//		var shard = this.shards[i]
//		shard.RLock()
//		for k, v := range shard.elements {
//			nMap[k] = v
//		}
//		shard.RUnlock()
//	}
//	return nMap
//}

//func (this *Map) Keys() []string {
//	var nKeys = make([]string, 0, this.Len())
//	for i := uint32(0); i < this.shard; i++ {
//		var shard = this.shards[i]
//		shard.RLock()
//		for k := range shard.elements {
//			nKeys = append(nKeys, k)
//		}
//		shard.RUnlock()
//	}
//	return nKeys
//}
