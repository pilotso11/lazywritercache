/*
 * Copyright (c) 2023. Vade Mecum Ltd.  All Rights Reserved.
 */

package lazywritercache

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testItem struct {
	id string
}

func (i testItem) Key() interface{} {
	return i.id
}

func (i testItem) CopyKeyDataFrom(from CacheItem) CacheItem {
	i.id = from.Key().(string)
	return i
}
func (i testItem) String() string {
	return i.id
}

func NewTestItem(key interface{}) CacheItem {
	return testItem{
		id: key.(string),
	}
}

func NewNoOpTestConfig(panics ...bool) Config {
	doPanics := len(panics) > 0 && panics[0]
	readerWriter := NewNoOpReaderWriter(NewTestItem, doPanics)
	return Config{
		handler:      readerWriter,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
}
func TestCacheStoreLoad(t *testing.T) {
	rfq := testItem{id: "test1"}
	rfq2 := testItem{id: "test2"}
	cache := NewLazyWriterCache(NewNoOpTestConfig())

	cache.Lock()
	cache.Save(rfq)
	cache.Save(rfq2)
	cache.Release()

	rfq3, ok := cache.GetAndLock("test1")
	cache.Release()
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, rfq, rfq3)

	rfq4, ok := cache.GetAndLock("test2")
	cache.Release()
	assert.Truef(t, ok, "loaded test2")
	assert.Equal(t, rfq2, rfq4)

	_, ok = cache.GetAndLock("missing")
	cache.Release()
	assert.Falsef(t, ok, "not loaded missing")

}

func TestCacheDirtyList(t *testing.T) {
	rfq := testItem{id: "test11"}
	rfq2 := testItem{id: "test22"}
	cache := NewLazyWriterCache(NewNoOpTestConfig())
	cache.Lock()
	cache.Save(rfq)
	cache.Save(rfq2)
	cache.Release()
	assert.Len(t, cache.dirty, 2, "dirty records")
	d := cache.getDirtyRecords()
	assert.Contains(t, d, rfq)
	assert.Contains(t, d, rfq2)
	assert.Len(t, cache.dirty, 0, "dirty records")

	cache.Lock()
	cache.Save(rfq2)
	cache.Release()
	assert.Len(t, cache.dirty, 1, "dirty records")
	d = cache.getDirtyRecords()
	assert.Contains(t, d, rfq2)
}

func TestCacheLockUnlockNoPanics(t *testing.T) {
	cache := NewLazyWriterCache(NewNoOpTestConfig())

	assert.NotPanics(t, func() {
		cache.Lock()
		cache.Release()
	}, "Lock and Release")
	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")

	assert.NotPanics(t, func() {
		cache.GetAndLock("missing")
		cache.Release()
	}, "get and Release")
	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")

	assert.NotPanics(t, func() {
		rfq := testItem{id: "test"}
		cache.GetAndLock("missing")
		cache.Save(&rfq)
		cache.Release()
	}, "get and Save")
	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")

}

func TestCachePanicOnBadLockState(t *testing.T) {
	cache := NewLazyWriterCache(NewNoOpTestConfig())

	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")
	assert.Panics(t, func() {
		cache.Save(testItem{})
	}, "Save when not locked")

	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")
	assert.Panics(t, func() {
		cache.Release()
	}, "Release when not locked")

	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")
	cache.locked.Store(true)
	assert.Panics(t, func() {
		cache.Lock()
	}, "Lock when not in mutex but locked")

}

func BenchmarkCacheWriteMax20k(b *testing.B) {
	cacheWrite(b, 20000)
}

func BenchmarkCacheWriteMax100k(b *testing.B) {
	cacheWrite(b, 100000)
}

func BenchmarkCacheRead20k(b *testing.B) {
	cacheRead(b, 20000)
}

func BenchmarkCacheRead100k(b *testing.B) {
	cacheRead(b, 100000)
}

