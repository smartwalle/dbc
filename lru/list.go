package lru

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is a fork of container/list

type List[K Key, V any] struct {
	root Element[K, V]
	len  int
}

func New[K Key, V any]() *List[K, V] {
	var n = &List[K, V]{}
	n.Init()
	return n
}

func (l *List[K, V]) Init() *List[K, V] {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func (l *List[K, V]) Len() int {
	return l.len
}

func (l *List[K, V]) Front() *Element[K, V] {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

func (l *List[K, V]) Back() *Element[K, V] {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

func (l *List[K, V]) lazyInit() {
	if l.root.next == nil {
		l.Init()
	}
}

func (l *List[K, V]) insert(e, at *Element[K, V]) *Element[K, V] {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.list = l
	l.len++
	return e
}

func (l *List[K, V]) insertValue(k K, v V, at *Element[K, V]) *Element[K, V] {
	return l.insert(&Element[K, V]{Key: k, Value: v}, at)
}

func (l *List[K, V]) remove(e *Element[K, V]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
}

func (l *List[K, V]) move(e, at *Element[K, V]) {
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
}

func (l *List[K, V]) Remove(e *Element[K, V]) V {
	if e.list == l {
		l.remove(e)
	}
	return e.Value
}

func (l *List[K, V]) PushFront(k K, v V) *Element[K, V] {
	l.lazyInit()
	return l.insertValue(k, v, &l.root)
}

func (l *List[K, V]) PushBack(k K, v V) *Element[K, V] {
	l.lazyInit()
	return l.insertValue(k, v, l.root.prev)
}

func (l *List[K, V]) InsertBefore(k K, v V, mark *Element[K, V]) *Element[K, V] {
	if mark.list != l {
		return nil
	}
	return l.insertValue(k, v, mark.prev)
}

func (l *List[K, V]) InsertAfter(k K, v V, mark *Element[K, V]) *Element[K, V] {
	if mark.list != l {
		return nil
	}
	return l.insertValue(k, v, mark)
}

func (l *List[K, V]) MoveToFront(e *Element[K, V]) {
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, &l.root)
}

func (l *List[K, V]) MoveToBack(e *Element[K, V]) {
	if e.list != l || l.root.prev == e {
		return
	}
	l.move(e, l.root.prev)
}

func (l *List[K, V]) MoveBefore(e, mark *Element[K, V]) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.prev)
}

func (l *List[K, V]) MoveAfter(e, mark *Element[K, V]) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark)
}
