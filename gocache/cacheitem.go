package gocache

import (
	"github.com/yifengyou/go-cache/ciface"
	"sync"
	"time"
)

type CacheItem struct {
	sync.RWMutex

	// 多元键
	key interface{}
	// 多元数据
	data interface{}
	// 生命周期，谁来处理销毁？
	lifeSpan time.Duration

	// 创建时间
	createdOn time.Time
	// 最近访问时间
	accessedOn time.Time
	// 访问次数，统计热点cache
	accessCount int64

	// 移除cache项时执行
	aboutToExpire []func(key interface{})
}

// 实例化一个cache，key=data
// 生存周期为lifeSpan，如果是0则表示永久存在
func NewCacheItem(key interface{}, lifeSpan time.Duration, data interface{}) ciface.ICacheItem {
	t := time.Now()
	return &CacheItem{
		key:           key,
		lifeSpan:      lifeSpan,
		createdOn:     t,
		accessedOn:    t,
		accessCount:   0,
		aboutToExpire: nil,
		data:          data,
	}
}

// 增加访问次数，更新访问时间
func (item *CacheItem) KeepAlive() {
	item.Lock()
	defer item.Unlock()
	item.accessedOn = time.Now()
	item.accessCount++
}

// 获取item生存时间。生存时间是不可改变的，因此无需上锁
func (item *CacheItem) LifeSpan() time.Duration {
	return item.lifeSpan
}

// 获取item访问时间，有并发问题，需上锁
func (item *CacheItem) AccessedOn() time.Time {
	item.RLock()
	defer item.RUnlock()
	return item.accessedOn
}

// 获取item创建时间，不可变，无需上锁
func (item *CacheItem) CreatedOn() time.Time {
	return item.createdOn
}

// AccessCount returns how often this item has been accessed.
func (item *CacheItem) AccessCount() int64 {
	item.RLock()
	defer item.RUnlock()
	return item.accessCount
}

// 获取item的key，不可变，无需加锁
func (item *CacheItem) Key() interface{} {
	return item.key
}

// 获取item的key，value可变，须加锁
func (item *CacheItem) Data() interface{} {
	item.Lock()
	defer item.Unlock()
	return item.data
}

// 重置销毁回调
func (item *CacheItem) SetAboutToExpireCallback(f func(interface{})) {
	if len(item.aboutToExpire) > 0 {
		item.RemoveAboutToExpireCallback()
	}
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = append(item.aboutToExpire, f)
}

// 添加销毁回调
func (item *CacheItem) AddAboutToExpireCallback(f func(interface{})) {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = append(item.aboutToExpire, f)
}

// 清空所有销毁回调
func (item *CacheItem) RemoveAboutToExpireCallback() {
	item.Lock()
	defer item.Unlock()
	item.aboutToExpire = nil
}
