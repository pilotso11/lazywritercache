// MIT License
//
// Copyright (c) LF0LF3 Seth Osher
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
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

type testItemLF struct {
	id string
}

func (i testItemLF) Key() string {
	return i.id
}

func (i testItemLF) CopyKeyDataFrom(from CacheableLF) CacheableLF {
	i.id = from.Key()
	return i
}
func (i testItemLF) String() string {
	return i.id
}

func newtestItemLF(key string) testItemLF {
	return testItemLF{
		id: key,
	}
}

func newNoOpTestConfigLF() ConfigLF[testItemLF] {
	readerWriter := NewNoOpReaderWriterLF[testItemLF](newtestItemLF)
	return ConfigLF[testItemLF]{
		handler:      readerWriter,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
}
func TestCacheStoreLoadLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load(ctx, "test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

	item4, ok := cache.Load(ctx, "testLF")
	assert.Truef(t, ok, "loaded testLF")
	assert.Equal(t, itemLF, item4)

	_, ok = cache.Load(ctx, "missing")
	assert.Falsef(t, ok, "not loaded missing")

}

func TestCacheDirtyListLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test11"}
	itemLF := testItemLF{id: "testLFLF"}
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "dirty records")
	// assert.True(t, findIn(cache.dirty, item)) // Before
	// assert.True(t, findIn(cache.dirty, itemLF)) // Before
	_, ok := cache.dirty.Load(item.Key())
	assert.True(t, ok, "item should be in dirty list")
	_, ok = cache.dirty.Load(itemLF.Key())
	assert.True(t, ok, "itemLF should be in dirty list")

	cache.Save(itemLF) // Save again
	assert.Equal(t, 2, cache.dirty.Size(), "dirty records count should not change on re-save")
	// assert.True(t, findIn(cache.dirty, itemLF)) // Before
	_, ok = cache.dirty.Load(itemLF.Key())
	assert.True(t, ok, "itemLF should still be in dirty list after re-save")

	// Use ctx to avoid unused variable warning
	_, _ = cache.Load(ctx, "nonexistent")
}

func TestCacheLockUnlockNoPanicsLF(t *testing.T) {
	ctx := context.Background()
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
	defer cache.Shutdown()

	assert.NotPanics(t, func() {
		cache.Load(ctx, "missing")
	}, "get and Unlock")

	assert.NotPanics(t, func() {
		item := testItemLF{id: "test"}
		cache.Load(ctx, "missing")
		cache.Save(item)
	}, "get and Save")
}

func BenchmarkCacheWriteMax20kLF(b *testing.B) {
	cacheWriteLF(b, 20000)
}

func BenchmarkCacheWriteMax100kLF(b *testing.B) {
	cacheWriteLF(b, 100000)
}

func BenchmarkCacheRead20kLF(b *testing.B) {
	cacheReadLF(b, 20000)
}

func BenchmarkCacheRead100kLF(b *testing.B) {
	cacheReadLF(b, 100000)
}

func BenchmarkParallel_x5_CacheRead20kLF(b *testing.B) {
	cacheSize := 20000
	nThreads := 5

	parallelRunLF(b, cacheSize, nThreads)
}

func BenchmarkParallel_x10_CacheRead20kLF(b *testing.B) {
	cacheSize := 20000
	nThreads := 10

	parallelRunLF(b, cacheSize, nThreads)
}

func BenchmarkParallel_x20_CacheRead20kLF(b *testing.B) {
	cacheSize := 20000
	nThreads := 20

	parallelRunLF(b, cacheSize, nThreads)
}

func parallelRunLF(b *testing.B, cacheSize int, nThreads int) {
	ctx := context.Background()
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
	defer cache.Shutdown()

	var keys []string
	for i := 0; i < cacheSize; i++ {
		id := strconv.Itoa(i % cacheSize)
		keys = append(keys, id)
		item := testItemLF{id: id}
		cache.Save(item)
	}

	var wait sync.WaitGroup
	for i := 0; i < nThreads; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := 0; i < b.N; i++ {
				key := rand.Intn(cacheSize)
				_, ok := cache.Load(ctx, keys[key])
				if ok {
				}
			}
		}()
	}
	wait.Wait()
	b.ReportAllocs()
}

