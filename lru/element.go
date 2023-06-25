package lru

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is a fork of container/list

type Element[K Key, V any] struct {
	next, prev *Element[K, V]
	list       *List[K, V]
	Key        K
	Value      V
}

func (e *Element[K, V]) Next() *Element[K, V] {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

func (e *Element[K, V]) Prev() *Element[K, V] {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}
