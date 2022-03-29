package nmap

import (
	"time"
)

type Item[T any] struct {
	value      T
	expiration int64
}

func NewItem[T any](value T, expiration int64) Item[T] {
	return Item[T]{
		value:      value,
		expiration: expiration,
	}
}

func (this *Item[T]) Expired() bool {
	if this.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > this.expiration
}

func (this *Item[T]) Value() T {
	return this.value
}

func (this *Item[T]) UpdateExpiration(expiration int64) {
	this.expiration = expiration
}

func (this *Item[T]) Extend(t int64) {
	if this.expiration == 0 {
		return
	}
	this.expiration += t
}