func cacheWriteLF(b *testing.B, cacheSize int) {
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
	defer cache.Shutdown()

	for i := 0; i < b.N; i++ {
		id := strconv.Itoa(i % cacheSize)
		item := testItemLF{id: id}
		cache.Save(item)
	}
	b.ReportAllocs()
}

func cacheReadLF(b *testing.B, cacheSize int) {
	// init
	ctx := context.Background()
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
	defer cache.Shutdown()

	var keys []string
	for i := 0; i < cacheSize; i++ {
		id := strconv.Itoa(i % cacheSize)
		keys = append(keys, id)
		item := testItemLF{id: id}
		cache.Save(item)
	}

	k := 0
	for i := 0; i < b.N; i++ {
		key := rand.Intn(cacheSize)
		_, ok := cache.Load(ctx, keys[key])
		if ok {
			k++
		}
	}
	assert.Truef(b, k > 0, "critical failure")
	b.ReportAllocs()
}

func TestCacheEvictionLF(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.Limit = 20
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()

	for i := 0; i < 30; i++ {
		id := strconv.Itoa(i)
		item := testItemLF{id: id}
		cache.Save(item)
	}
	assert.Equal(t, 30, cache.cache.Size())
	cache.evictionProcessor(ctx)
	assert.Equal(t, 30, cache.cache.Size(), "nothing evicted until flushed")
	cache.Flush(ctx)
	cache.evictionProcessor(ctx)
	assert.Equal(t, 20, cache.cache.Size())
	_, ok := cache.cache.Load("0")
	assert.Truef(t, ok, "0 has not been evicted")
	_, ok = cache.cache.Load("1")
	assert.Falsef(t, ok, "1 has  been evicted")
	_, ok = cache.cache.Load("9")
	assert.Falsef(t, ok, "9 has been evicted")
	_, ok = cache.cache.Load("10")
	assert.Falsef(t, ok, "10 has been evicted")
	_, ok = cache.cache.Load("11")
	assert.Truef(t, ok, "11 has not been evicted")
	_, ok = cache.cache.Load("15")
	assert.Truef(t, ok, "15 has not been evicted")
	_, ok = cache.cache.Load("29")
	assert.Truef(t, ok, "29 has not been evicted")

}

func Test_GetAndReleaseLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load(ctx, "test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

}

func Test_GetAndReleaseWithForcedPanicLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()

	cache.LookupOnMiss = true

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load(ctx, "test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

	cfg.handler.(NoOpReaderWriterLF[testItemLF]).panicOnNext.Store(true)
	assert.Panics(t, func() {
		_, ok := cache.Load(ctx, "test4")
		assert.Falsef(t, ok, "should not be found")
	})
	assert.Equal(t, int64(1), cache.Misses.Load(), "1 miss expected")

}

func TestCacheStats_JSONLF(t *testing.T) {
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
	defer cache.Shutdown()

	jsonStr := cache.JSON()

	stats := make(map[string]int64)

	err := json.Unmarshal([]byte(jsonStr), &stats)
	assert.Nil(t, err, "json parses")
	hits, ok := stats["hits"]
	assert.Truef(t, ok, "found in map")
	assert.Equal(t, int64(0), hits)
}

func TestRangeLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	item3 := testItemLF{id: "test3"}
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)
	cache.Save(item3)

	n := 0
	cache.Range(func(k string, v testItemLF) bool {
		n++
		return true
	})

	assert.Equal(t, 3, n, "iterated over all cache items")
}

func TestRangeAbortLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	item3 := testItemLF{id: "test3"}
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)
	cache.Save(item3)

	n := 0
	cache.Range(func(k string, v testItemLF) bool {
		n++
		if n == 2 {
			return false
		}
		return true
	})

	assert.Equal(t, 2, n, "iterated over all cache items")
}

func TestNoGoroutineLeaksLF(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewLazyWriterCacheWithContextLF[testItemLF](ctx, newNoOpTestConfigLF())
	cache.Save(testItemLF{id: "test"})
	time.Sleep(100 * time.Millisecond)
	cancel()
	cache.Shutdown()
	time.Sleep(100 * time.Millisecond)
}

