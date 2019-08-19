package dbc_test

import (
	"fmt"
	"github.com/smartwalle/dbc"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	var c = dbc.NewCache(time.Second * 5)
	c.Set("aa", "ccc")
	fmt.Println(c.Count())

	c.Set("bb", "ddd")
	fmt.Println(c.Count())

	time.Sleep(time.Second * 6)
	fmt.Println(c.Count())

}

func BenchmarkNewCache(b *testing.B) {
	var c = dbc.NewCache(time.Second * 1)
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("%d", i), "xxxxx")
	}
}
