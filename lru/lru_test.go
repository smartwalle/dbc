package lru_test

import (
	"github.com/smartwalle/dbc/lru"
	"testing"
)

func BenchmarkLRU_SetIntString(b *testing.B) {
	var m = lru.New[int, string](lru.WithMaxSize(1000000), lru.WithInitSize(1000000))
	m.OnEvicted(func(key int, value string) {
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(i, "hello")
	}
}

func BenchmarkLRU_GetIntString(b *testing.B) {
	var m = lru.New[int, string](lru.WithMaxSize(1000000), lru.WithInitSize(1000000))
	m.OnEvicted(func(key int, value string) {
	})
	for i := 0; i < b.N; i++ {
		m.Set(i, "hello")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(i)
	}
}
