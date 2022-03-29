package dbc_test

import (
	"github.com/smartwalle/dbc"
	"strconv"
	"testing"
)

func set(c dbc.Cache[string], b *testing.B) {
	for i := 0; i < b.N; i++ {
		c.Set("sss"+strconv.Itoa(i), "hello")
	}
}

func get(c dbc.Cache[string], b *testing.B) {
	for i := 0; i < b.N; i++ {
		c.Get("sss" + strconv.Itoa(i))
	}
}

func BenchmarkCache_Set(b *testing.B) {
	c := dbc.New[string]()
	b.ResetTimer()
	set(c, b)
}

func BenchmarkCache_Get(b *testing.B) {
	c := dbc.New[string]()
	set(c, b)
	b.ResetTimer()
	get(c, b)
}
