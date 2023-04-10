package dbc_test

import (
	"fmt"
	"github.com/smartwalle/dbc"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func BenchmarkCache_SetStringString(b *testing.B) {
	var m = dbc.New[string]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(strconv.Itoa(i), "hello")
	}
}

func BenchmarkCache_SetIntString(b *testing.B) {
	var m = dbc.NewCache[int, string](func(key int) uint32 {
		return uint32(key % dbc.ShardCount)
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(i, "hello")
	}
}

func BenchmarkCache_GetStringString(b *testing.B) {
	var m = dbc.New[string]()
	for i := 0; i < b.N; i++ {
		m.Set(strconv.Itoa(i), "hello")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(strconv.Itoa(i))
	}
}

func TestCache_SetEx(t *testing.T) {
	c := dbc.New[string]()
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 2)

	time.Sleep(time.Second * 3)

	if _, ok := c.Get("k1"); ok {
		t.Fatal("k1 应该不存在")
	}
}

func TestCache_SetEx2(t *testing.T) {
	c := dbc.New[string]()
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 3)

	time.Sleep(time.Second * 2)

	if v, _ := c.Get("k1"); v != "v1" {
		t.Fatal("k1 的值应该是 v1")
	}
}

func TestCache_SetEx3(t *testing.T) {
	c := dbc.New[string]()
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 2)
	c.Del("k1")
	c.Set("k1", "v2")

	time.Sleep(time.Second * 3)
	if v, _ := c.Get("k1"); v != "v2" {
		t.Fatal("k1 的值应该是 v2")
	}
}

func TestCache_SetEx4(t *testing.T) {
	c := dbc.New[string]()
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 2)
	c.Set("k1", "v2")

	time.Sleep(time.Second * 3)
	if v, _ := c.Get("k1"); v != "v2" {
		t.Fatal("k1 的值应该是 v2")
	}
}

func TestCache_SetEx5(t *testing.T) {
	c := dbc.New[string]()
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 2)
	time.Sleep(time.Second * 2)
	c.SetEx("k1", "v2", 2)
}

func TestCache_HitTTL(t *testing.T) {
	c := dbc.New[string](dbc.WithHitTTL(5))
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	// 3 秒后过期
	c.SetEx("k1", "v1", 3)

	// 获取到数据，过期时间延长 5 秒，大概就是 8 秒后过期
	c.Get("k1")

	time.Sleep(time.Second * 7)

	// 过期时间还剩 1 秒，所以可以获取到数据，并且过期时间延长 5 秒，大概就是 6 秒后过期
	if v, _ := c.Get("k1"); v != "v1" {
		t.Fatal("k1 的值应该是 v1")
	}

	time.Sleep(time.Second * 7)

	if _, ok := c.Get("k1"); ok {
		t.Fatal("k1 应该不存在")
	}
}

func TestCache_Expire(t *testing.T) {
	c := dbc.New[string]()
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.Set("k1", "v1")
	time.Sleep(time.Second * 1)
	c.Expire("k1", 2)
	time.Sleep(time.Second * 3)
	if _, ok := c.Get("k1"); ok {
		t.Fatal("k1 应该不存在")
	}
}

func TestCache_Del(t *testing.T) {
	c := dbc.New[string]()
	c.OnEvicted(func(key string, value string) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
		c.Set("k2", "v2")
	})
	c.Set("k1", "v1")
	c.Del("k1")

	if v, _ := c.Get("k2"); v != "v2" {
		t.Fatal("k2 的值应该是 v2")
	}
}

func TestCache_SetNx(t *testing.T) {
	c := dbc.New[string]()
	if c.SetNx("k1", "v1") == false {
		t.Fatal("应该返回 true")
	}

	if c.SetNx("k1", "v2") {
		t.Fatal("应该返回 false")
	}

	if v, _ := c.Get("k1"); v != "v1" {
		t.Fatal("k1 的值应该是 v1")
	}
}

const N = 30000000

func TimeGC() time.Duration {
	start := time.Now()
	runtime.GC()
	return time.Since(start)
}

func TestCache_GC_SV(t *testing.T) {
	var m = dbc.New[int32]()
	for i := 0; i < N; i++ {
		n := int32(i)
		m.Set(fmt.Sprintf("%d", n), n)
	}
	runtime.GC()
	t.Logf("With %T, GC took %s\n", m, TimeGC())
	_, _ = m.Get("0")
}

func TestMap_GC_IV(t *testing.T) {
	var m = dbc.NewCache[int32, int32](func(key int32) uint32 {
		return uint32(key % dbc.ShardCount)
	})
	for i := 0; i < N; i++ {
		n := int32(i)
		m.Set(n, n)
	}
	runtime.GC()
	t.Logf("With %T, GC took %s\n", m, TimeGC())
	_, _ = m.Get(0)
}
