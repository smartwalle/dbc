package lru

import (
	"github.com/smartwalle/dbc/internal"
)

type Key = internal.Key

type Cache[K Key, V any] struct {
	size      int
	list      *internal.List[K, V]
	elements  map[K]*internal.Element[K, V]
	onEvicted func(K, V)
}

func New[K Key, V any](size int) *Cache[K, V] {
	if size <= 0 {
		size = 512
	}
	var nLRU = &Cache[K, V]{}
	nLRU.size = size
	nLRU.list = internal.New[K, V]()
	nLRU.elements = make(map[K]*internal.Element[K, V], size)
	return nLRU
}

func (this *Cache[K, V]) Set(key K, value V) bool {
	var ele, found = this.elements[key]
	if found {
		ele.Value = value
		this.list.MoveToFront(ele)
	} else {
		ele = this.list.PushFront(key, value)
		this.elements[key] = ele

		if this.list.Len() > this.size {
			this.removeOldest()
		}
	}
	return true
}

func (this *Cache[K, V]) Exists(key K) bool {
	var _, found = this.elements[key]
	return found
}

func (this *Cache[K, V]) Get(key K) (value V, ok bool) {
	if ele, found := this.elements[key]; found {
		this.list.MoveToFront(ele)
		return ele.Value, true
	}
	return
}

func (this *Cache[K, V]) Del(key K) {
	if ele, found := this.elements[key]; found {
		this.removeElement(ele)
	}
}

func (this *Cache[K, V]) Len() int {
	return this.list.Len()
}

func (this *Cache[K, V]) OnEvicted(fn func(key K, value V)) {
	this.onEvicted = fn
}

func (this *Cache[K, V]) removeOldest() {
	if ele := this.list.Back(); ele != nil {
		this.removeElement(ele)
	}
}

func (this *Cache[K, V]) removeElement(ele *internal.Element[K, V]) {
	this.list.Remove(ele)
	delete(this.elements, ele.Key)

	if this.onEvicted != nil {
		this.onEvicted(ele.Key, ele.Value)
	}
}
