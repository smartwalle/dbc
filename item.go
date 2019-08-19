package dbc

import (
	"sync"
	"time"
)

func newItem(value interface{}, ttl time.Duration) *item {
	var i = &item{}
	i.value = value
	i.touch(ttl)
	return i
}

type item struct {
	sync.RWMutex
	value   interface{}
	expires *time.Time
}

func (this *item) touch(ttl time.Duration) {
	this.Lock()
	defer this.Unlock()

	var expires = time.Now().Add(ttl)
	this.expires = &expires
}

func (this *item) expired() bool {
	this.RLock()
	defer this.RUnlock()

	if this.expires == nil {
		return true
	}

	if this.expires.Before(time.Now()) {
		return true
	}
	return false
}
