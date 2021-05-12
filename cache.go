package dbc

import "time"

type Cache interface {
	Set(key string, value interface{})

	SetEx(key string, value interface{}, expiration time.Duration)

	SetNx(key string, value interface{}) bool

	Exists(key string) bool

	Expire(key string, expiration time.Duration)

	TTL(key string) time.Duration

	Get(key string) interface{}

	Del(key string)
}
