package nmap

import "time"

type Item struct {
	data       interface{}
	expiration int64
}

func NewItem(data interface{}, expiration int64) Item {
	return Item{
		data:       data,
		expiration: expiration,
	}
}

func (this *Item) Expired() bool {
	if this.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > this.expiration
}

func (this *Item) Data() interface{} {
	return this.data
}

func (this *Item) UpdateExpiration(expiration int64) {
	this.expiration = expiration
}
