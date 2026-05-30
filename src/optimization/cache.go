package optimization

import (
	"crypto/md5"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type CacheEntry struct {
	Content    string
	ModTime    time.Time
	FileSize   int64
	Hash       string
}

type Cache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	hits    atomic.Int64
	misses  atomic.Int64
}

func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
	}
}

func (c *Cache) Get(path string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.entries[path]
	if !exists {
		c.misses.Add(1)
		return "", false
	}
	
	info, err := os.Stat(path)
	if err != nil {
		c.misses.Add(1)
		return "", false
	}
	
	if info.ModTime().After(entry.ModTime) || info.Size() != entry.FileSize {
		c.misses.Add(1)
		return "", false
	}
	
	c.hits.Add(1)
	return entry.Content, true
}

func (c *Cache) Set(path string, content string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	
	hash := c.computeHash(content)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries[path] = &CacheEntry{
		Content:  content,
		ModTime:  info.ModTime(),
		FileSize: info.Size(),
		Hash:     hash,
	}
}

func (c *Cache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.entries, path)
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries = make(map[string]*CacheEntry)
	c.hits.Store(0)
	c.misses.Store(0)
}

func (c *Cache) Stats() (hits, misses int, hitRate float64) {
	hits = int(c.hits.Load())
	misses = int(c.misses.Load())
	total := hits + misses
	
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	
	return
}

func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.entries)
}

func (c *Cache) computeHash(content string) string {
	h := md5.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (c *Cache) HasChanged(path string, content string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.entries[path]
	if !exists {
		return true
	}
	
	newHash := c.computeHash(content)
	return entry.Hash != newHash
}