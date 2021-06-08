package ecache

import (
	"sync"
	"time"
)

/**
*@Author lyer
*@Date 5/21/21 21:45
*@Describe
**/

type Value struct {
	Data       interface{}
	Expiration int64 //timestamp
}

const (
	NoExpiration           int64         = 0
	DefaultCleanUpInterval time.Duration = time.Minute
)

type Cache struct {
	rwm       sync.RWMutex
	cache     sync.Map
	OnEvicted func(key string, data interface{})
}

func (c *Cache) Set(key string, data interface{}) {
	val := Value{
		Data:       data,
		Expiration: NoExpiration,
	}
	c.cache.Store(key, val)
}
func (c *Cache) SetWithExpiration(key string, data interface{}, d time.Duration) {
	val := Value{
		Data:       data,
		Expiration: time.Now().Add(d).UnixNano(),
	}
	c.cache.Store(key, val)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.getValue(key)
	if !ok {
		return nil, false
	}
	return val.Data, true
}

func (c *Cache) Delete(key string) {
	if v, ok := c.Get(key); ok && c.OnEvicted != nil {
		val := v.(Value)
		c.OnEvicted(key, val.Data)
		c.cache.Delete(key)
	}
}

type kv struct {
	key string
	val Value
}

func (c *Cache) DeleteExpired() (count int) {
	deletedKV := []kv{}
	c.cache.Range(func(k, v interface{}) bool {
		if val := v.(Value); val.Expired() {
			key := k.(string)
			c.cache.Delete(k)
			deletedKV = append(deletedKV, kv{key, val})
		}
		return true
	})
	if c.OnEvicted != nil {
		for _, kv := range deletedKV {
			c.OnEvicted(kv.key, kv.val)
		}
	}
	count = len(deletedKV)
	return
}

func (c *Cache) getValue(key string) (Value, bool) {
	data, ok := c.cache.Load(key)
	if !ok {
		return Value{}, false
	}
	val := data.(Value)
	if val.Expired() { //expired or no expired
		return Value{}, false
	}
	return val, true
}

//func (c *Cache) TTL(key string) int64 {
//	val, ok := c.getValue(key)
//	if !ok {
//		return -1
//	}
//	return val.ttl()
//}
//
//func (val *Value) ttl() int64 {
//	if val.Expired() {
//		return 0
//	}
//	return int64(val.Expiration) - int64(time.Now().UnixNano())
//}

func (val *Value) Expired() bool {
	if val.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > val.Expiration
}

func New(cleanupInterval time.Duration) *Cache {
	cache := &Cache{
		cache:     sync.Map{},
		OnEvicted: nil,
	}
	go cache.runCleanUpJob(&cache.cache, cleanupInterval)
	return cache
}

func (c *Cache) runCleanUpJob(cache *sync.Map, interval time.Duration) {
	if interval == 0 {
		interval = DefaultCleanUpInterval
	}
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		}
	}
}
