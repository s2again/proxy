package main

import (
	"container/list"
	"log"
	"net/http"
	"sync"
	"time"
)

type CacheBlock struct {
	Key        string
	Value      []byte
	Size       int
	Header     http.Header // 头部信息
	Expiration time.Time   // 过期时间
}

type LRUCache struct {
	cacheList    *list.List
	cacheMap     map[string]*list.Element
	cacheSize    int
	maxCacheSize int
	mutex        sync.Mutex
}

func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		cacheList:    list.New(),
		cacheMap:     make(map[string]*list.Element),
		cacheSize:    0,
		maxCacheSize: maxSize,
	}
}

func (c *LRUCache) Get(key string) ([]byte, http.Header, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Println("LRUCache detect", key)
	if ele, ok := c.cacheMap[key]; ok {
		// 检查是否已过期
		if time.Now().After(ele.Value.(*CacheBlock).Expiration) {
			c.removeElement(ele)
			log.Println("LRUCache expired", key)
			return nil, nil, false
		}

		c.cacheList.MoveToFront(ele)
		log.Println("LRUCache hit", key)
		return ele.Value.(*CacheBlock).Value, ele.Value.(*CacheBlock).Header, true
	}

	log.Println("LRUCache miss", key)
	return nil, nil, false
}

func (c *LRUCache) Put(key string, value []byte, header http.Header, size int, expiration time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Println("LRUCache Put", key)
	if ele, ok := c.cacheMap[key]; ok {
		c.cacheList.MoveToFront(ele)

		ele.Value.(*CacheBlock).Value = value
		ele.Value.(*CacheBlock).Header = header
		ele.Value.(*CacheBlock).Size = size
		ele.Value.(*CacheBlock).Expiration = expiration

		c.cacheSize += size - ele.Value.(*CacheBlock).Size
		ele.Value.(*CacheBlock).Size = size
	} else {
		newBlock := &CacheBlock{
			Key:        key,
			Value:      value,
			Header:     header,
			Size:       size,
			Expiration: expiration,
		}
		newElement := c.cacheList.PushFront(newBlock)
		c.cacheMap[key] = newElement
		c.cacheSize += size

		if c.cacheSize > c.maxCacheSize {
			c.evictOldest()
		}
	}
}

func (c *LRUCache) removeElement(ele *list.Element) {
	c.cacheList.Remove(ele)
	delete(c.cacheMap, ele.Value.(*CacheBlock).Key)
	c.cacheSize -= ele.Value.(*CacheBlock).Size
}

func (c *LRUCache) evictOldest() {
	oldestElement := c.cacheList.Back()
	if oldestElement != nil {
		c.removeElement(oldestElement)
	}
}
