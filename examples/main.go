package main

import (
	"fmt"
	"github.com/smartwalle/dbc"
	"time"
)

func main() {
	var cache = dbc.New[string]()
	cache.OnEvicted(func(key string, value string) {
		//fmt.Println("Del", time.Now().Unix(), key, value)
	})

	cache.Set("k1", "v1")
	cache.Set("k2", "v2")

	cache.SetEx("kk1", "vv1", 0)
	cache.SetEx("kk2", "vv2", 2)
	cache.SetEx("kk3", "vv3", 2)
	cache.SetEx("kk4", "vv4", 2)
	//
	//fmt.Println(cache.Get("k1"))
	//fmt.Println(cache.Get("k2"))
	//fmt.Println(cache.Get("k3"))

	go func() {
		var i = 0
		for {
			cache.SetEx(fmt.Sprintf("%d", i), "testsss", 20)
			i++
		}
	}()

	var i = 0
	for {
		fmt.Println(i)
		cache.Expire("kk1", 5)
		time.Sleep(time.Second)
		i++
	}

	select {}
}
