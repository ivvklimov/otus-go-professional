package hw04lrucache

import "sync"

// Key - тип ключа для кэша.
type Key string

// Cache - интерфейс LRU-кэша.
type Cache interface {
	Set(key Key, value interface{}) bool
	Get(key Key) (interface{}, bool)
	Clear()
}

// cacheItem - внутренний элемент кэша, хранящийся в списке.
type cacheItem struct {
	key   Key
	value interface{}
}

// lruCache - реализация LRU-кэша.
type lruCache struct {
	capacity int
	queue    List
	items    map[Key]*ListItem
	mu       sync.RWMutex // RWMutex для потокобезопасности
}

// NewCache создает новый LRU-кэш заданной ёмкости.
func NewCache(capacity int) Cache {
	return &lruCache{
		capacity: capacity,
		queue:    NewList(),
		items:    make(map[Key]*ListItem, capacity),
	}
}

// Get возвращает значение по ключу и флаг его наличия в кэше.
func (c *lruCache) Get(key Key) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// MoveToFront под тем же RLock - он только читает список
	// и меняет порядок, но не удаляет элементы
	c.queue.MoveToFront(item)

	if ci, ok := item.Value.(*cacheItem); ok {
		return ci.value, true
	}

	return nil, false
}

// Set добавляет или обновляет значение по ключу.
func (c *lruCache) Set(key Key, value interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if exists {
		if ci, ok := item.Value.(*cacheItem); ok {
			ci.value = value
		}
		c.queue.MoveToFront(item)
		return true
	}

	ci := &cacheItem{key: key, value: value}
	listItem := c.queue.PushFront(ci)
	c.items[key] = listItem

	if c.queue.Len() > c.capacity {
		c.removeOldest()
	}

	return false
}

// Clear полностью очищает кэш.
func (c *lruCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queue = NewList()
	c.items = make(map[Key]*ListItem, c.capacity)
}

// removeOldest удаляет самый старый элемент из кэша.
func (c *lruCache) removeOldest() {
	last := c.queue.Back()
	if last == nil {
		return
	}

	if ci, ok := last.Value.(*cacheItem); ok {
		delete(c.items, ci.key)
	}

	c.queue.Remove(last)
}
