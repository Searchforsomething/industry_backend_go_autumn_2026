package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	cache := Cache[K, V]{capacity: capacity, items: make(map[K]V)}
	return &cache
}

func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	value, ok := c.items[k]
	return value, ok
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	_, ok := c.items[k]
	if ok {
		c.items[k] = v
		return true
	}
	if len(c.items) >= c.capacity {
		return false
	}
	c.items[k] = v
	return true
}
