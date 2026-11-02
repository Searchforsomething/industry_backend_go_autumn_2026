package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}

type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	listEntry, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	c.ll.MoveToBack(listEntry)
	return listEntry.Value.(entry[K, V]).value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	element := entry[K, V]{key: key, value: value}
	listEntry, ok := c.items[key]
	if ok {
		item := listEntry.Value.(entry[K, V])
		item.value = value
		listEntry.Value = item
		return
	}
	if c.ll.Len() >= c.capacity {
		elementToDelete := c.ll.Front()
		delete(c.items, elementToDelete.Value.(entry[K, V]).key)
		c.ll.Remove(elementToDelete)
	}
	listElement := c.ll.PushBack(element)
	c.items[key] = listElement
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
