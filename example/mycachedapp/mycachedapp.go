/*
	任意类型的Value，Key还是String
*/
package main

import (
	"fmt"
	"github.com/yifengyou/go-cache/ciface"
	"github.com/yifengyou/go-cache/gocache"
	"time"
)

type myStruct struct {
	text     string
	moreData []byte
}

func main() {
	cache := gocache.NewCache("myCache")

	cache.SetAddedItemCallback(func(entry ciface.ICacheItem) {
		fmt.Println("Added Callback 1:", entry.Key(), entry.Data(), entry.CreatedOn())
	})

	// 初始化数据，放到cache表项中
	val := myStruct{"This is Key1 data!", []byte{1, 2, 3, 4, 5}}
	cache.Add("Key1", 5*time.Second, &val)

	// 获取表项
	res, err := cache.Value("Key1")
	if err == nil {
		fmt.Println("Found value in cache:", res.Data().(*myStruct).text)
	} else {
		fmt.Println("Error retrieving value from cache:", err)
	}

	// 等待6秒，查看表项
	time.Sleep(6 * time.Second)
	res, err = cache.Value("Key1")
	if err != nil {
		fmt.Println("Item is not cached (anymore).")
	}

	cache.Add("Key2", 0, &val)

	cache.SetAboutToDeleteItemCallback(func(e ciface.ICacheItem) {
		fmt.Println("Deleting:", e.Key(), e.Data().(*myStruct).text, e.CreatedOn())
	})

	cache.Delete("Key2")

	// 清空表项
	cache.Flush()
	fmt.Println("@All done!")
}
