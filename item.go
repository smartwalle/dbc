package dbc

import "time"

type Item struct {
	data       interface{}
	expiration int64
}

func (this *Item) expired() bool {
	if this.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > this.expiration
}
