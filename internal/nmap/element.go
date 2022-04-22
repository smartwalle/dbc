package nmap

type Element struct {
	key        string
	value      interface{}
	expiration int64
}

func NewElement(key string, value interface{}, expiration int64) *Element {
	return &Element{
		key:        key,
		value:      value,
		expiration: expiration,
	}
}

func (this *Element) Key() string {
	return this.key
}

func (this *Element) Value() interface{} {
	return this.value
}

func (this *Element) UpdateValue(value interface{}) {
	this.value = value
}

func (this *Element) Expiration() int64 {
	return this.expiration
}

func (this *Element) UpdateExpiration(expiration int64) {
	this.expiration = expiration
}
