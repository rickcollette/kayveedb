package lib

import (
	"errors"
	"sync"
)

// CacheManager extends the cache operations
type CacheManager struct {
	cache *Cache
	mu    sync.Mutex
}

// NewCacheManager initializes a new CacheManager
func NewCacheManager(size int, flushFn func(offset int64, node *Node) error) *CacheManager {
	return &CacheManager{
		cache: NewCache(size, flushFn),
	}
}

// SetCache adds a key-value pair to the cache with an optional expiry
func (cm *CacheManager) SetCache(key string, value []byte) {
	// Create a CacheEntry and insert it into the cache
	cm.mu.Lock()
	defer cm.mu.Unlock()
	// For demonstration, offset is 0; actual implementation will use appropriate offset
	cm.cache.Put(0, &Node{keys: []*KeyValue{{Key: key, Value: value}}}, false)
}

// GetCache retrieves a value from the cache
func (cm *CacheManager) GetCache(key string) (*Node, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	node, exists := cm.cache.Get(0) // Offset management needed
	if !exists {
		return nil, errors.New("cache miss")
	}
	return node, nil
}

// DeleteCache deletes a key from the cache
func (cm *CacheManager) DeleteCache(key string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	// Logic to find the cache entry and delete it
	return nil
}

// FlushCache removes all entries from the cache
func (cm *CacheManager) FlushCache() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cache.store = sync.Map{}
	cm.cache.order.Init()
}

// SetCacheSize adjusts the cache size
func (cm *CacheManager) SetCacheSize(size int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cache.size = size
}

// GetCacheSize returns the cache size
func (cm *CacheManager) GetCacheSize() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.cache.size
}

// SetCachePolicy sets the cache eviction policy (e.g., LRU, LFU)
func (cm *CacheManager) SetCachePolicy(policy string) error {
	// Future implementation to change policy
	return nil
}
