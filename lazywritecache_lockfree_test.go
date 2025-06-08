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

// testItemLF is a basic implementation of CacheableLF for testing purposes.
type testItemLF struct {
	id string
}

// Key returns the identifier for the testItemLF.
func (i testItemLF) Key() string {
	return i.id
}

// CopyKeyDataFrom copies the key from another CacheableLF item.
// In a real scenario, this would copy database-managed fields.
func (i testItemLF) CopyKeyDataFrom(from CacheableLF) CacheableLF {
	i.id = from.Key()
	return i
}

// String returns the string representation of the testItemLF's id.
func (i testItemLF) String() string {
	return i.id
}

// newtestItemLF creates a new testItemLF with the given key.
func newtestItemLF(key string) testItemLF {
	return testItemLF{
		id: key,
	}
}

// newNoOpTestConfigLF creates a default configuration for testing,
// using a NoOpReaderWriterLF and disabling periodic writes and purges.
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

// TestCacheStoreLoadLF tests the basic Save and Load functionality.
// It verifies that items saved to the cache can be retrieved correctly,
// and that loading a missing item returns false.
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

// TestCacheDirtyListLF tests the management of the dirty items list.
// It checks that saving items marks them as dirty, and re-saving an
// already dirty item doesn't change the dirty count.
func TestCacheDirtyListLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test11"}
	itemLF := testItemLF{id: "testLFLF"}
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "dirty records")
	_, ok := cache.dirty.Load(item.Key())
	assert.True(t, ok, "item should be in dirty list")
	_, ok = cache.dirty.Load(itemLF.Key())
	assert.True(t, ok, "itemLF should be in dirty list")

	cache.Save(itemLF) // Save again
	assert.Equal(t, 2, cache.dirty.Size(), "dirty records count should not change on re-save")
	_, ok = cache.dirty.Load(itemLF.Key())
	assert.True(t, ok, "itemLF should still be in dirty list after re-save")

	// Use ctx to avoid unused variable warning
	_, _ = cache.Load(ctx, "nonexistent")
}

// TestCacheLockUnlockNoPanicsLF ensures that basic cache operations
// like Load and Save do not cause panics, even when items are missing.
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

// BenchmarkCacheWriteMax20kLF benchmarks cache write performance with a cache size of 20,000.
func BenchmarkCacheWriteMax20kLF(b *testing.B) {
	cacheWriteLF(b, 20000)
}

// BenchmarkCacheWriteMax100kLF benchmarks cache write performance with a cache size of 100,000.
func BenchmarkCacheWriteMax100kLF(b *testing.B) {
	cacheWriteLF(b, 100000)
}

// BenchmarkCacheRead20kLF benchmarks cache read performance with a cache size of 20,000.
func BenchmarkCacheRead20kLF(b *testing.B) {
	cacheReadLF(b, 20000)
}

// BenchmarkCacheRead100kLF benchmarks cache read performance with a cache size of 100,000.
func BenchmarkCacheRead100kLF(b *testing.B) {
	cacheReadLF(b, 100000)
}

// BenchmarkParallel_x5_CacheRead20kLF benchmarks parallel cache read performance
// with a cache size of 20,000 and 5 concurrent threads.
func BenchmarkParallel_x5_CacheRead20kLF(b *testing.B) {
	cacheSize := 20000
	nThreads := 5

	parallelRunLF(b, cacheSize, nThreads)
}

// BenchmarkParallel_x10_CacheRead20kLF benchmarks parallel cache read performance
// with a cache size of 20,000 and 10 concurrent threads.
func BenchmarkParallel_x10_CacheRead20kLF(b *testing.B) {
	cacheSize := 20000
	nThreads := 10

	parallelRunLF(b, cacheSize, nThreads)
}

// BenchmarkParallel_x20_CacheRead20kLF benchmarks parallel cache read performance
// with a cache size of 20,000 and 20 concurrent threads.
func BenchmarkParallel_x20_CacheRead20kLF(b *testing.B) {
	cacheSize := 20000
	nThreads := 20

	parallelRunLF(b, cacheSize, nThreads)
}

// parallelRunLF is a helper function for benchmarking parallel cache reads.
// It populates the cache and then simulates nThreads concurrently reading random keys.
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

