package lru

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key       string
	value     string
	expiresAt time.Time
}

type Cache struct {
	mu      sync.Mutex
	maxSize int
	ttl     time.Duration
	items   map[string]*list.Element
	order   *list.List
	now     func() time.Time
}

func New(maxSize int, ttl time.Duration) *Cache {
	if maxSize < 1 {
		maxSize = 1
	}

	return &Cache{
		maxSize: maxSize,
		ttl:     ttl,
		items:   make(map[string]*list.Element),
		order:   list.New(),
		now:     time.Now,
	}
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]

	if !ok {
		return "", false
	}

	cached := element.Value.(entry)
	if !cached.expiresAt.IsZero() && c.now().After(cached.expiresAt) {
		delete(c.items, key)
		c.order.Remove(element)
		return "", false
	}
	c.order.MoveToFront(element)
	return cached.value, true
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if c.ttl > 0 {
		expiresAt = c.now().Add(c.ttl)
	}
	if element, ok := c.items[key]; ok {
		element.Value = entry{key: key, value: value, expiresAt: expiresAt}
		c.order.MoveToFront(element)
		return
	}

	element := c.order.PushFront(entry{key: key, value: value, expiresAt: expiresAt})
	c.items[key] = element

	if c.order.Len() > c.maxSize {
		oldest := c.order.Back()
		oldestEntry := oldest.Value.(entry)

		delete(c.items, oldestEntry.key)
		c.order.Remove(oldest)
	}

}
