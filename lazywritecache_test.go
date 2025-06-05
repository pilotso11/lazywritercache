// MIT License
//
// Copyright (c) 2023 Seth Osher
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package lazywritercache

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

type testItem struct {
	id string
}

func (i testItem) Key() interface{} {
	return i.id
}

func (i testItem) CopyKeyDataFrom(from Cacheable) Cacheable {
	i.id = from.Key().(string)
	return i
}
func (i testItem) String() string {
	return i.id
}

func newTestItem(key interface{}) testItem {
	return testItem{
		id: key.(string),
	}
}

func newNoOpTestConfig(panics ...bool) Config[string, testItem] {
	doPanics := len(panics) > 0 && panics[0]
	readerWriter := NewNoOpReaderWriter[testItem](newTestItem, doPanics)
	return Config[string, testItem]{
		handler:      readerWriter,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
}
func TestCacheStoreLoad(t *testing.T) {
	item := testItem{id: "test1"}
	item2 := testItem{id: "test2"}
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	err = cache.Unlock()
	assert.NoError(t, err)

	item3, ok, err := cache.GetAndLock("test1")
	assert.NoError(t, err)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

	item4, ok, err := cache.GetAndLock("test2")
	assert.NoError(t, err)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.Truef(t, ok, "loaded test2")
	assert.Equal(t, item2, item4)

	_, ok, err = cache.GetAndLock("missing")
	assert.NoError(t, err)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.Falsef(t, ok, "not loaded missing")

}

func TestCacheDirtyList(t *testing.T) {
	item := testItem{id: "test11"}
	item2 := testItem{id: "test22"}
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.True(t, cache.IsDirty(), "dirty records")
	d, err := cache.getDirtyRecords()
	assert.NoError(t, err)
	assert.Contains(t, d, item)
	assert.Contains(t, d, item2)
	assert.False(t, cache.IsDirty(), "no dirty records")

	err = cache.Lock()
	assert.NoError(t, err)
	cache.Save(item2)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.True(t, cache.IsDirty(), "dirty records")
	d, err = cache.getDirtyRecords()
	assert.NoError(t, err)
	assert.Contains(t, d, item2)
}

func TestInvalidate(t *testing.T) {
	item := testItem{id: "test11"}
	item2 := testItem{id: "test22"}
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.True(t, cache.IsDirty(), "dirty records")

	err = cache.Invalidate()
	assert.NoError(t, err)
	assert.False(t, cache.IsDirty(), "no dirty records")
	assert.Len(t, cache.cache, 0, "cache is empty")
}

func TestCacheLockUnlockNoPanics(t *testing.T) {
	cache := NewLazyWriterCache(newNoOpTestConfig())
	defer cache.Shutdown()

	assert.NotPanics(t, func() {
		err := cache.Lock()
		assert.NoError(t, err)
		err = cache.Unlock()
		assert.NoError(t, err)
	}, "Lock and Unlock")
	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")

	assert.NotPanics(t, func() {
		_, _, err := cache.GetAndLock("missing")
		assert.NoError(t, err)
		err = cache.Unlock()
		assert.NoError(t, err)
	}, "get and Unlock")
	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")

	assert.NotPanics(t, func() {
		item := testItem{id: "test"}
		_, _, err := cache.GetAndLock("missing")
		assert.NoError(t, err)
		cache.Save(item)
		err = cache.Unlock()
		assert.NoError(t, err)
	}, "get and Save")
	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")

}

func TestCachePanicOnBadLockState(t *testing.T) {
	cache := NewLazyWriterCache(newNoOpTestConfig())
	defer cache.Shutdown()

	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")
	assert.Panics(t, func() {
		cache.Save(testItem{})
	}, "Save when not locked")

	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")
	err := cache.Unlock()
	assert.NotNil(t, err)
	assert.Panics(t, func() {
		cache.unlockWithPanic()
	}, "Unlock when not locked")

	assert.Falsef(t, cache.locked.Load(), "cache us unlocked")
	cache.locked.Store(true)
	err = cache.Lock()
	assert.NotNil(t, err)

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

func BenchmarkParallel_x5_CacheRead20k(b *testing.B) {
	cacheSize := 20000
	nThreads := 5

	parallelRun(b, cacheSize, nThreads)
}

func BenchmarkParallel_x10_CacheRead20k(b *testing.B) {
	cacheSize := 20000
	nThreads := 10

	parallelRun(b, cacheSize, nThreads)
}

func parallelRun(b *testing.B, cacheSize int, nThreads int) {
	cache := NewLazyWriterCache(newNoOpTestConfig())
	defer cache.Shutdown()

	var keys []string
	for i := 0; i < cacheSize; i++ {
		id := strconv.Itoa(i % cacheSize)
		keys = append(keys, id)
		item := testItem{id: id}
		err := cache.Lock()
		assert.Nil(b, err)
		cache.Save(item)
		err = cache.Unlock()
		assert.Nil(b, err)
	}

	wait := sync.WaitGroup{}
	for i := 0; i < nThreads; i++ {
		wait.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				key := rand.Intn(cacheSize)
				_, ok, err := cache.GetAndLock(keys[key])
				assert.Nil(b, err)
				if ok {
				}
				err = cache.Unlock()
				assert.Nil(b, err)
			}
			wait.Add(-1)
		}()
	}
	wait.Wait()
	b.ReportAllocs()
}

