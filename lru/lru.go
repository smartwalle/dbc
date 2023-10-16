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

func (c *Cache[K, V]) Set(key K, value V) bool {
	c.mu.Lock()
	var ele, found = c.elements[key]
	if found {
		ele.Value = value
		c.list.MoveToFront(ele)
		c.mu.Unlock()
		return true
	}

	ele = c.list.PushFront(key, value)
	c.elements[key] = ele
	var needEvict = c.list.Len() > c.size

	if needEvict {
		var last = c.list.Back()
		c.removeElement(last)
		c.mu.Unlock()
		c.evict(last)
		return true
	}
	c.mu.Unlock()
	return true
}

func (c *Cache[K, V]) Exists(key K) bool {
	c.mu.Lock()
	var _, found = c.elements[key]
	c.mu.Unlock()
	return found
}

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	if ele, found := c.elements[key]; found {
		c.list.MoveToFront(ele)
		c.mu.Unlock()
		return ele.Value, true
	}
	c.mu.Unlock()
	return
}

func (c *Cache[K, V]) Del(key K) {
	c.mu.Lock()
	if ele, found := c.elements[key]; found {
		c.removeElement(ele)
		c.mu.Unlock()
		c.evict(ele)
		return
	}
	c.mu.Unlock()
}

func (c *Cache[K, V]) Len() int {
	c.mu.Lock()
	var l = c.list.Len()
	c.mu.Unlock()
	return l
}

func (c *Cache[K, V]) OnEvicted(fn func(key K, value V)) {
	c.onEvicted = fn
}

func (c *Cache[K, V]) removeElement(ele *internal.Element[K, V]) {
	c.list.Remove(ele)
	delete(c.elements, ele.Key)
}

func (c *Cache[K, V]) evict(ele *internal.Element[K, V]) {
	if ele != nil && c.onEvicted != nil {
		c.onEvicted(ele.Key, ele.Value)
	}
}