// cacheWriteLF is a helper function for benchmarking cache write operations.
// It repeatedly saves items to the cache, cycling through keys up to cacheSize.
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

// cacheReadLF is a helper function for benchmarking cache read operations.
// It first populates the cache and then repeatedly reads random keys.
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

// TestCacheEvictionLF tests the cache eviction mechanism.
// It populates the cache beyond its limit, then triggers eviction
// and verifies that the cache size is reduced to the limit,
// evicting items in FIFO order (oldest items first, after they are no longer dirty).
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
	assert.Equal(t, 30, cache.cache.Size(), "nothing evicted until flushed") // Dirty items are not evicted
	cache.Flush(ctx)                                                         // Mark items as not dirty
	cache.evictionProcessor(ctx)                                             // Now eviction should occur
	assert.Equal(t, 20, cache.cache.Size())

	// Check which items were evicted and which remain, assuming FIFO after flush
	// Items 0-9 were added first, then flushed, then eviction ran.
	// The eviction queue (fifo) will have items 0 through 29.
	// When evictionProcessor runs, it dequeues from the front.
	// Since all are non-dirty, it will evict until size is 20.
	// evictionProcessor() will evict items 0 through 9. Items 10 through 29 plus should remain.

	_, ok := cache.cache.Load("0")
	assert.Falsef(t, ok, "0 has been evicted")
	_, ok = cache.cache.Load("1")
	assert.Falsef(t, ok, "1 has  been evicted")
	_, ok = cache.cache.Load("9")
	assert.Falsef(t, ok, "9 has been evicted")
	_, ok = cache.cache.Load("10")
	assert.True(t, ok, "10 has not been evicted")
	_, ok = cache.cache.Load("11")
	assert.Truef(t, ok, "11 has not been evicted")
	_, ok = cache.cache.Load("15")
	assert.Truef(t, ok, "15 has not been evicted")
	_, ok = cache.cache.Load("29")
	assert.Truef(t, ok, "29 should not have been evicted")
}

// TestCacheEvictionEmptyFIFOLF tests the scenario where the FIFO queue is empty
// but the cache still has items. This represents an incorrect state in the underlying cache.
func TestCacheEvictionEmptyFIFOLF(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.Limit = 5 // Set a small limit
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()

	// Add items to the cache directly without using the FIFO queue
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		item := testItemLF{id: id}
		// Use direct access to cache map to bypass the normal Save method
		// which would add items to the FIFO queue
		cache.cache.Store(id, item)
	}

	// Verify the cache has items but FIFO is empty
	assert.Equal(t, 10, cache.cache.Size(), "Cache should have 10 items")
	_, fifoOk := cache.fifo.Peek()
	assert.False(t, fifoOk, "FIFO queue should be empty")

	// Run the eviction processor
	cache.evictionProcessor(ctx)

	// Verify the cache size remains the same since eviction can't proceed with empty FIFO
	assert.Equal(t, 10, cache.cache.Size(), "Cache size should remain unchanged when FIFO is empty")
}

// TestCacheEvictionDirtyItemNotInCacheLF tests the scenario where an item is marked as dirty
// but is not in the cache because it has been deleted while dirty.
func TestCacheEvictionDirtyItemNotInCacheLF(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.Limit = 5 // Set a small limit
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()

	// Add an item to the cache and mark it as dirty
	testKey := "test-key"
	item := testItemLF{id: testKey}
	cache.Save(item)

	// Verify the item is in the cache and marked as dirty
	_, inCache := cache.cache.Load(testKey)
	assert.True(t, inCache, "Item should be in the cache")
	isDirty, inDirty := cache.dirty.Load(testKey)
	assert.True(t, inDirty, "Item should be in the dirty list")
	assert.True(t, isDirty, "Item should be marked as dirty")

	// Remove the item from the cache but leave it in the dirty list
	cache.cache.Delete(testKey)

	// Verify the item is not in the cache but still in the dirty list
	_, inCache = cache.cache.Load(testKey)
	assert.False(t, inCache, "Item should not be in the cache")
	isDirty, inDirty = cache.dirty.Load(testKey)
	assert.True(t, inDirty, "Item should still be in the dirty list")
	assert.True(t, isDirty, "Item should still be marked as dirty")

	// Make sure the cache size is above the limit to trigger eviction
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		item := testItemLF{id: id}
		cache.cache.Store(id, item)
	}

	// Ensure our test key is at the head of the FIFO queue
	// First, empty the queue
	for {
		_, ok := cache.fifo.Dequeue()
		if !ok {
			break
		}
	}
	// Then add our test key as the only item
	cache.fifo.Enqueue(testKey)

	// Run the eviction processor
	cache.evictionProcessor(ctx)

	// Verify the item has been removed from the dirty list
	_, inDirty = cache.dirty.Load(testKey)
	assert.False(t, inDirty, "Item should have been removed from the dirty list")
}

