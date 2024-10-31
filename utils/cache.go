package utils

import (
	"sync"
	"time"
)

type Node[T any] struct {
	key        string
	value      T
	expireAt   time.Time
	prev, next *Node[T]
}

type LRUCache[T any] struct {
	mu       sync.RWMutex
	items    map[string]*Node[T]
	head     *Node[T]
	tail     *Node[T]
	capacity int
	ttl      time.Duration
	stopChan chan struct{}
}

func NewLRUCache[T any](capacity int, ttl time.Duration) *LRUCache[T] {
	if capacity <= 0 {
		panic("capacity must be greater than zero")
	}
	cache := &LRUCache[T]{
		items:    make(map[string]*Node[T]),
		capacity: capacity,
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}
	if ttl > 0 {
		go cache.cleanupExpiredItems()
	}
	return cache
}

func (c *LRUCache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, exists := c.items[key]; exists {
		if !node.expireAt.IsZero() && node.expireAt.Before(time.Now()) {
			c.removeNode(node)
			var zero T
			return zero, false
		}
		if c.ttl > 0 {
			node.expireAt = time.Now().Add(c.ttl)
		}
		c.moveToFront(node)
		return node.value, true
	}

	var zero T
	return zero, false
}

func (c *LRUCache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expireAt := time.Time{}
	if c.ttl > 0 {
		expireAt = time.Now().Add(c.ttl)
	}

	if node, exists := c.items[key]; exists {
		node.value = value
		node.expireAt = expireAt
		c.moveToFront(node)
	} else {
		node := &Node[T]{
			key:      key,
			value:    value,
			expireAt: expireAt,
		}
		c.items[key] = node
		c.addToFront(node)

		if len(c.items) > c.capacity {
			c.removeOldest()
		}
	}
}

func (c *LRUCache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, exists := c.items[key]; exists {
		c.removeNode(node)
	}
}

func (c *LRUCache[T]) moveToFront(node *Node[T]) {
	if node == c.head {
		return
	}
	c.unlinkNode(node)
	c.addToFront(node)
}

func (c *LRUCache[T]) addToFront(node *Node[T]) {
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
	}
}

func (c *LRUCache[T]) unlinkNode(node *Node[T]) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
	node.prev = nil
	node.next = nil
}

func (c *LRUCache[T]) removeNode(node *Node[T]) {
	delete(c.items, node.key)
	c.unlinkNode(node)
}

func (c *LRUCache[T]) removeOldest() {
	if c.tail != nil {
		c.removeNode(c.tail)
	}
}

func (c *LRUCache[T]) cleanupExpiredItems() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			var expiredNodes []*Node[T]

			c.mu.RLock()
			for _, node := range c.items {
				if !node.expireAt.IsZero() && node.expireAt.Before(now) {
					expiredNodes = append(expiredNodes, node)
				}
			}
			c.mu.RUnlock()
			if len(expiredNodes) > 0 {
				c.mu.Lock()
				now = time.Now()
				for _, node := range expiredNodes {
					if node.expireAt.Before(now) {
						c.removeNode(node)
					}
				}
				c.mu.Unlock()
			}
		case <-c.stopChan:
			return
		}
	}
}

func (c *LRUCache[T]) Close() {
	select {
	case <-c.stopChan:
	default:
		close(c.stopChan)
	}
}
