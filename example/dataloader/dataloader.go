/*
	SetDataLoader添加回调函数，当访问的key不存在时调用
 */
package main

import (
	"fmt"
	"github.com/yifengyou/go-cache/ciface"
	"github.com/yifengyou/go-cache/gocache"
	"strconv"
)

func main() {
	cache := gocache.NewCache("myCache")
	// 访问表项会触发SetDataLoader回调，当访问的cache不存在时调用
	cache.SetDataLoader(func(key interface{}, args ...interface{}) ciface.ICacheItem {
		val := "This is a test with key " + key.(string)
		fmt.Println("@Key not found!Add new key val : " + val)
		item := gocache.NewCacheItem(key, 0, val)
		return item
	})

	// 检查表项
	for i := 0; i < 10; i++ {
		res, err := cache.Value("someKey_" + strconv.Itoa(i))
		if err == nil {
			fmt.Println("Found value in cache:", res.Data())
		} else {
			fmt.Println("Error retrieving value from cache:", err)
		}
	}
	fmt.Println("@All done!")
}
