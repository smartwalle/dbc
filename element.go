package dbc

type Element[T any] struct {
	value      T
	expiration int64
}
