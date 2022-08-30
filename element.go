package dbc

type Element[T any] struct {
	value      T
	expiration int64
}

func (this *Element[T]) expired(now int64) bool {
	return this.expiration > 0 && now >= this.expiration
}
