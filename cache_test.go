package dbc_test

import (
	"github.com/smartwalle/dbc"
	"strconv"
	"sync"
	"testing"
	"time"
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

func BenchmarkCache_Set2(b *testing.B) {
	c := dbc.New[string]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("test", "test")
	}
}

func BenchmarkCache_Get2(b *testing.B) {
	c := dbc.New[string]()
	c.Set("test", "test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("test")
	}
}

func BenchmarkCache_Get3(b *testing.B) {
	c := dbc.New[string](dbc.WithHitTTL(10))
	c.SetEx("test", "test", 4)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("test")
	}
}

func BenchmarkCache_Close(b *testing.B) {
	c := dbc.New[string]()

	var w = &sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		c.Set("sss"+strconv.Itoa(i), "hello")
		w.Add(1)
	}

	b.ResetTimer()

	c.OnEvicted(func(key string, value string) {
		w.Done()
	})

	c.Close()

	w.Wait()
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
