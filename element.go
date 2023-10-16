package dbc

type Element[V any] struct {
	value      V
	expiration int64
}

func (e Element[V]) expired(now int64) bool {
	return e.expiration > 0 && now >= e.expiration
}