func TestNewDefaultConfigLF(t *testing.T) {
	handler := NewNoOpReaderWriterLF[testItemLF](newtestItemLF)
	config := NewDefaultConfigLF[testItemLF](handler)

	// Verify default values
	assert.NotNil(t, config.handler, "Handler should not be nil")
	assert.Equal(t, 10000, config.Limit, "Default limit should be 10000")
	assert.True(t, config.LookupOnMiss, "LookupOnMiss should be true by default")
	assert.Equal(t, 500*time.Millisecond, config.WriteFreq, "Default WriteFreq should be 500ms")
	assert.Equal(t, 10*time.Second, config.PurgeFreq, "Default PurgeFreq should be 10s")
	assert.False(t, config.FlushOnShutdown, "FlushOnShutdown should be false by default")
}

func TestEmptyCacheableLF(t *testing.T) {
	empty := EmptyCacheableLF{}

	// Test Key method
	key := empty.Key()
	assert.Equal(t, "", key, "EmptyCacheableLF.Key should return empty string")

	// Test CopyKeyDataFrom method
	item := testItemLF{id: "test"}
	result := empty.CopyKeyDataFrom(item)
	assert.Equal(t, item, result, "EmptyCacheableLF.CopyKeyDataFrom should return the input item")
}

func TestClearDirtyLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	item2 := testItemLF{id: "test2"}
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
	defer cache.Shutdown()

	// Add items to the cache and make them dirty
	cache.Save(item)
	cache.Save(item2)
	assert.Equal(t, 2, cache.dirty.Size(), "Should have 2 dirty items")

	// Test ClearDirty
	cache.ClearDirty()
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty after ClearDirty")
}

func TestPeriodicSaveLF(t *testing.T) {
	// Create a cache with a short write frequency
	cfg := newNoOpTestConfigLF()
	cfg.WriteFreq = 50 * time.Millisecond

	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	// Add items to the cache
	cache.Save(testItemLF{id: "test1"})
	cache.Save(testItemLF{id: "test2"})

	// Verify items are marked as dirty
	assert.Equal(t, 2, cache.dirty.Size(), "Should have 2 dirty items")

	assert.Eventuallyf(t, func() bool {
		return !cache.IsDirty()
	}, 100*time.Millisecond, time.Millisecond, "Cache should not be dirty after save")

	// Verify dirty items were processed
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty after lazy writer runs")
	assert.Greater(t, cache.DirtyWrites.Load(), int64(0), "DirtyWrites counter should be incremented")
}

func TestPeriodicEvictionsLF(t *testing.T) {
	// Create a cache with a small limit and short purge frequency
	cfg := newNoOpTestConfigLF()
	cfg.Limit = 5
	cfg.PurgeFreq = 50 * time.Millisecond
	cfg.WriteFreq = 50 * time.Millisecond // Need to flush dirty items for eviction to work

	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	// Add more items than the limit
	for i := 0; i < 10; i++ {
		cache.Save(testItemLF{id: strconv.Itoa(i)})
	}

	// Verify all items are in the cache initially
	assert.Equal(t, 10, cache.cache.Size(), "Should have 10 items in cache initially")

	// Wait for eviction manager to run multiple times
	// We need to wait longer to ensure the eviction process completes
	assert.Eventuallyf(t, func() bool {
		return cache.cache.Size() <= cfg.Limit
	}, 500*time.Millisecond, time.Millisecond, "Cache size should be at or below the limit after eviction")

	// Verify cache size is now at or below the limit
	assert.LessOrEqual(t, cache.cache.Size(), cfg.Limit, "Cache size should be at or below the limit after eviction")
	assert.Greater(t, cache.Evictions.Load(), int64(0), "Evictions counter should be incremented")
}

