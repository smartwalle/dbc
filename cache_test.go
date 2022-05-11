package dbc_test

import (
	"github.com/smartwalle/dbc"
	"strconv"
	"sync"
	"testing"
	"time"
)

func set(c dbc.Cache, b *testing.B) {
	for i := 0; i < b.N; i++ {
		c.Set("sss"+strconv.Itoa(i), "hello")
	}
}

func get(c dbc.Cache, b *testing.B) {
	for i := 0; i < b.N; i++ {
		c.Get("sss" + strconv.Itoa(i))
	}
}

func BenchmarkCache_Set(b *testing.B) {
	c := dbc.New()
	b.ResetTimer()
	set(c, b)
}

func BenchmarkCache_Get(b *testing.B) {
	c := dbc.New()
	set(c, b)
	b.ResetTimer()
	get(c, b)
}

func BenchmarkCache_Close(b *testing.B) {
	c := dbc.New()

	var w = &sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		c.Set("sss"+strconv.Itoa(i), "hello")
		w.Add(1)
	}

	b.ResetTimer()

	c.OnEvicted(func(key string, value interface{}) {
		w.Done()
	})

	c.Close()

	w.Wait()
}

func TestCache_SetEx(t *testing.T) {
	c := dbc.New()
	c.OnEvicted(func(key string, value interface{}) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 2)

	time.Sleep(time.Second * 3)

	if _, ok := c.Get("k1"); ok {
		t.Fatal("k1 应该不存在")
	}
}

func TestCache_SetEx2(t *testing.T) {
	c := dbc.New()
	c.OnEvicted(func(key string, value interface{}) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 3)

	time.Sleep(time.Second * 2)

	if v, _ := c.Get("k1"); v != "v1" {
		t.Fatal("k1 的值应该是 v1")
	}
}

func TestCache_SetEx3(t *testing.T) {
	c := dbc.New()
	c.OnEvicted(func(key string, value interface{}) {
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
	c := dbc.New()
	c.OnEvicted(func(key string, value interface{}) {
		t.Log("OnEvicted", time.Now().Unix(), key, value)
	})

	c.SetEx("k1", "v1", 2)
	c.Set("k1", "v2")

	time.Sleep(time.Second * 3)
	if v, _ := c.Get("k1"); v != "v2" {
		t.Fatal("k1 的值应该是 v2")
	}
}

func TestCache_Expire(t *testing.T) {
	c := dbc.New()
	c.OnEvicted(func(key string, value interface{}) {
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