func cacheWrite(b *testing.B, cacheSize int) {
	cache := NewLazyWriterCache(newNoOpTestConfig())
	defer cache.Shutdown()

	for i := 0; i < b.N; i++ {
		id := strconv.Itoa(i % cacheSize)
		item := testItem{id: id}
		err := cache.Lock()
		assert.Nil(b, err)
		cache.Save(item)
		err = cache.Unlock()
		assert.Nil(b, err)
	}
	b.ReportAllocs()
}

func cacheRead(b *testing.B, cacheSize int) {
	// init
	cache := NewLazyWriterCache(newNoOpTestConfig())
	defer cache.Shutdown()

	var keys []string
	for i := 0; i < cacheSize; i++ {
		id := strconv.Itoa(i % cacheSize)
		keys = append(keys, id)
		item := testItem{id: id}
		err := cache.Lock()
		assert.Nil(b, err)
		cache.Save(item)
		err = cache.Unlock()
		assert.Nil(b, err)
	}

	k := 0
	for i := 0; i < b.N; i++ {
		key := rand.Intn(cacheSize)
		_, ok, err := cache.GetAndLock(keys[key])
		assert.Nil(b, err)
		if ok {
			k++
		}
		err = cache.Unlock()
		assert.Nil(b, err)
	}
	assert.Truef(b, k > 0, "critical failure")
	b.ReportAllocs()
}

func TestCacheEviction(t *testing.T) {

	cfg := newNoOpTestConfig()
	cfg.Limit = 20
	cache := NewLazyWriterCache(cfg)
	defer cache.Shutdown()

	for i := 0; i < 30; i++ {
		id := strconv.Itoa(i)
		item := testItem{id: id}
		err := cache.Lock()
		assert.NoError(t, err)
		cache.Save(item)
		err = cache.Unlock()
		assert.NoError(t, err)
	}
	assert.Len(t, cache.cache, 30)
	err := cache.evictionProcessor()
	assert.NoError(t, err)

	assert.Len(t, cache.cache, 30, "nothing evicted until flushed")
	err = cache.Flush()
	assert.NoError(t, err)
	err = cache.evictionProcessor()
	assert.NoError(t, err)
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
	item := testItem{id: "test1"}
	item2 := testItem{id: "test2"}
	cache := NewLazyWriterCache(newNoOpTestConfig())
	defer cache.Shutdown()

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	err = cache.Unlock()
	assert.NoError(t, err)

	item3, ok, err := cache.GetAndRelease("test1")
	assert.NoError(t, err)
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)
	assert.Falsef(t, cache.locked.Load(), "not locked after GetAndRelease")

}

func TestGormLazyCache_GetAndReleaseWithForcedPanic(t *testing.T) {
	item := testItem{id: "test1"}
	item2 := testItem{id: "test2"}
	cfg := newNoOpTestConfig(true)
	cache := NewLazyWriterCache(cfg)
	defer cache.Shutdown()

	cache.LookupOnMiss = true

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	err = cache.Unlock()
	assert.NoError(t, err)

	item3, ok, err := cache.GetAndRelease("test1")
	assert.NoError(t, err)
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)
	assert.Falsef(t, cache.locked.Load(), "not locked after GetAndRelease")

	assert.Panics(t, func() {
		_, ok, _ := cache.GetAndRelease("test4")
		assert.Falsef(t, ok, "should not be found")
	})
	assert.Falsef(t, cache.locked.Load(), "not locked after GetAndRelease")
	assert.Equal(t, int64(1), cache.Misses.Load(), "1 miss expected")

}

