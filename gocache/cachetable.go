package gocache

import (
	"fmt"
	"github.com/yifengyou/go-cache/ciface"
	"log"
	"sort"
	"sync"
	"time"
)

type CacheTable struct {
	sync.RWMutex // 读写锁

	// 表名称,关键元素
	name string

	// 表项,关键元素
	items map[interface{}]ciface.ICacheItem

	// 定时清理
	cleanupTimer *time.Timer

	// 当前表下一次清理的时间间隔
	cleanupInterval time.Duration

	// 表日志句柄
	logger *log.Logger

	// 访问不存在表项时指定的回调函数
	loadData func(key interface{}, args ...interface{}) ciface.ICacheItem

	// 增加cache表项时执行的回调函数
	addedItem []func(item ciface.ICacheItem)

	// 删除cache表项时执行的回调函数
	aboutToDeleteItem []func(item ciface.ICacheItem)
}

// 统计表项，并发安全，需要加锁
func (table *CacheTable) Count() int {
	table.RLock()
	defer table.RUnlock()
	return len(table.items)
}

// 遍历所有表项，并发安全，需要加锁
func (table *CacheTable) Foreach(trans func(key interface{}, item ciface.ICacheItem)) {
	table.RLock()
	defer table.RUnlock()

	for k, v := range table.items {
		trans(k, v)
	}
}

// 当视图访问不存在的表项时触发回调函数，而该函数配置对应func
func (table *CacheTable) SetDataLoader(f func(interface{}, ...interface{}) ciface.ICacheItem) {
	// 重置DataLoader
	table.Lock()
	defer table.Unlock()
	table.loadData = f
}

// 重置条件表项的回调函数
func (table *CacheTable) SetAddedItemCallback(f func(ciface.ICacheItem)) {
	// 重置AddedItemCallback
	if len(table.addedItem) > 0 {
		table.RemoveAddedItemCallbacks()
	}
	table.Lock()
	defer table.Unlock()
	fmt.Println("run here")
	table.addedItem = append(table.addedItem, f)
}