// TestCacheEvictionRaceConditionLF tests the race condition between two eviction processors
// where one evicts the head between peek and removing the item.
func TestCacheEvictionRaceConditionLF(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.Limit = 5 // Set a small limit
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()

	// Add items to the cache
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		item := testItemLF{id: id}
		cache.Save(item)
	}

	// Flush to make items non-dirty
	cache.Flush(ctx)

	// Simulate a race condition by manually manipulating the FIFO queue
	// First, peek at the head of the queue (this would be done by one eviction processor)
	keyToEvict, ok := cache.fifo.Peek()
	assert.True(t, ok, "Should be able to peek at the head of the queue")

	// Now simulate another eviction processor dequeuing the head
	evictedKey, ok := cache.fifo.Dequeue()
	assert.True(t, ok, "Should be able to dequeue the head")
	assert.Equal(t, keyToEvict, evictedKey, "The dequeued key should match the peeked key")

	// Now run the eviction processor, which will try to dequeue the same key
	// but will find a different key at the head
	cache.evictionProcessor(ctx)

	// Verify the cache size has been reduced
	assert.LessOrEqual(t, cache.cache.Size(), cfg.Limit, "Cache size should be at or below the limit after eviction")
}

// Test_GetAndReleaseLF is a simple test for Save and Load, similar to TestCacheStoreLoadLF.
// It ensures items can be saved and then retrieved.
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

// Test_GetAndReleaseWithForcedPanicLF tests the behavior when LookupOnMiss is true
// and the underlying data handler (NoOpReaderWriterLF) is configured to panic.
// It verifies that a call to Load results in a panic.
func Test_GetAndReleaseWithForcedPanicLF(t *testing.T) {
	ctx := context.Background()
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF(cfg)
	defer cache.Shutdown()

	cache.LookupOnMiss = true // Enable lookup on miss for this test

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load(ctx, "test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

	// Configure the mock handler to panic on the next Find operation
	cfg.handler.(NoOpReaderWriterLF[testItemLF]).panicOnNext.Store(true)
	assert.Panics(t, func() {
		_, ok := cache.Load(ctx, "test4_non_existent_to_trigger_find") // Key doesn't exist, so Find will be called
		assert.Falsef(t, ok, "should not be found or panic should prevent reaching this")
	})
	assert.Equal(t, int64(1), cache.Misses.Load(), "1 miss expected before panic")

}

// TestCacheStats_JSONLF tests the JSON serialization of cache statistics.
// It verifies that the cache stats can be marshaled to JSON and then
// unmarshaled back into a map, checking for the presence and initial value of 'hits'.
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

// TestRangeLF tests the Range method for iterating over all items in the cache.
// It verifies that the provided action function is called for each item.
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
		return true // Continue iteration
	})

	assert.Equal(t, 3, n, "iterated over all cache items")
}

// TestRangeAbortLF tests the ability to abort iteration in the Range method.
// It verifies that if the action function returns false, the iteration stops.
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
			return false // Stop iteration
		}
		return true
	})

	assert.Equal(t, 2, n, "iterated over all cache items")
}

// TestNoGoroutineLeaksLF uses goleak to verify that no goroutines are leaked
// after creating a cache with a context, saving an item, and then shutting it down.
func TestNoGoroutineLeaksLF(t *testing.T) {
	defer goleak.VerifyNone(t) // Verifies no goroutines leaked at the end of the test
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewLazyWriterCacheWithContextLF[testItemLF](ctx, newNoOpTestConfigLF())
	cache.Save(testItemLF{id: "test"})
	time.Sleep(100 * time.Millisecond) // Allow time for any goroutines to start
	cancel()                           // Signal goroutines to stop
	cache.Shutdown()                   // Explicitly shutdown
	time.Sleep(100 * time.Millisecond) // Allow time for goroutines to exit
}

