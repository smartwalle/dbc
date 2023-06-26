package lru

import (
	"github.com/smartwalle/dbc/internal"
	"sync"
)

type Key = internal.Key

type options struct {
	maxSize  int
	initSize int
}

const kDefaultSize = 512

type Option func(opts *options)

func WithMaxSize(size int) Option {
	return func(opts *options) {
		if size <= 0 {
			size = kDefaultSize
		}
		opts.maxSize = size
	}
}

func WithInitSize(size int) Option {
	return func(opts *options) {
		if size <= 0 {
			size = 0
		}
		opts.initSize = size
	}
}

type Cache[K Key, V any] struct {
	mu        sync.Mutex
	size      int
	list      *internal.List[K, V]
	elements  map[K]*internal.Element[K, V]
	onEvicted func(K, V)
}

func New[K Key, V any](opts ...Option) *Cache[K, V] {
	var nOpts = &options{
		maxSize: kDefaultSize,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(nOpts)
		}
	}
	if nOpts.initSize > nOpts.maxSize {
		nOpts.initSize = nOpts.maxSize
	}

	var nLRU = &Cache[K, V]{}
	nLRU.size = nOpts.maxSize
	nLRU.list = internal.New[K, V]()
	nLRU.elements = make(map[K]*internal.Element[K, V], nOpts.initSize)
	return nLRU
}

func (this *Cache[K, V]) Set(key K, value V) bool {
	this.mu.Lock()
	var ele, found = this.elements[key]
	if found {
		ele.Value = value
		this.list.MoveToFront(ele)
		this.mu.Unlock()
		return true
	}

	ele = this.list.PushFront(key, value)
	this.elements[key] = ele
	var needEvict = this.list.Len() > this.size

	if needEvict {
		var last = this.list.Back()
		this.removeElement(last)
		this.mu.Unlock()
		this.evict(last)
		return true
	}
	this.mu.Unlock()
	return true
}

func (this *Cache[K, V]) Exists(key K) bool {
	this.mu.Lock()
	var _, found = this.elements[key]
	this.mu.Unlock()
	return found
}

func (this *Cache[K, V]) Get(key K) (value V, ok bool) {
	this.mu.Lock()
	if ele, found := this.elements[key]; found {
		this.list.MoveToFront(ele)
		this.mu.Unlock()
		return ele.Value, true
	}
	this.mu.Unlock()
	return
}

func (this *Cache[K, V]) Del(key K) {
	this.mu.Lock()
	if ele, found := this.elements[key]; found {
		this.removeElement(ele)
		this.mu.Unlock()
		this.evict(ele)
		return
	}
	this.mu.Unlock()
}

func (this *Cache[K, V]) Len() int {
	this.mu.Lock()
	var l = this.list.Len()
	this.mu.Unlock()
	return l
}

func (this *Cache[K, V]) OnEvicted(fn func(key K, value V)) {
	this.onEvicted = fn
}

func (this *Cache[K, V]) removeElement(ele *internal.Element[K, V]) {
	this.list.Remove(ele)
	delete(this.elements, ele.Key)
}

func (this *Cache[K, V]) evict(ele *internal.Element[K, V]) {
	if ele != nil && this.onEvicted != nil {
		this.onEvicted(ele.Key, ele.Value)
	}
}
