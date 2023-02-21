package nhistory

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// NHistory is a history of items that can be used to prevent duplicate items
type NHistory struct {
	useHashing      bool
	entryTimeToLive time.Duration
	useRedis        bool
	redisClient     redis.UniversalClient
	redisContext    context.Context
	redisKey        string
	interval        *GoInterval
	safety          sync.Mutex
	hashes          map[string]time.Time // string -> TimeToDie
	hashFunction    func(string) string
}

// NewNHistory creates a new NHistory
func NewNHistory(name string, timeToLive time.Duration, cleanInterval time.Duration, redisClient *redis.UniversalClient, useHashing bool) *NHistory {
	redisKey, err := CreateRedisKey([]string{name}, "nhistory", ":")
	if err != nil {
		redisKey = "nhistory:" + NID("nh", 16)
	}
	history := &NHistory{
		hashes:          make(map[string]time.Time),
		useHashing:      useHashing,
		entryTimeToLive: timeToLive,
		redisClient:     *redisClient,
		useRedis:        (redisClient != nil),
		redisKey:        redisKey,
		redisContext:    context.Background(),
		hashFunction:    HashIt,
	}
	history.SetCleanInterval(cleanInterval)
	return history
}

// Add adds a new item to the history
func (history *NHistory) Add(key string, timeToDie time.Time) {
	var value string = key
	if history.useHashing {
		value = history.hashFunction(key)
	}
	if history.useRedis && history.redisClient != nil {
		timeScore := float64(timeToDie.Unix())
		history.redisClient.ZAdd(history.redisContext, history.redisKey, &redis.Z{Score: timeScore, Member: value})
	} else {
		history.safety.Lock()
		defer history.safety.Unlock()
		history.hashes[value] = timeToDie
	}
}

// SetTimeToLive sets the time to live for each entry
func (history *NHistory) SetTimeToLive(timeToLive time.Duration) {
	if timeToLive < 1 {
		return
	}
	history.entryTimeToLive = timeToLive
}

// SetRedisContext sets the redis context
func (history *NHistory) SetRedisContext(ctx context.Context) {
	history.redisContext = ctx
}

// SetCleanInterval sets the interval for cleaning the history
func (history *NHistory) SetCleanInterval(cleanInterval time.Duration) {
	if history.interval != nil {
		history.interval.Stop()
	}
	history.interval = Interval(func() bool {
		history.Clean()
		return true
	}, cleanInterval, false)
}

// SetHashFunction sets the hash function to use
func (history *NHistory) SetHashFunction(hashFunction func(string) string) {
	if hashFunction == nil {
		return
	}
	history.useHashing = true
}

// UseHashing sets whether to use hashing
func (history *NHistory) UseHashing(hash bool) {
	history.useHashing = hash
}

// Has checks if an item is in the history (and not expired)
func (history *NHistory) Has(key string) bool {
	var value string = key
	if history.useHashing {
		value = history.hashFunction(key)
	}
	now := time.Now()
	oldestAllowed := now.Add(-history.entryTimeToLive)
	minTimeScore := float64(oldestAllowed.Unix())
	if history.useRedis && history.redisClient != nil {
		return (history.redisClient.ZScore(history.redisContext, history.redisKey, value).Val() >= minTimeScore)
	} else {
		history.safety.Lock()
		defer history.safety.Unlock()
		_, ok := history.hashes[value]
		return ok
	}
}

// Get gets the time an item was added to the history.
// Get ignores expiration, hence you can successfully get an item that has expired and will return .Has -> false!
func (history *NHistory) Get(key string) (setAt time.Time, wasFound bool) {
	var value string = key
	if history.useHashing {
		value = history.hashFunction(key)
	}
	if history.useRedis && history.redisClient != nil {
		score := history.redisClient.ZScore(history.redisContext, history.redisKey, value).Val()
		if score > 0 {
			return time.Unix(int64(score), 0), true
		}
		return time.Time{}, false
	} else {
		history.safety.Lock()
		defer history.safety.Unlock()
		v, ok := history.hashes[value]
		return v, ok
	}
}

// Remove removes an item from the history
func (history *NHistory) Remove(key string) {
	var value string = key
	if history.useHashing {
		value = history.hashFunction(key)
	}
	if history.useRedis && history.redisClient != nil {
		history.redisClient.ZRem(history.redisContext, history.redisKey, value)
	} else {
		history.safety.Lock()
		defer history.safety.Unlock()
		delete(history.hashes, value)
	}
}

// Clean cleans the history and removes all expired entries.
func (history *NHistory) Clean() {
	now := time.Now()
	oldestAllowed := now.Add(-history.entryTimeToLive)
	if history.useRedis && history.redisClient != nil {
		max := strconv.FormatFloat(float64(oldestAllowed.Unix()), 'f', -1, 64)
		history.redisClient.ZRemRangeByScore(history.redisContext, history.redisKey, "-inf", max)
	} else {
		history.safety.Lock()
		defer history.safety.Unlock()
		for k, v := range history.hashes {
			if v.Before(oldestAllowed) {
				delete(history.hashes, k)
			}
		}
	}
}
