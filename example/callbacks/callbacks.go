package main

import (
	"fmt"
	"github.com/yifengyou/go-cache/ciface"
	"github.com/yifengyou/go-cache/gocache"
	"os"
	"time"
)

func main() {
	cache := gocache.NewCache("myCache")

	fmt.Println("@register add callback")
	// 设置一个添加表项时的回调函数
	cache.SetAddedItemCallback(func(entry ciface.ICacheItem) {
		fmt.Println("Added Callback 1:", entry.Key(), entry.Data(), entry.CreatedOn())
	})

	fmt.Println("@register delete callback")
	// 配置删除表项时候的回调函数
	cache.SetAboutToDeleteItemCallback(func(entry ciface.ICacheItem) {
		fmt.Println("Deleting:", entry.Key(), entry.Data(), entry.CreatedOn())
	})

	fmt.Println("@add key ...")
	cache.Add("Key1", 0, "Key1 data")
	cache.Add("Key2", 0, "Key2 data")
	cache.Add("Key3", 0, "Key3 data")

	fmt.Println("@test get key")
	res, err := cache.Value("Key1")
	if err == nil {
		fmt.Println("Found value in cache:", res.Data())
	} else {
		fmt.Println("Error retrieving value from cache:", err)
		os.Exit(1)
	}
	fmt.Println("@test delete key")
	// 删除表项触发回调函数
	_, err = cache.Delete("Key1")

	fmt.Println("@remove add item callback")
	// 清空所有删除回调函数
	cache.RemoveAddedItemCallbacks()
	res = cache.Add("Key4", 3*time.Second, "Key4 data")

	fmt.Println("@add key expire callback")
	// 设置表项过期触发的回调函数
	res.SetAboutToExpireCallback(func(key interface{}) {
		fmt.Println("About to expire:", key.(string))
	})

	fmt.Println("wait 3 second to delete and wait 6 second to exit...")
	time.Sleep(10 * time.Second)

	fmt.Println("@try to get key which already delete")
	res, err = cache.Value("Key4")
	if err == nil {
		fmt.Println("Found value in cache:", res.Data())
	} else {
		fmt.Println("Error retrieving value from cache:", err)
	}
	fmt.Println("All done!")
}
