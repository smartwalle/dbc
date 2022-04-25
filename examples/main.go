package main

import (
	"fmt"
	"github.com/smartwalle/dbc"
	"time"
)

func main() {
	var cache = dbc.New()
	cache.OnEvicted(func(key string, value interface{}) {
		fmt.Println("Del", time.Now().Unix(), key, value)
	})

	cache.Set("k1", "v1")
	cache.Set("k2", "v2")

	cache.SetEx("kk1", "vv1", 0)
	cache.SetEx("kk2", "vv2", time.Now().Add(time.Second*2).Unix())
	cache.SetEx("kk3", "vv3", time.Now().Add(time.Second*2).Unix())
	cache.SetEx("kk4", "vv4", time.Now().Add(time.Second*2).Unix())
	//
	//fmt.Println(cache.Get("k1"))
	//fmt.Println(cache.Get("k2"))
	//fmt.Println(cache.Get("k3"))

	for {
		fmt.Println(cache.Get("kk1"))
		cache.Expire("kk1", time.Now().Add(time.Second*5).Unix())
		time.Sleep(time.Second)
	}

	select {}
}