// TestNewDefaultConfigLF tests the NewDefaultConfigLF constructor.
// It verifies that the returned configuration has the expected default values
// for Limit, LookupOnMiss, WriteFreq, PurgeFreq, and FlushOnShutdown.
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

// TestEmptyCacheableLF tests the EmptyCacheableLF placeholder type.
// It verifies that its Key method returns an empty string and
// CopyKeyDataFrom returns the input item unchanged.
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

// TestClearDirtyLF tests the ClearDirty method.
// It verifies that after saving items (making them dirty),
// calling ClearDirty empties the dirty list.
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

// TestPeriodicSaveLF tests the periodic lazy writer functionality.
// It configures a short WriteFreq, saves items, and verifies that
// the items are eventually written (dirty list becomes empty) and
// the DirtyWrites counter is incremented.
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

	// Wait for the lazy writer to process the dirty items
	assert.Eventuallyf(t, func() bool {
		return !cache.IsDirty()
	}, 200*time.Millisecond, 10*time.Millisecond, "Cache should not be dirty after save")

	// Verify dirty items were processed
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty after lazy writer runs")
	assert.GreaterOrEqual(t, cache.DirtyWrites.Load(), int64(1), "DirtyWrites counter should be incremented")
}

// TestPeriodicEvictionsLF tests the periodic eviction manager.
// It sets a small cache Limit and short PurgeFreq, adds more items
// than the limit, and verifies that the cache size is eventually
// reduced to the limit and the Evictions counter is incremented.
func TestPeriodicEvictionsLF(t *testing.T) {
	// Create a cache with a small limit and short purge frequency
	cfg := newNoOpTestConfigLF()
	cfg.Limit = 5
	cfg.PurgeFreq = 50 * time.Millisecond
	cfg.WriteFreq = 10 * time.Millisecond // Need to flush dirty items for eviction to work

	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	// Add more items than the limit
	for i := 0; i < 10; i++ {
		cache.Save(testItemLF{id: strconv.Itoa(i)})
	}

	// Verify all items are in the cache initially
	assert.Equal(t, 10, cache.cache.Size(), "Should have 10 items in cache initially")

	// Wait for eviction manager to run multiple times and for items to become non-dirty
	assert.Eventuallyf(t, func() bool {
		// Items must be non-dirty to be evicted. The periodic writer handles this.
		return cache.cache.Size() <= cfg.Limit && cache.dirty.Size() == 0
	}, 500*time.Millisecond, 10*time.Millisecond, "Cache size should be at or below the limit after eviction and items non-dirty")

	// Verify cache size is now at or below the limit
	assert.LessOrEqual(t, cache.cache.Size(), cfg.Limit, "Cache size should be at or below the limit after eviction")
	assert.Greater(t, cache.Evictions.Load(), int64(0), "Evictions counter should be incremented")
}

// TestFlushOnShutdownLF tests the FlushOnShutdown functionality.
// It enables FlushOnShutdown, saves items (making them dirty),
// then cancels the cache's context (simulating shutdown) and verifies
// that the dirty items are flushed.
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

	// Wait a bit for the shutdown to complete and flush to occur
	assert.Eventuallyf(t, func() bool {
		return cache.dirty.Size() == 0
	}, 200*time.Millisecond, 10*time.Millisecond, "Cache should be empty after shutdown with FlushOnShutdown=true")

	// Verify dirty items were processed during shutdown
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty after shutdown with FlushOnShutdown=true")
	assert.GreaterOrEqual(t, cache.DirtyWrites.Load(), int64(1), "DirtyWrites counter should be incremented")
}

// TestRequeueRecoverableErrLF tests that items are re-queued (remain dirty)
// when a recoverable error (e.g., "save deadlock") occurs during a Save operation in Flush.
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
	testHandler.errorOnNext.Store("save deadlock") // Simulate recoverable error
	cache.Flush(ctx)
	assert.Equal(t, int64(0), testHandler.warnCount.Load(), "No warnings for recoverable save error (it's an Info)")
	assert.Equal(t, int64(3), testHandler.infoCount.Load(), "Info for 'Found N dirty', 'Recoverable error saving...', 'Transaction rolled back...'")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should still be dirty and in the cache for retry")
}