func TestFlushOnShutdownLF(t *testing.T) {
	// Create a cache with FlushOnShutdown enabled
	cfg := newNoOpTestConfigLF()
	cfg.WriteFreq = 1 * time.Hour // Long enough that it won't trigger during the test
	cfg.FlushOnShutdown = true

	ctx, cancel := context.WithCancel(context.Background())
	cache := NewLazyWriterCacheWithContextLF[testItemLF](ctx, cfg)

	// Add items to the cache
	cache.Save(testItemLF{id: "test1"})
	cache.Save(testItemLF{id: "test2"})

	// Verify items are marked as dirty
	assert.Equal(t, 2, cache.dirty.Size(), "Should have 2 dirty items")

	// Cancel the context to trigger shutdown
	cancel()

	// Wait a bit for the shutdown to complete
	assert.Eventuallyf(t, func() bool {
		return cache.dirty.Size() == 0
	}, 100*time.Millisecond, time.Millisecond, "Cache should be empty after shutdown with FlushOnShutdown=true")

	// Verify dirty items were processed during shutdown
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty after shutdown with FlushOnShutdown=true")
	assert.Greater(t, cache.DirtyWrites.Load(), int64(0), "DirtyWrites counter should be incremented")
}

func TestRequeueRecoverableErrLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("save deadlock")
	cache.Flush(ctx)
	assert.Equal(t, int64(0), testHandler.warnCount.Load(), "Warning received")
	assert.Equal(t, int64(3), testHandler.infoCount.Load(), "Info received")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
}

func TestRequeueSkipsNonRecoverableErrLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("save duplicate key")
	cache.Flush(ctx)
	assert.Equal(t, int64(1), testHandler.warnCount.Load(), "Warning received")
	assert.Equal(t, int64(2), testHandler.infoCount.Load(), "Info received")
	assert.Equal(t, 1, cache.dirty.Size(), "1 items should be in the cache")
}

func TestRequeueCommitRecoverableErrLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("commit deadlock")
	cache.Flush(ctx)
	assert.Equal(t, int64(2), testHandler.warnCount.Load(), "Warning received")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
}

func TestRequeueCommitSkipsNonRecoverableErrLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("commit duplicate key")
	cache.Flush(ctx)
	assert.Equal(t, int64(2), testHandler.warnCount.Load(), "Warning received")
	assert.Equal(t, 0, cache.dirty.Size(), "0 items should be in the cache")
}

func TestRequeueBeginRecoverableErrLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("begin db bad connection")
	cache.Flush(ctx)
	assert.Equal(t, int64(2), testHandler.infoCount.Load(), "Info received")
	assert.Equal(t, int64(0), testHandler.warnCount.Load(), "Warning received")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
}

func TestRequeueRollbackRecoverableErrLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("save deadlock,rollback deadlock")
	cache.Flush(ctx)
	assert.Equal(t, "", testHandler.errorOnNext.Load(), "both errors handled")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	assert.Equal(t, int64(0), testHandler.warnCount.Load(), "Warnings received")
	assert.Equal(t, int64(4), testHandler.infoCount.Load(), "Info received")
}

func TestRequeueRollbackUnrecoverableErrLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("save deadlock,rollback failed")
	cache.Flush(ctx)
	assert.Equal(t, "", testHandler.errorOnNext.Load(), "both errors handled")
	assert.Equal(t, 1, cache.dirty.Size(), "1 item should be in the cache")
	assert.Equal(t, int64(1), testHandler.warnCount.Load(), "Warnings received")
	assert.Equal(t, int64(2), testHandler.infoCount.Load(), "Info received")
}

func TestPanicHandlerLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	testHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()
	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should be in the cache")
	testHandler.errorOnNext.Store("save deadlock,panic")
	cache.Flush(ctx)
	assert.Equal(t, "", testHandler.errorOnNext.Load(), "both errors handled")
	assert.Equal(t, 1, cache.dirty.Size(), "1 items should be in the cache")
	assert.Equal(t, int64(1), testHandler.warnCount.Load(), "Warnings received")
}

