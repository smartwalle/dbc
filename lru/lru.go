package lru

type Key interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type LRU[K Key, V any] struct {
	size      int
	lruList   *List[K, V]
	elements  map[K]*Element[K, V]
	onEvicted func(K, V)
}

func NewLRU[K Key, V any](size int) *LRU[K, V] {
	if size <= 0 {
		size = 512
	}
	var nLRU = &LRU[K, V]{}
	nLRU.size = size
	nLRU.lruList = New[K, V]()
	nLRU.elements = make(map[K]*Element[K, V], size)
	return nLRU
}

func (this *LRU[K, V]) Set(key K, value V) bool {
	var ele, found = this.elements[key]
	if found {
		ele.Value = value
		this.lruList.MoveToFront(ele)
	} else {
		ele = this.lruList.PushFront(key, value)
		this.elements[key] = ele

		if this.lruList.Len() > this.size {
			this.removeOldest()
		}
	}
	return true
}

func (this *LRU[K, V]) Exists(key K) bool {
	var _, found = this.elements[key]
	return found
}

func (this *LRU[K, V]) Get(key K) (value V, ok bool) {
	if ele, found := this.elements[key]; found {
		this.lruList.MoveToFront(ele)
		return ele.Value, true
	}
	return
}

func (this *LRU[K, V]) Del(key K) {
	if ele, found := this.elements[key]; found {
		this.removeElement(ele)
	}
}

func (this *LRU[K, V]) Len() int {
	return this.lruList.len
}

func (this *LRU[K, V]) OnEvicted(fn func(key K, value V)) {
	this.onEvicted = fn
}

func (this *LRU[K, V]) removeOldest() {
	if ele := this.lruList.Back(); ele != nil {
		this.removeElement(ele)
	}
}

func (this *LRU[K, V]) removeElement(ele *Element[K, V]) {
	this.lruList.Remove(ele)
	delete(this.elements, ele.Key)

	if this.onEvicted != nil {
		this.onEvicted(ele.Key, ele.Value)
	}
}
