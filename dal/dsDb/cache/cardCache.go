package cache

import (
	"github.com/patrickmn/go-cache"
	"time"
)

type CardCache struct {
	cache *cache.Cache
}

var cardCache *CardCache

type CardCacheInterface interface {
	getSequence(cardId string) int
	updateSequence(cardId string, sequence int) bool
}

func GetCardCache() *CardCache {
	if cardCache == nil {
		cardCache = &CardCache{cache: cache.New(5*time.Minute, 10*time.Minute)}
	}
	return cardCache
}

func (c *CardCache) GetSequence(cardId string) int {
	seq, ok := c.cache.Get(cardId)
	if !ok {
		return 0
	}
	return seq.(int)
}

func (c *CardCache) UpdateSequence(cardId string, sequence int) {
	c.cache.Set(cardId, sequence, 5*time.Minute)
}
