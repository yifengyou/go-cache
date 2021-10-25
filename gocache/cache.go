package gocache

import (
	"github.com/yifengyou/go-cache/ciface"
	"sync"
)

var (
	// 嵌套map
	// 顶层是map[string]ciface.ICacheTable
	// 下一层是map[interface{}]ciface.ICacheItem
	cache = make(map[string]ciface.ICacheTable)
	// cache配套的读写锁
	mutex sync.RWMutex
)

// 用于创建新表或者返回已经存在的表句柄
func NewCache(table string) ciface.ICacheTable {
	// 全局变量实现的单例模式，最基本的模式
	mutex.RLock()
	t, ok := cache[table]
	mutex.RUnlock()

	if !ok {
		mutex.Lock()
		// 此次并发安全，还需要再次check一下，double check
		t, ok = cache[table]
		// Double check whether the table exists or not.
		if !ok {
			// 实例传递给接口，需要取实例地址
			// 否则会报 method has pointer receiverd
			t = &CacheTable{
				name:  table,
				items: make(map[interface{}]ciface.ICacheItem),
			}
			cache[table] = t
		}
		mutex.Unlock()
	}
	return t
}
