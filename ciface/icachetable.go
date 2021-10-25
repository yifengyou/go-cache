package ciface

import (
	"log"
	"time"
)

type ICacheTable interface {
	Count() int
	SetAddedItemCallback(f func(ICacheItem))
	Foreach(trans func(key interface{}, item ICacheItem))
	SetDataLoader(f func(interface{}, ...interface{}) ICacheItem)
	AddAddedItemCallback(f func(ICacheItem))
	RemoveAddedItemCallbacks()
	SetAboutToDeleteItemCallback(f func(ICacheItem))
	AddAboutToDeleteItemCallback(f func(ICacheItem))
	RemoveAboutToDeleteItemCallback()
	SetLogger(logger *log.Logger)
	Add(key interface{}, lifeSpan time.Duration, data interface{}) ICacheItem
	Delete(key interface{}) (ICacheItem, error)
	Exists(key interface{}) bool
	NotFoundAdd(key interface{}, lifeSpan time.Duration, data interface{}) bool
	Value(key interface{}, args ...interface{}) (ICacheItem, error)
	Flush()
	MostAccessed(count int64) []ICacheItem
}
