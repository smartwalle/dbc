package nmap

type Item struct {
	key        string
	value      interface{}
	expiration int64
}

func NewItem(key string, value interface{}, expiration int64) *Item {
	return &Item{
		key:        key,
		value:      value,
		expiration: expiration,
	}
}

func (this *Item) Key() string {
	return this.key
}

func (this *Item) Value() interface{} {
	return this.value
}

func (this *Item) UpdateValue(value interface{}) {
	this.value = value
}

func (this *Item) Expiration() int64 {
	return this.expiration
}

func (this *Item) UpdateExpiration(expiration int64) {
	this.expiration = expiration
}

func (this *Item) Extend(t int64) int64 {
	if this.expiration == 0 {
		return 0
	}
	this.expiration += t
	return this.expiration
}
