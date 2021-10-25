package ciface

import "time"

type ICacheItem interface {
	KeepAlive()
	LifeSpan() time.Duration
	AccessedOn() time.Time
	CreatedOn() time.Time
	AccessCount() int64
	Key() interface{}
	Data() interface{}
	SetAboutToExpireCallback(f func(interface{}))
	AddAboutToExpireCallback(f func(interface{}))
	RemoveAboutToExpireCallback()
}