// 添加表项回调函数
func (table *CacheTable) AddAddedItemCallback(f func(ciface.ICacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.addedItem = append(table.addedItem, f)
}

// 清空添加表项的回调函数
func (table *CacheTable) RemoveAddedItemCallbacks() {
	// 直接清空，哪叫Remove呀
	table.Lock()
	defer table.Unlock()
	table.addedItem = nil
}

// 清理表项触发的回调函数
func (table *CacheTable) SetAboutToDeleteItemCallback(f func(ciface.ICacheItem)) {
	if len(table.aboutToDeleteItem) > 0 {
		table.RemoveAboutToDeleteItemCallback()
	}
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = append(table.aboutToDeleteItem, f)
}

// 添加清理表项会触发的回调函数
func (table *CacheTable) AddAboutToDeleteItemCallback(f func(ciface.ICacheItem)) {
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = append(table.aboutToDeleteItem, f)
}

// 清空清理表项时触发的回调函数
func (table *CacheTable) RemoveAboutToDeleteItemCallback() {
	table.Lock()
	defer table.Unlock()
	table.aboutToDeleteItem = nil
}

// 设置表日志句柄
func (table *CacheTable) SetLogger(logger *log.Logger) {
	table.Lock()
	defer table.Unlock()
	table.logger = logger
}

// 自适应调整链路定时器
func (table *CacheTable) expirationCheck() {
	table.Lock()
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
	if table.cleanupInterval > 0 {
		table.log("Expiration check triggered after", table.cleanupInterval, "for table", table.name)
	} else {
		table.log("Expiration check installed for table", table.name)
	}

	now := time.Now()
	smallestDuration := 0 * time.Second

	// 遍历表项
	for key, item := range table.items {
		// Cache values so we don't keep blocking the mutex.
		itemReal := (item).(*CacheItem)
		itemReal.RLock()
		// 互斥获取item项的生存时间以及上次访问时间
		lifeSpan := itemReal.lifeSpan
		accessedOn := itemReal.accessedOn
		itemReal.RUnlock()

		if lifeSpan == 0 {
			// lifeSpan 为0表示永久存在
			continue
		}
		// 生存到期
		if now.Sub(accessedOn) >= lifeSpan {
			_, err := table.deleteInternal(key)
			if err == nil {
				fmt.Println("delete ok!")
			} else {
				fmt.Println("delete failed!")
			}
		} else {
			// 获取到所有项最小时间间隔
			if smallestDuration == 0 || lifeSpan-now.Sub(accessedOn) < smallestDuration {
				smallestDuration = lifeSpan - now.Sub(accessedOn)
			}
		}
	}

	// 下次清理的最小时间间隔
	table.cleanupInterval = smallestDuration
	if smallestDuration > 0 {
		// 最小时间间隔执行函数，清理时间
		table.cleanupTimer = time.AfterFunc(smallestDuration, func() {
			// 关键，海量键值超时，则会遍历最短时间，每次增加一个键都会遍历所有值获取最短时间间隔
			// 每次执行，其实也是找到下一个最短时间间隔进行执行
			go table.expirationCheck()
		})
	}
	table.Unlock()
}

func (table *CacheTable) addInternal(item ciface.ICacheItem) {
	// 添加表项，此函数千万不可直接执行，有并发问题
	itemReal, ok := (item).(*CacheItem)
	if !ok {
		log.Fatalln("error!")
	}
	table.log("Adding item with key", itemReal.key, "and lifespan of", itemReal.lifeSpan, "to table", table.name)
	table.items[itemReal.key] = item

	expDur := table.cleanupInterval
	addedItem := table.addedItem
	table.Unlock()

	// 执行所有addedItem的回调函数
	if addedItem != nil {
		for _, callback := range addedItem {
			callback(item)
		}
	}

	// lifSpan 非0值，则非永久存在，需要定时删除操作
	// 这里减少操作，如果当前的项的lifeSpan比下一个cleanupInterval小，则需要立即进行清理工作
	// 否则可以等下一次cleanupInterval进行清理
	if itemReal.lifeSpan > 0 && (expDur == 0 || itemReal.lifeSpan < expDur) {
		table.expirationCheck()
	}
}

// 添加表项
func (table *CacheTable) Add(key interface{}, lifeSpan time.Duration, data interface{}) ciface.ICacheItem {
	item := NewCacheItem(key, lifeSpan, data)
	table.Lock()
	table.addInternal(item)
	// 这里没有defer Unlock
	return item
}

func (table *CacheTable) deleteInternal(key interface{}) (ciface.ICacheItem, error) {
	// 删除表项的内部函数。这里不用加锁，在调用方加锁了
	r, ok := table.items[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	rReal := interface{}(r).(*CacheItem)

	aboutToDeleteItem := table.aboutToDeleteItem
	table.Unlock()

	// 循环执行删除表项的回调函数，队列，先进先出
	if aboutToDeleteItem != nil {
		for _, callback := range aboutToDeleteItem {
			callback(r)
		}
	}

	rReal.RLock()
	defer rReal.RUnlock()
	if rReal.aboutToExpire != nil {
		for _, callback := range rReal.aboutToExpire {
			callback(key)
		}
	}

	table.Lock()
	table.log("Deleting item with key", key, "created on", rReal.createdOn, "and hit", rReal.accessCount, "times from table", table.name)
	delete(table.items, key)

	return r, nil
}

// 删除表项
func (table *CacheTable) Delete(key interface{}) (ciface.ICacheItem, error) {
	table.Lock()
	defer table.Unlock()

	return table.deleteInternal(key)
}

// 判断表项是否存在，需要读锁
func (table *CacheTable) Exists(key interface{}) bool {
	table.RLock()
	defer table.RUnlock()
	_, ok := table.items[key]
	return ok
}

// 如果不存在就添加
func (table *CacheTable) NotFoundAdd(key interface{}, lifeSpan time.Duration, data interface{}) bool {
	table.Lock()

	if _, ok := table.items[key]; ok {
		table.Unlock()
		return false
	}

	item := NewCacheItem(key, lifeSpan, data)
	table.addInternal(item)

	return true
}

// 获取cache值，每次访问会kept alive
func (table *CacheTable) Value(key interface{}, args ...interface{}) (ciface.ICacheItem, error) {
	table.RLock()
	r, ok := table.items[key]
	rReal, ok := (r).(*CacheItem)

	loadData := table.loadData
	table.RUnlock()

	if ok {
		rReal.KeepAlive()
		return rReal, nil
	}

	if loadData != nil {
		item := loadData(key, args...)
		if item != nil {
			itemReal := (item).(*CacheItem)
			table.Add(key, itemReal.lifeSpan, itemReal.data)
			return itemReal, nil
		}

		return nil, ErrKeyNotFoundOrLoadable
	}

	return nil, ErrKeyNotFound
}

// 清空所有cache表项
func (table *CacheTable) Flush() {
	table.Lock()
	defer table.Unlock()

	table.log("Flushing table", table.name)

	table.items = make(map[interface{}]ciface.ICacheItem)
	table.cleanupInterval = 0
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
}

// cache表项访问次数，用于统计热点cache
type CacheItemPair struct {
	Key         interface{}
	AccessCount int64
}

// 热点cache切片
type CacheItemPairList []CacheItemPair

// 排序所需基础函数
func (p CacheItemPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p CacheItemPairList) Len() int           { return len(p) }
func (p CacheItemPairList) Less(i, j int) bool { return p[i].AccessCount > p[j].AccessCount }

// 获取热点cache，统计所有表项，得出top X的cache表项
func (table *CacheTable) MostAccessed(count int64) []ciface.ICacheItem {
	table.RLock()
	defer table.RUnlock()

	p := make(CacheItemPairList, len(table.items))
	i := 0
	for k, v := range table.items {
		vReal := (v).(*CacheItem)
		p[i] = CacheItemPair{k, vReal.accessCount}
		i++
	}
	sort.Sort(p)

	var r []ciface.ICacheItem
	c := int64(0)
	for _, v := range p {
		if c >= count {
			break
		}

		item, ok := table.items[v.Key]
		if ok {
			r = append(r, item)
		}
		c++
	}

	return r
}

// 使用日志句柄打印日志
func (table *CacheTable) log(v ...interface{}) {
	if table.logger == nil {
		return
	}

	table.logger.Println(v...)
}