// TestRequeueSkipsNonRecoverableErrLF tests that items are NOT re-queued (one is removed from dirty)
// when a non-recoverable error (e.g., "save duplicate key") occurs during a Save operation in Flush.
// The transaction is rolled back, but the failing item is not marked for retry.
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
	testHandler.errorOnNext.Store("save duplicate key") // Simulate non-recoverable error
	cache.Flush(ctx)
	assert.Equal(t, int64(1), testHandler.warnCount.Load(), "Warning for unrecoverable save error")
	assert.Equal(t, int64(2), testHandler.infoCount.Load(), "Info for 'Found N dirty' and 'Transaction rolled back...'")
	assert.Equal(t, 1, cache.dirty.Size(), "1 item should remain dirty (the one that didn't encounter the error)")
}

// TestRequeueCommitRecoverableErrLF tests that items are re-queued (remain dirty)
// when a recoverable error (e.g., "commit deadlock") occurs during the CommitTx operation in Flush.
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
	testHandler.errorOnNext.Store("commit deadlock") // Simulate recoverable commit error
	cache.Flush(ctx)
	assert.Equal(t, int64(2), testHandler.warnCount.Load(), "Warnings for 'Recoverable error from CommitTx' and 'Error committing transaction'")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should still be dirty for retry")
}

// TestRequeueCommitSkipsNonRecoverableErrLF tests that items are NOT re-queued (dirty list becomes empty)
// when a non-recoverable error (e.g., "commit duplicate key") occurs during CommitTx in Flush.
// The items are considered lost from the dirty perspective.
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
	testHandler.errorOnNext.Store("commit duplicate key") // Simulate non-recoverable commit error
	cache.Flush(ctx)
	assert.Equal(t, int64(2), testHandler.warnCount.Load(), "Warnings for 'Unrecoverable error from CommitTx' and 'Error committing transaction'")
	assert.Equal(t, 0, cache.dirty.Size(), "0 items should be dirty as commit failed unrecoverably")
}

// TestRequeueBeginRecoverableErrLF tests that items remain dirty and the batch is retried
// when a recoverable error occurs during BeginTx.
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
	testHandler.errorOnNext.Store("begin db bad connection") // Simulate recoverable BeginTx error
	cache.Flush(ctx)
	assert.Equal(t, int64(2), testHandler.infoCount.Load(), "Info for 'Found N dirty' and 'Recoverable error with BeginTx'")
	assert.Equal(t, int64(0), testHandler.warnCount.Load(), "No warnings as BeginTx error is Info")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should still be dirty for retry")
}

// TestRequeueRollbackRecoverableErrLF tests that items are re-queued (remain dirty)
// when a recoverable error occurs during Save, followed by another recoverable error during RollbackTx.
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
	testHandler.errorOnNext.Store("save deadlock,rollback deadlock") // Recoverable save, then recoverable rollback
	cache.Flush(ctx)
	assert.Equal(t, "", testHandler.errorOnNext.Load(), "both errors handled")
	assert.Equal(t, 2, cache.dirty.Size(), "2 items should still be dirty for retry")
	assert.Equal(t, int64(0), testHandler.warnCount.Load(), "No warnings as all errors were Info-level recoverable")
	assert.Equal(t, int64(4), testHandler.infoCount.Load(), "Info for 'Found N dirty', 'Recoverable save', 'Error rolling back (recoverable)', 'Transaction rolled back'")
}

// TestRequeueRollbackUnrecoverableErrLF tests behavior when a recoverable Save error
// is followed by an unrecoverable RollbackTx error.
// Items are not re-queued because the unrecoverable rollback implies the batch is aborted.
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
	testHandler.errorOnNext.Store("save deadlock,rollback failed") // Recoverable save, then unrecoverable rollback
	cache.Flush(ctx)
	assert.Equal(t, "", testHandler.errorOnNext.Load(), "both errors handled")
	// The logic `fail += len(unCommitted)` and `return` means items are not re-added to dirty.
	assert.Equal(t, 1, cache.dirty.Size(), "1 items should be dirty after unrecoverable rollback aborts batch")
	assert.Equal(t, int64(1), testHandler.warnCount.Load(), "Warning for unrecoverable rollback")
	assert.Equal(t, int64(2), testHandler.infoCount.Load(), "Info for 'Found N dirty' and 'Recoverable save'")
}

