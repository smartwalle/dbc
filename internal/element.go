package internal

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is a fork of container/list

type Key interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

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
