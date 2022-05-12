package nmap

type Element struct {
	Value      interface{}
	Expiration int64
}

//func NewElement(key string, value interface{}, expiration int64) *Element {
//	return &Element{
//		key:        key,
//		value:      value,
//		expiration: expiration,
//	}
//}

//func (this *Element) Init(value interface{}, expiration int64) {
//	this.value = value
//	this.expiration = expiration
//}
//
//func (this *Element) Reset() {
//	this.value = nil
//	this.expiration = 0
//}
//
//func (this *Element) Value() interface{} {
//	return this.value
//}
//
//func (this *Element) UpdateValue(value interface{}) {
//	this.value = value
//}
//
//func (this *Element) Expiration() int64 {
//	return this.expiration
//}
//
//func (this *Element) UpdateExpiration(expiration int64) {
//	this.expiration = expiration
//}
