package main

import (
	"fmt"
	"github.com/smartwalle/dbc"
	"time"
)

func main() {
	stringKey()
}

func stringKey() {
	var sCache = dbc.New[string]()
	sCache.OnEvicted(func(key string, value string) {
		fmt.Println("删除对象：", time.Now().Unix(), key, value)
	})

	sCache.Set("k1", "v1")
	sCache.Set("k2", "v2")

	sCache.SetEx("kk1", "vv1", 0)
	sCache.SetEx("kk2", "vv2", 2)
	sCache.SetEx("kk3", "vv3", 2)
	sCache.SetEx("kk4", "vv4", 2)

	fmt.Println("缓存对象数量：", sCache.Len())

	time.Sleep(time.Second * 3)

	fmt.Println(sCache.Get("k1"))
	fmt.Println(sCache.Get("k2"))
	fmt.Println(sCache.Get("k3"))

	fmt.Println(sCache.Get("kk1"))
	fmt.Println(sCache.Get("kk2"))
	fmt.Println(sCache.Get("kk3"))
	fmt.Println(sCache.Get("kk4"))

	fmt.Println("缓存对象数量：", sCache.Len())
}

func intKey() {
	var iCache = dbc.NewCache[int, string](func(key int) uint32 {
		return uint32(key % dbc.ShardCount)
	})

	iCache.Set(1, "v1")
	iCache.Set(2, "v2")

	fmt.Println(iCache.Get(1))
	fmt.Println(iCache.Get(2))
	fmt.Println(iCache.Get(3))
}
