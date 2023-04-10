package dbc

type Element[V any] struct {
	value      V
	expiration int64
}

func (this *Element[V]) expired(now int64) bool {
	return this.expiration > 0 && now >= this.expiration
}
