package merge

import (
	"testing"
)

func TestMapCache_Get(t *testing.T) {
	cache := NewCache()
	cache.Set("key1", "data1")
	cache.Set("key2", "data2")

	t.Run("Existing key", func(t *testing.T) {
		data, ok := cache.Get("key1")
		if !ok {
			t.Errorf("Get returned false for an existing key")
		}
		if data != "data1" {
			t.Errorf("Get returned %s, want data1", data)
		}
	})

	t.Run("Non-existing key", func(t *testing.T) {
		data, ok := cache.Get("key3")
		if ok {
			t.Errorf("Get returned true for a non-existing key")
		}
		if data != "" {
			t.Errorf("Get returned %s, want empty string", data)
		}
	})
}

func TestMapCache_Set(t *testing.T) {
	cache := NewCache()

	t.Run("Set key and data", func(t *testing.T) {
		cache.Set("key1", "data1")
		data, ok := cache.Get("key1")
		if !ok {
			t.Errorf("Get returned false for an existing key")
		}
		if data != "data1" {
			t.Errorf("Get returned %s, want data1", data)
		}
	})

	t.Run("Update existing key", func(t *testing.T) {
		cache.Set("key1", "updatedData")
		data, ok := cache.Get("key1")
		if !ok {
			t.Errorf("Get returned false for an existing key")
		}
		if data != "updatedData" {
			t.Errorf("Get returned %s, want updatedData", data)
		}
	})
}

func TestMapCache_Clear(t *testing.T) {
	cache := NewCache()
	cache.Set("key1", "data1")
	cache.Set("key2", "data2")

	cache.Clear()

	t.Run("Get after clear", func(t *testing.T) {
		data, ok := cache.Get("key1")
		if ok {
			t.Errorf("Get returned true after clearing the cache")
		}
		if data != "" {
			t.Errorf("Get returned %s, want empty string", data)
		}
	})
}
