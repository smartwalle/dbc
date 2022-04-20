package main

import (
	"fmt"
	"github.com/smartwalle/dbc"
	"time"
)

func main() {
	var cache = dbc.New(dbc.WithHitTTL(10))
	cache.OnEvicted(func(key string, value interface{}) {
		fmt.Println("Del", key, value)
	})

	cache.Set("k1", "v1")
	cache.Set("k2", "v2")

	cache.SetEx("kk1", "vv1", time.Now().Add(time.Second*2).Unix())
	cache.SetEx("kk2", "vv2", time.Now().Add(time.Second*2).Unix())
	//
	//fmt.Println(cache.Get("k1"))
	//fmt.Println(cache.Get("k2"))
	//fmt.Println(cache.Get("k3"))

	for {
		fmt.Println(cache.Get("kk1"))
		time.Sleep(time.Second)
	}

	select {}
}