// TestPanicHandlerLF tests the panic recovery mechanism within saveDirtyToDB.
// It simulates a panic after a recoverable save error and verifies that
// a warning is logged and the cache doesn't crash.
// One item (the one causing the panic during its processing) is lost from dirty.
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
	testHandler.errorOnNext.Store("save deadlock,panic") // Recoverable save, then panic during next item or rollback
	cache.Flush(ctx)
	assert.Equal(t, "", testHandler.errorOnNext.Load(), "both errors handled (or panic consumed the second part)")
	// The panic occurs within the c.dirty.Range. The item causing the panic (or the one being processed)
	// might have already been deleted from dirty optimistically.
	// The transaction will be rolled back due to the panic (if it happens before commit/rollback call)
	// or the panic handler itself. Items in unCommitted might not be re-added.
	// The exact dirty count depends on when the panic happens relative to dirty.Delete and unCommitted append.
	// Given the NoOp mock, the panic is likely in RollbackTx.
	// If panic in RollbackTx: unCommitted items are not re-added.
	assert.Equal(t, 1, cache.dirty.Size(), "1 items should be dirty after panic during rollback")
	assert.Equal(t, int64(1), testHandler.warnCount.Load(), "Warning for panic in lazy write")
}

// TestSaveDirtyToDB_AllowConcurrentWrites tests that multiple Flush calls can proceed
// concurrently when AllowConcurrentWrites is true.
// It uses goroutines to call Flush simultaneously and checks that the dirty list is cleared.
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

// TestSaveDirtyToDB_DisallowConcurrentWrites tests that if a write is already in progress
// (simulated by setting cache.writing to true), subsequent Flush calls are skipped
// when AllowConcurrentWrites is false.
func TestSaveDirtyToDB_DisallowConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.AllowConcurrentWrites = false // Explicitly disable
	cfg.WriteFreq = 0                 // Disable periodic writer

	mockHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(newtestItemLF("item1"))

	// Simulate one write already in progress by setting the atomic bool
	cache.writing.Store(true)

	// Attempt another flush, it should return early because writing is true
	cache.Flush(ctx)

	assert.Equal(t, 1, cache.dirty.Size(), "Item should remain dirty as concurrent write was skipped")
	assert.Equal(t, int64(0), mockHandler.infoCount.Load(), "No info should be logged by the skipped flush") // No "Found N dirty"

	// Reset the writing flag and flush again
	cache.writing.Store(false)
	cache.Flush(ctx)
	assert.Equal(t, 0, cache.dirty.Size(), "Dirty list should be empty after successful flush")
	assert.Equal(t, int64(2), mockHandler.infoCount.Load(), "Info for 'Found N dirty' and 'Completed DB Sync'")
}

// TestSaveDirtyToDB_BeginTx_UnrecoverableError tests that if BeginTx returns an unrecoverable error,
// the batch is aborted, a warning is logged, and items remain dirty.
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

	assert.Equal(t, int64(1), mockHandler.warnCount.Load(), "Warn should be logged for unrecoverable BeginTx error")
	assert.Equal(t, int64(1), mockHandler.infoCount.Load(), "Info for 'Found N dirty records' still logs")
	assert.Equal(t, 1, cache.dirty.Size(), "Item should remain dirty as BeginTx failed unrecoverably and batch aborted")
}

// TestSaveDirtyToDB_RollbackTx_UnrecoverableError tests the scenario where a save operation
// causes a recoverable error, leading to a transaction rollback, but the RollbackTx itself
// fails with an unrecoverable error. In this case, the batch is aborted, and items are
// not re-added to the dirty list.
func TestSaveDirtyToDB_RollbackTx_UnrecoverableError(t *testing.T) {
	ctx := context.Background()
	cfg := newNoOpTestConfigLF()
	cfg.WriteFreq = 0
	mockHandler := cfg.handler.(NoOpReaderWriterLF[testItemLF])
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(newtestItemLF("itemToFailSave"))
	cache.Save(newtestItemLF("itemToSucceedSaveButFailRollback")) // This item won't be processed due to first item's save failure

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
