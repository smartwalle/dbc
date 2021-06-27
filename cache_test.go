package dbc_test

import (
	"github.com/smartwalle/dbc"
	"strconv"
	"testing"
)

func BenchmarkCache_Set(b *testing.B) {
	tc := dbc.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("test"+strconv.Itoa(i), "value")
	}
}