func TestCacheStats_JSON(t *testing.T) {
	cache := NewLazyWriterCache(newNoOpTestConfig())
	defer cache.Shutdown()

	jsonStr := cache.JSON()

	stats := make(map[string]int64)

	err := json.Unmarshal([]byte(jsonStr), &stats)
	assert.Nil(t, err, "json parses")
	hits, ok := stats["hits"]
	assert.Truef(t, ok, "found in map")
	assert.Equal(t, int64(0), hits)
}

func TestRange(t *testing.T) {
	item := testItem{id: "test1"}
	item2 := testItem{id: "test2"}
	item3 := testItem{id: "test3"}
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	cache.Save(item3)
	err = cache.Unlock()
	assert.NoError(t, err)

	n := 0
	r, err := cache.Range(func(k string, v testItem) bool {
		n++
		return true
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, r)

	assert.Equal(t, 3, n, "iterated over all cache items")
}

func TestRangeAbort(t *testing.T) {
	item := testItem{id: "test1"}
	item2 := testItem{id: "test2"}
	item3 := testItem{id: "test3"}
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	cache.Save(item3)
	err = cache.Unlock()
	assert.NoError(t, err)

	n := 0
	r, err := cache.Range(func(k string, v testItem) bool {
		n++
		if n == 2 {
			return false
		}
		return true
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, r)
	assert.Equal(t, 2, n, "iterated over all cache items")
}

func TestNoGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewLazyWriterCacheWithContext[string, testItem](ctx, newNoOpTestConfig())

	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "test"})
	err = cache.Unlock()
	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	cancel()
	cache.Shutdown()
	time.Sleep(300 * time.Millisecond)
}

func TestGetFromLockedErrIfNotLocked(t *testing.T) {
	assert.NotPanics(t, func() {
		cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
		_, _, err := cache.GetFromLocked("test")
		assert.NotNil(t, err)
	})
}

func TestEmptyCacheable(t *testing.T) {
	empty := EmptyCacheable{}

	// Test Key method
	key := empty.Key()
	assert.Equal(t, "", key, "EmptyCacheable.Key should return empty string")

	// Test CopyKeyDataFrom method
	item := testItem{id: "test"}
	result := empty.CopyKeyDataFrom(item)
	assert.Equal(t, item, result, "EmptyCacheable.CopyKeyDataFrom should return the input item")
}

func TestNewDefaultConfig(t *testing.T) {
	handler := NewNoOpReaderWriter[testItem](newTestItem, false)
	config := NewDefaultConfig[string, testItem](handler)

	// Verify default values
	assert.NotNil(t, config.handler, "Handler should not be nil")
	assert.Equal(t, 10000, config.Limit, "Default limit should be 10000")
	assert.True(t, config.LookupOnMiss, "LookupOnMiss should be true by default")
	assert.Equal(t, 500*time.Millisecond, config.WriteFreq, "Default WriteFreq should be 500ms")
	assert.Equal(t, 10*time.Second, config.PurgeFreq, "Default PurgeFreq should be 10s")
	assert.False(t, config.SyncWrites, "SyncWrites should be false by default")
	assert.False(t, config.FlushOnShutdown, "FlushOnShutdown should be false by default")
}

func TestCacheStats_String(t *testing.T) {
	stats := CacheStats{}
	stats.Hits.Store(10)
	stats.Misses.Store(5)
	stats.Stores.Store(15)
	stats.Evictions.Store(3)
	stats.DirtyWrites.Store(8)

	expected := "Hits: 10, Misses 5, Stores 15, Evictions 3, Dirty Writes: 8"
	assert.Equal(t, expected, stats.String(), "String representation should match expected format")
}

