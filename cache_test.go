package dbc_test

import (
	"fmt"
	"github.com/smartwalle/dbc"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	var c = dbc.NewCache()
	c.OnRemovedItem(func(key string, value interface{}) {
		fmt.Println("OnRemovedItem", key, value)
	})
	c.Set("aa", "ccc", time.Second*5)
	c.Set("ab", "ccc", time.Second*5)
	c.Set("ac", "ccc", time.Second*5)
	c.Set("ad", "ccc", time.Second*5)
	fmt.Println(c.Count())

	c.Set("bb", "ddd", time.Second*4)
	fmt.Println(c.Count())

	time.Sleep(time.Second * 6)
	fmt.Println(c.Count())

}

func BenchmarkNewCache(b *testing.B) {
	var c = dbc.NewCache()
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("%d", i), "xxxxx", time.Second*5)
	}
}