func TestSaveDirtyToDB_AllowConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.AllowConcurrentWrites = true // Explicitly enable
	cfg.WriteFreq = 0                // Disable periodic writer for manual flush

	mockHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(newtestItemLF("item1"))
	cache.Save(newtestItemLF("item2"))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		cache.Flush(ctx) // Call Flush concurrently
	}()
	go func() {
		defer wg.Done()
		cache.Flush(ctx) // Call Flush concurrently
	}()

	wg.Wait()

	// With AllowConcurrentWrites = true, both flushes might attempt to write.
	// The NoOpReaderWriterLF doesn't have internal state to show distinct writes easily,
	// but we can check that items were processed (became non-dirty).
	// A more sophisticated mock could track concurrent access.
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty")
	// We expect DirtyWrites to be 2, but due to the nature of NoOp and potential race in clearing dirty,
	// it might be 2 or more if the test runs fast enough for multiple calls to saveDirtyToDB to interleave
	// before dirty list is fully cleared by one.
	// For this test, primarily ensure no deadlock and items are cleared.
	assert.GreaterOrEqual(t, mockHandler.infoCount.Load(), int64(1), "At least one flush attempt should log info")
}

func TestSaveDirtyToDB_DisallowConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.AllowConcurrentWrites = false // Explicitly disable (default, but good to test)
	cfg.WriteFreq = 0                 // Disable periodic writer

	mockHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(newtestItemLF("item1"))

	// Simulate one write already in progress by setting the atomic bool
	cache.writing.Store(true)

	// Attempt another flush, it should return early
	cache.Flush(ctx)

	assert.Equal(t, 1, cache.dirty.Size(), "Item should remain dirty as concurrent write was skipped")
	assert.Equal(t, int64(0), mockHandler.infoCount.Load(), "No info should be logged by the skipped flush")

	// Reset the writing flag and flush again
	cache.writing.Store(false)
	cache.Flush(ctx)
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty after successful flush")
	assert.Equal(t, int64(2), mockHandler.infoCount.Load(), "Info logged by successful flush")
}

func TestSaveDirtyToDB_BeginTx_UnrecoverableError(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.WriteFreq = 0 // Disable periodic writer for manual flush
	mockHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(newtestItemLF("item1"))
	assert.Equal(t, 1, cache.dirty.Size(), "Item should be dirty")

	// Simulate an unrecoverable error on BeginTx
	// The NoOpReaderWriterLF's IsRecoverable doesn't treat "begin unrecoverable" as unrecoverable by default.
	// For this test, we'll rely on the logging difference.
	// A more robust mock could be made for IsRecoverable.
	mockHandler.errorOnNext.Store("begin unrecoverable_db_error") // This error string won't match IsRecoverable's defaults

	cache.Flush(ctx)

	// Check logs: Warn for unrecoverable, Info for recoverable
	assert.Equal(t, int64(1), mockHandler.warnCount.Load(), "Warn should be logged for unrecoverable BeginTx error")
	assert.Equal(t, int64(1), mockHandler.infoCount.Load(), "Info for 'Found N dirty records' still logs")
	assert.Equal(t, 1, cache.dirty.Size(), "Item should remain dirty as BeginTx failed unrecoverably (from cache's perspective of not retrying immediately)")
}

func TestSaveDirtyToDB_RollbackTx_UnrecoverableError(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.WriteFreq = 0
	mockHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(newtestItemLF("itemToFailSave"))
	cache.Save(newtestItemLF("itemToSucceedSaveButFailRollback"))

	// Setup:
	// 1. First item's Save() will cause a recoverable error (e.g., "save deadlock").
	// 2. This triggers a RollbackTx().
	// 3. RollbackTx() will then error unrecoverably (e.g., "rollback unrecoverable_db_error").
	mockHandler.errorOnNext.Store("save deadlock,rollback unrecoverable_db_error")

	cache.Flush(ctx)

	// Logs:
	// - Info for "Recoverable error saving itemToFailSave..."
	// - Warn for "Error rolling back transaction after failures, batch aborted: unrecoverable_db_error"
	assert.Equal(t, int64(1), mockHandler.warnCount.Load(), "One Warn for unrecoverable rollback")
	// Info logs: "Found N dirty", "Recoverable error saving..."
	assert.Equal(t, int64(2), mockHandler.infoCount.Load(), "Two Info logs expected")

	// State:
	// Because rollback failed unrecoverably, the `fail += len(unCommitted)` line runs,
	// and items are NOT re-added to dirty list.
	assert.Equal(t, 1, cache.dirty.Size(), "Items should NOT be re-added to dirty list after unrecoverable rollback")
}