func TestClearDirty(t *testing.T) {
	item := testItem{id: "test1"}
	item2 := testItem{id: "test2"}
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	// Add items to the cache and make them dirty
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(item)
	cache.Save(item2)
	assert.Equal(t, 2, len(cache.dirty), "dirty records") // cant use IsDirty, we're already locked.

	// Test ClearDirty
	cache.ClearDirty()
	err = cache.Unlock()
	assert.NoError(t, err)

	assert.False(t, cache.IsDirty(), "dirty records")

	// Test that ClearDirty panics when cache is not locked
	assert.Panics(t, func() {
		cache.ClearDirty()
	}, "ClearDirty should panic when cache is not locked")
}

func TestIsDirty(t *testing.T) {
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	// Initially, cache should not be dirty
	assert.False(t, cache.IsDirty(), "Cache should not be dirty initially")

	// Add an item to make the cache dirty
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "test1"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Now cache should be dirty
	assert.True(t, cache.IsDirty(), "Cache should be dirty after saving an item")

	// Flush the cache
	err = cache.Flush()
	assert.NoError(t, err)

	// After flush, cache should not be dirty
	assert.False(t, cache.IsDirty(), "Cache should not be dirty after flush")
}

func TestGetKeys(t *testing.T) {
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	// Add items to the cache
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "test1"})
	cache.Save(testItem{id: "test2"})
	cache.Save(testItem{id: "test3"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Get keys
	keys, err := cache.getKeys()
	assert.NoError(t, err, "getKeys should not return an error")
	assert.Len(t, keys, 3, "Should have 3 keys")
	assert.Contains(t, keys, "test1", "Keys should contain test1")
	assert.Contains(t, keys, "test2", "Keys should contain test2")
	assert.Contains(t, keys, "test3", "Keys should contain test3")

	// Test getKeys when cache is already locked
	cache.locked.Store(true)
	_, err = cache.getKeys()
	assert.Error(t, err, "getKeys should return an error when cache is already locked")
	cache.locked.Store(false)
}

func TestEvictionManager(t *testing.T) {
	// Create a cache with a small limit and short purge frequency
	cfg := newNoOpTestConfig()
	cfg.Limit = 5
	cfg.PurgeFreq = 50 * time.Millisecond

	cache := NewLazyWriterCache[string, testItem](cfg)
	defer cache.Shutdown()

	// Add more items than the limit
	for i := 0; i < 10; i++ {
		err := cache.Lock()
		assert.NoError(t, err)
		cache.Save(testItem{id: strconv.Itoa(i)})
		err = cache.Unlock()
		assert.NoError(t, err)
	}

	// Flush the cache to make items eligible for eviction
	err := cache.Flush()
	assert.NoError(t, err)

	// Wait for eviction manager to run
	time.Sleep(200 * time.Millisecond)

	// Check that the cache size is now at or below the limit
	err = cache.Lock()
	assert.NoError(t, err)
	cacheSize := len(cache.cache)
	err = cache.Unlock()
	assert.NoError(t, err)

	assert.LessOrEqual(t, cacheSize, cfg.Limit, "Cache size should be at or below the limit after eviction")
	assert.Greater(t, cache.Evictions.Load(), int64(0), "Evictions counter should be incremented")
}

func TestLazyWriter(t *testing.T) {
	// Create a cache with a short write frequency
	cfg := newNoOpTestConfig()
	cfg.WriteFreq = 50 * time.Millisecond

	cache := NewLazyWriterCache[string, testItem](cfg)
	defer cache.Shutdown()

	// Add items to the cache
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "test1"})
	cache.Save(testItem{id: "test2"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Verify items are marked as dirty
	assert.True(t, cache.IsDirty(), "dirty records")

	// Wait for lazy writer to run
	time.Sleep(200 * time.Millisecond)

	// Verify dirty items were processed
	assert.False(t, cache.IsDirty(), "dirty records")
	assert.Greater(t, cache.DirtyWrites.Load(), int64(0), "DirtyWrites counter should be incremented")
}

func TestNewLazyWriterCacheWithContext_Cancellation(t *testing.T) {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Create a cache with the context
	cfg := newNoOpTestConfig()
	cfg.WriteFreq = 50 * time.Millisecond
	cfg.PurgeFreq = 50 * time.Millisecond
	cfg.FlushOnShutdown = true

	cache := NewLazyWriterCacheWithContext[string, testItem](ctx, cfg)

	// Add an item to the cache
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "test1"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Cancel the context to trigger shutdown
	cancel()

	// Wait for goroutines to exit
	time.Sleep(200 * time.Millisecond)

	// Verify that the cache was flushed
	assert.False(t, cache.IsDirty(), 0, "Dirty list should be empty after flush on shutdown")
}

func TestGetAndLock_Error(t *testing.T) {
	cache := NewLazyWriterCache[string, testItem](newNoOpTestConfig())
	defer cache.Shutdown()

	// Lock the cache to simulate concurrent access
	cache.locked.Store(true)

	// GetAndLock should return an error when the cache is already locked
	_, _, err := cache.GetAndLock("test")
	assert.Error(t, err, "GetAndLock should return an error when cache is already locked")
	assert.Equal(t, ErrConcurrentModification, err, "Error should be ErrConcurrentModification")

	// Reset lock state
	cache.locked.Store(false)
}

func TestGetFromLocked_CompleteCodeCoverage(t *testing.T) {
	cfg := newNoOpTestConfig()
	cfg.LookupOnMiss = true
	cache := NewLazyWriterCache[string, testItem](cfg)
	defer cache.Shutdown()

	// Test when cache is not locked
	_, _, err := cache.GetFromLocked("test")
	assert.Error(t, err, "GetFromLocked should return an error when cache is not locked")
	assert.Equal(t, ErrNotLocked, err, "Error should be ErrNotLocked")

	// Lock the cache
	err = cache.Lock()
	assert.NoError(t, err)

	// Test when item is in cache
	cache.Save(testItem{id: "test1"})
	item, found, err := cache.GetFromLocked("test1")
	assert.NoError(t, err, "GetFromLocked should not return an error")
	assert.True(t, found, "Item should be found")
	assert.Equal(t, "test1", item.id, "Item ID should match")

	// Test when item is not in cache but LookupOnMiss is true
	// The NoOpReaderWriter's Find method always returns an error, so the item won't be found
	// GetFromLocked doesn't propagate the error from handler.Find, it just returns found=false
	item, found, err = cache.GetFromLocked("test2")
	assert.NoError(t, err, "GetFromLocked should not return an error when item is not found")
	assert.False(t, found, "Item should not be found via lookup because NoOpReaderWriter always returns an error")

	// Unlock the cache
	err = cache.Unlock()
	assert.NoError(t, err)
}

func TestSaveDirtyToDB_CompleteCodeCoverage(t *testing.T) {
	// Create a cache with a handler that will return an error on Save
	handler := NewNoOpReaderWriter[testItem](newTestItem, false)
	cfg := Config[string, testItem]{
		handler:      handler,
		Limit:        1000,
		LookupOnMiss: true,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
	cache := NewLazyWriterCache[string, testItem](cfg)
	defer cache.Shutdown()

	// Add items to the cache
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "test1"})
	cache.Save(testItem{id: "test2"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Verify items are marked as dirty
	assert.True(t, cache.IsDirty(), "dirty records")

	// Flush the cache
	err = cache.Flush()
	assert.NoError(t, err)

	// Verify dirty items were processed
	assert.False(t, cache.IsDirty(), "Dirty list should be empty after flush")
	assert.Equal(t, int64(2), cache.DirtyWrites.Load(), "DirtyWrites counter should be incremented")
}

// CustomReaderWriter extends NoOpReaderWriter to simulate specific scenarios
type CustomReaderWriter[T Cacheable] struct {
	NoOpReaderWriter[T]
	existingKeys        map[string]bool // Keys that should be treated as existing in the DB
	deadlockRemainCount atomic.Int64    // If not 0, trigger a deadlock on save and decrement
	dupKeyCount         atomic.Int64    // If not 0, trigger a duplicate key error on save and decrement
}

// Track save attempts across all instances
var saveAttempts = make(map[string]int)
var saveAttemptsMutex sync.Mutex

func NewCustomReaderWriter[T Cacheable](itemTemplate func(key any) T) *CustomReaderWriter[T] {
	// Reset save attempts for testing
	saveAttemptsMutex.Lock()
	saveAttempts = make(map[string]int)
	saveAttemptsMutex.Unlock()

	return &CustomReaderWriter[T]{
		NoOpReaderWriter: NoOpReaderWriter[T]{
			getTemplateItem: itemTemplate,
			panicOnWrite:    false,
			panicOnLoad:     false,
		},
		existingKeys: make(map[string]bool),
	}
}

func (g *CustomReaderWriter[T]) Find(key string, _ interface{}) (T, error) {
	template := g.getTemplateItem(key)
	if g.existingKeys[key] {
		// Simulate finding an existing record
		return template, nil
	}
	return template, errors.New("NoOp, item not found")
}

func (g *CustomReaderWriter[T]) Save(item T, _ interface{}) error {
	if g.deadlockRemainCount.Load() > 0 {
		g.deadlockRemainCount.Add(-1)
		// Simulate a deadlock error on first attempt
		return errors.New("code 1213: Deadlock found when trying to get lock")
	}
	if g.dupKeyCount.Load() > 0 {
		g.dupKeyCount.Add(-1)
		return errors.New("code 1234: duplicate key exception")
	}
	return nil
}

// Test for line 300 - updating an existing DB record
func TestSaveDirtyToDB_UpdateExistingRecord(t *testing.T) {
	// Create a custom handler that simulates finding an existing record
	handler := NewCustomReaderWriter[testItem](newTestItem)
	handler.existingKeys["existing"] = true

	cfg := Config[string, testItem]{
		handler:      handler,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
	cache := NewLazyWriterCache[string, testItem](cfg)
	defer cache.Shutdown()

	// Add an item to the cache with a key that will be found in the DB
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "existing"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Flush the cache
	err = cache.Flush()
	assert.NoError(t, err)

	// Verify the item was processed
	assert.False(t, cache.IsDirty(), "Dirty list should be empty after flush")
	assert.Equal(t, int64(1), cache.DirtyWrites.Load(), "DirtyWrites counter should be incremented")
}

func TestSaveToDB_DeadlockRetry(t *testing.T) {
	// Create a custom handler that simulates finding an existing record
	handler := NewCustomReaderWriter[testItem](newTestItem)
	handler.existingKeys["existing"] = true

	// Set one pending deadlock
	handler.deadlockRemainCount.Store(1)

	cfg := Config[string, testItem]{
		handler:      handler,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
	cache := NewLazyWriterCache[string, testItem](cfg)
	defer cache.Shutdown()

	// Add an item to the cache with a key that will be found in the DB
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "existing"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Flush the cache
	err = cache.Flush()
	assert.Error(t, err)
	assert.True(t, cache.IsDirty(), "Still dirty after deadlock")

	// Flush the cache again
	err = cache.Flush()
	assert.NoError(t, err)

	// Verify the item was processed
	assert.False(t, cache.IsDirty(), "Dirty list should be empty after flush")
	assert.Equal(t, int64(1), cache.DirtyWrites.Load(), "DirtyWrites counter should be incremented")
}

func TestSaveToDB_NonRecoverableError(t *testing.T) {
	// Create a custom handler that simulates finding an existing record
	handler := NewCustomReaderWriter[testItem](newTestItem)
	handler.existingKeys["existing"] = true

	// Set one pending deadlock
	handler.dupKeyCount.Store(1)

	cfg := Config[string, testItem]{
		handler:      handler,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
		SyncWrites:   true,
	}
	cache := NewLazyWriterCache[string, testItem](cfg)
	defer cache.Shutdown()

	// Add an item to the cache with a key that will be found in the DB
	err := cache.Lock()
	assert.NoError(t, err)
	cache.Save(testItem{id: "existing"})
	err = cache.Unlock()
	assert.NoError(t, err)

	// Flush the cache
	err = cache.Flush()
	assert.Error(t, err)
	assert.False(t, cache.IsDirty(), "Not dirty after discard")

	// Flush the cache again
	err = cache.Flush()
	assert.NoError(t, err)

	// Verify the item was processed
	assert.False(t, cache.IsDirty(), "Dirty list should be empty after flush")
	assert.Equal(t, int64(0), cache.DirtyWrites.Load(), "DirtyWrites counter should be 0, nothing was written")
	assert.Equal(t, int64(1), cache.FailedWrites.Load(), "FailedWrites counter should be 1")
}
