package dbc

type Element[T any] struct {
	expiration int64
	value      T
}

func (this *Element[T]) expired(now int64) bool {
	return this.expiration > 0 && now >= this.expiration
}