func cacheWrite(b *testing.B, cacheSize int) {
	cache := NewLazyWriterCache(NewNoOpTestConfig())
	for i := 0; i < b.N; i++ {
		id := strconv.Itoa(i % cacheSize)
		rfq := testItem{id: id}
		cache.Lock()
		cache.Save(&rfq)
		cache.Release()
	}
	b.ReportAllocs()
}

func cacheRead(b *testing.B, cacheSize int) {
	// init
	cache := NewLazyWriterCache(NewNoOpTestConfig())
	var keys []string
	for i := 0; i < cacheSize; i++ {
		id := strconv.Itoa(i % cacheSize)
		keys = append(keys, id)
		rfq := testItem{id: id}
		cache.Lock()
		cache.Save(&rfq)
		cache.Release()
	}

	k := 0
	for i := 0; i < b.N; i++ {
		_, ok := cache.GetAndLock(keys[i%cacheSize])
		if ok {
			k++
		}
		cache.Release()
	}
	assert.Truef(b, k > 0, "critical failure")
	b.ReportAllocs()
}

func TestCacheEviction(t *testing.T) {

	cfg := NewNoOpTestConfig()
	cfg.Limit = 20
	cache := NewLazyWriterCache(cfg)

	for i := 0; i < 30; i++ {
		id := strconv.Itoa(i)
		rfq := testItem{id: id}
		cache.Lock()
		cache.Save(&rfq)
		cache.Release()
	}
	assert.Len(t, cache.cache, 30)
	cache.evictionProcessor()
	assert.Len(t, cache.cache, 20)
	_, ok := cache.cache["0"]
	assert.Falsef(t, ok, "0 has been evicted")
	_, ok = cache.cache["9"]
	assert.Falsef(t, ok, "9 has been evicted")
	_, ok = cache.cache["10"]
	assert.Truef(t, ok, "10 has not been evicted")
	_, ok = cache.cache["11"]
	assert.Truef(t, ok, "11 has not been evicted")
	_, ok = cache.cache["15"]
	assert.Truef(t, ok, "15 has not been evicted")
	_, ok = cache.cache["29"]
	assert.Truef(t, ok, "29 has not been evicted")

}

func TestGormLazyCache_GetAndRelease(t *testing.T) {
	rfq := testItem{id: "test1"}
	rfq2 := testItem{id: "test2"}
	cache := NewLazyWriterCache(NewNoOpTestConfig())

	cache.Lock()
	cache.Save(rfq)
	cache.Save(rfq2)
	cache.Release()

	rfq3, ok := cache.GetAndRelease("test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, rfq, rfq3)
	assert.Falsef(t, cache.locked.Load(), "not locked after GetAndRelease")

}

func TestGormLazyCache_GetAndReleaseWithForcedPanic(t *testing.T) {
	rfq := testItem{id: "test1"}
	rfq2 := testItem{id: "test2"}
	cfg := NewNoOpTestConfig(true)
	cache := NewLazyWriterCache(cfg)
	cache.LookupOnMiss = true

	cache.Lock()
	cache.Save(rfq)
	cache.Save(rfq2)
	cache.Release()

	rfq3, ok := cache.GetAndRelease("test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, rfq, rfq3)
	assert.Falsef(t, cache.locked.Load(), "not locked after GetAndRelease")

	assert.Panics(t, func() {
		_, ok := cache.GetAndRelease("test4")
		assert.Falsef(t, ok, "should not be found")
	})
	assert.Falsef(t, cache.locked.Load(), "not locked after GetAndRelease")
	assert.Equal(t, int64(1), cache.Misses.Load(), "1 miss expected")

}

func TestCacheStats_JSON(t *testing.T) {
	cache := NewLazyWriterCache(NewNoOpTestConfig())
	jsonStr := cache.JSON()

	stats := make(map[string]int64)

	err := json.Unmarshal([]byte(jsonStr), &stats)
	assert.Nil(t, err, "json parses")
	hits, ok := stats["hits"]
	assert.Truef(t, ok, "found in map")
	assert.Equal(t, int64(0), hits)
}
