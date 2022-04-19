package nmap

import (
	"time"
)

type Item struct {
	value      interface{}
	expiration int64
}

func NewItem(value interface{}, expiration int64) *Item {
	return &Item{
		value:      value,
		expiration: expiration,
	}
}

func (this *Item) Expired() bool {
	if this.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > this.expiration
}

func (this *Item) Value() interface{} {
	return this.value
}

func (this *Item) UpdateExpiration(expiration int64) {
	this.expiration = expiration
}

func (this *Item) Extend(t int64) {
	if this.expiration == 0 {
		return
	}
	this.expiration += t
}
