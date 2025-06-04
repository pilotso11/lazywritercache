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

package lockfree

import (
	"context"
	"encoding/json"
	"errors" // Added for SetErrorOnSave etc.
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/puzpuzpuz/xsync"
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

func newNoOpTestConfigLF(panics ...bool) (ConfigLF[testItemLF], *NoOpReaderWriterLF[testItemLF]) {
	doPanics := len(panics) > 0 && panics[0]
	// Ensure newtestItemLF matches the expected func(key string) T signature
	readerWriter := NewNoOpReaderWriterLF[testItemLF](newtestItemLF, doPanics)
	cfg := ConfigLF[testItemLF]{
		handler:      readerWriter,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
	return cfg, readerWriter
}

func TestCacheStoreLoadLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load("test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

	item4, ok := cache.Load("testLF")
	assert.Truef(t, ok, "loaded testLF")
	assert.Equal(t, itemLF, item4)

	_, ok = cache.Load("missing")
	assert.Falsef(t, ok, "not loaded missing")

}

func TestCacheDirtyListLF(t *testing.T) {
	item := testItemLF{id: "test11"}
	itemLF := testItemLF{id: "testLFLF"}
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "dirty records")
	assert.True(t, findIn(cache.dirty, item))
	assert.True(t, findIn(cache.dirty, itemLF))

	cache.Save(itemLF)
	assert.Equal(t, 2, cache.dirty.Size(), "dirty records")
	assert.True(t, findIn(cache.dirty, itemLF))
}

func findIn(dirty *xsync.MapOf[string, bool], item testItemLF) (found bool) {
	found = false
	dirty.Range(func(k string, _ bool) bool {
		if k == item.Key() {
			found = true
			return false // exit the loop
		}
		return true
	})
	return found
}

func TestCacheLockUnlockNoPanicsLF(t *testing.T) {
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	assert.NotPanics(t, func() {
		cache.Load("missing")
	}, "get and Unlock")

	assert.NotPanics(t, func() {
		item := testItemLF{id: "test"}
		cache.Load("missing")
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

	parallelRun(b, cacheSize, nThreads)
}

func BenchmarkParallel_x10_CacheRead20kLF(b *testing.B) {
	cacheSize := 20000
	nThreads := 10

	parallelRun(b, cacheSize, nThreads)
}

func parallelRun(b *testing.B, cacheSize int, nThreads int) {
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	var keys []string
	for i := 0; i < cacheSize; i++ {
		id := strconv.Itoa(i % cacheSize)
		keys = append(keys, id)
		item := testItemLF{id: id}
		cache.Save(item)
	}

	wait := sync.WaitGroup{}
	for i := 0; i < nThreads; i++ {
		wait.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				key := rand.Intn(cacheSize)
				_, ok := cache.Load(keys[key])
				if ok {
				}
			}
			wait.Add(-1)
		}()
	}
	wait.Wait()
	b.ReportAllocs()
}

func cacheWriteLF(b *testing.B, cacheSize int) {
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
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
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
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
		_, ok := cache.Load(keys[key])
		if ok {
			k++
		}
	}
	assert.Truef(b, k > 0, "critical failure")
	b.ReportAllocs()
}

func TestCacheEvictionLF(t *testing.T) {

	cfg, _ := newNoOpTestConfigLF()
	cfg.Limit = 20
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	for i := 0; i < 30; i++ {
		id := strconv.Itoa(i)
		item := testItemLF{id: id}
		cache.Save(item)
	}
	assert.Equal(t, 30, cache.cache.Size())
	cache.evictionProcessor()
	assert.Equal(t, 30, cache.cache.Size(), "nothing evicted until flushed")
	cache.Flush()
	cache.evictionProcessor()
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

func TestGormLazyCache_GetAndReleaseLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load("test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

}

func TestGormLazyCache_GetAndReleaseWithForcedPanicLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg, handler := newNoOpTestConfigLF(false) // forcePanics via constructor is now just for default panicOnLoad/Write
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	handler.SetPanicOnLoad(true) // Explicitly set panic for this test
	cache.LookupOnMiss = true

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load("test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

	assert.Panics(t, func() {
		_, ok := cache.Load("test4")
		assert.Falsef(t, ok, "should not be found")
	})
	assert.Equal(t, int64(1), cache.Misses.Load(), "1 miss expected")

}

func TestCacheStats_JSONLF(t *testing.T) {
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
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
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
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
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
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
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheWithContextLF[testItemLF](ctx, cfg)
	cache.Save(testItemLF{id: "test"})
	time.Sleep(100 * time.Millisecond)
	cancel()
	cache.Shutdown()
	time.Sleep(300 * time.Millisecond)

}

func TestCacheLF_SaveDirtyToDB_PanicRecovery(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	panickingItemKey := "panicItemLF"
	normalItemKey := "normalItemLF"

	// Configure Save to panic only for panickingItemKey
	handler.SetPanicOnSaveKey(panickingItemKey)
	handler.SetPanicOnWrite(false) // Ensure global panic is off

	// Add items
	cache.Save(newtestItemLF(panickingItemKey))
	cache.Save(newtestItemLF(normalItemKey))

	_, isDirtyPanic := cache.dirty.Load(panickingItemKey)
	assert.True(t, isDirtyPanic, "Panicking item should be dirty before flush")
	_, isDirtyNormal := cache.dirty.Load(normalItemKey)
	assert.True(t, isDirtyNormal, "Normal item should be dirty before flush")

	assert.NotPanics(t, func() {
		cache.Flush() // Flush calls saveDirtyToDB
	}, "Flush should not propagate panic from Save")

	// Assert that Warn method was called with the panic message
	handler.mu.Lock()
	assert.True(t, handler.WarnCallCount.Load() > 0, "Warn should have been called")
	foundPanicMessage := false
	expectedPanicText := "recovered from panic during saveDirtyToDB: test panic, write on specific key: " + panickingItemKey
	for _, msg := range handler.WarnMessages {
		if msg == expectedPanicText {
			foundPanicMessage = true
			break
		}
	}
	assert.True(t, foundPanicMessage, "Expected panic message ('%s') not found in Warn messages. Got: %v", expectedPanicText, handler.WarnMessages)
	handler.mu.Unlock()

	// Assert panicking item remains dirty
	_, isDirtyPanicAfterFlush := cache.dirty.Load(panickingItemKey)
	assert.True(t, isDirtyPanicAfterFlush, "Panicking item should remain dirty")

	// Assert normalItemKey status (depends on processing order relative to panic)
	_, isDirtyNormalAfterFlush := cache.dirty.Load(normalItemKey)
	if handler.GetSaveCount(normalItemKey) > 0 {
		assert.False(t, isDirtyNormalAfterFlush, "Normal item should be clean if it was saved")
	} else {
		assert.True(t, isDirtyNormalAfterFlush, "Normal item should be dirty if it was not saved (due to panic before it)")
	}

	// Reset panic flags for other tests
	handler.SetPanicOnSaveKey("")
	handler.SetPanicOnWrite(false)
	handler.ResetCountersAndMessages()
}

func TestCacheLF_LookupOnMiss_False(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cfg.LookupOnMiss = false // Explicitly set for clarity

	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	itemKey := "lookupMissItemLF"

	// Configure handler to succeed on find IF it were called
	handler.SetSucceedOnFind(itemKey, true)
	handler.SetMockedItem(itemKey, newtestItemLF(itemKey))
	handler.FindCallCount.Store(0) // Reset to be sure

	_, found := cache.Load(itemKey) // Load is the equivalent of GetAndRelease for LF cache

	assert.False(t, found, "Item should not be found when LookupOnMiss is false")
	assert.Equal(t, int64(1), cache.Misses.Load(), "Cache Misses count should be 1")
	assert.Equal(t, int64(0), handler.FindCallCount.Load(), "Handler's Find method should not have been called")

	handler.ResetCountersAndMessages()
}

func TestCacheLF_ClearDirty(t *testing.T) {
	cfg, _ := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	item1Key := "itemToClearLF1"
	item2Key := "itemToClearLF2"

	cache.Save(newtestItemLF(item1Key))
	cache.Save(newtestItemLF(item2Key))

	assert.Equal(t, 2, cache.dirty.Size(), "Should have 2 dirty items initially")

	cache.ClearDirty()

	assert.Equal(t, 0, cache.dirty.Size(), "Dirty map should be empty after ClearDirty")

	// Verify items are still in cache if ClearDirty only affects the dirty map
	_, found1 := cache.cache.Load(item1Key)
	_, found2 := cache.cache.Load(item2Key)
	assert.True(t, found1, "Item1 should still be in cache")
	assert.True(t, found2, "Item2 should still be in cache")
}

func TestCacheLF_LazyWriter_FlushOnShutdown(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cfg.WriteFreq = 10 * time.Millisecond // Shortened for faster eventual check
	cfg.FlushOnShutdown = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled

	cache := NewLazyWriterCacheWithContextLF[testItemLF](ctx, cfg)
	// No immediate defer cache.Shutdown() as we want to call it explicitly

	itemKey := "flushOnShutdownItemLF"
	cache.Save(newtestItemLF(itemKey))

	_, isDirty := cache.dirty.Load(itemKey)
	assert.True(t, isDirty, "Item should be dirty before shutdown")

	// Give a moment for the lazy writer to potentially start a write cycle (though not strictly necessary for this test's core logic)
	time.Sleep(cfg.WriteFreq / 2)

	cache.Shutdown() // This should trigger Flush if FlushOnShutdown is true

	// Use assert.Eventually to check for the save
	assert.Eventually(t, func() bool {
		return handler.GetSaveCount(itemKey) == 1
	}, 2*time.Second, 50*time.Millisecond, "Save should have been called for the item during shutdown")

	assert.Eventually(t, func() bool {
		_, isDirty := cache.dirty.Load(itemKey)
		return !isDirty
	}, 2*time.Second, 50*time.Millisecond, "Item should not be dirty after FlushOnShutdown")

	handler.ResetCountersAndMessages()
}

func TestCacheLF_LazyWriter_SaveDirtyErrorInLoop(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cfg.WriteFreq = 10 * time.Millisecond // Frequent writes

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewLazyWriterCacheWithContextLF[testItemLF](ctx, cfg)
	// defer cache.Shutdown() // Explicit shutdown at end

	itemKey := "saveErrorItemLF"
	expectedError := errors.New("save failed LF intentionally")

	handler.SetErrorOnSave(itemKey, expectedError)

	cache.Save(newtestItemLF(itemKey))

	// Use assert.Eventually to check for warnings and save attempts
	expectedMessagePrefix := "Error saving " + itemKey + " to DB" // Message from saveDirtyToDB in lockfree
	assert.Eventually(t, func() bool {
		handler.mu.Lock()
		defer handler.mu.Unlock()
		if handler.WarnCallCount.Load() == 0 {
			return false
		}
		for _, msg := range handler.WarnMessages {
			// Example actual log: "Error saving item_save_error_lf to DB: save failed intentionally"
			if strings.HasPrefix(msg, expectedMessagePrefix) && strings.HasSuffix(msg, expectedError.Error()) {
				return true
			}
		}
		return false
	}, 5*time.Second, 50*time.Millisecond, "Expected save error message not found in Warn messages. Got: %v", handler.WarnMessages)

	assert.Eventually(t, func() bool {
		return handler.GetSaveCount(itemKey) >= 1
	}, 5*time.Second, 50*time.Millisecond, "Save should have been attempted at least once")

	// Assert item is still dirty
	_, itemIsStillDirty := cache.dirty.Load(itemKey)
	assert.True(t, itemIsStillDirty, "Item should still be dirty after save error in loop")

	handler.ResetCountersAndMessages()
	handler.SetErrorOnSave(itemKey, nil) // cleanup
	cache.Shutdown()                     // Explicitly shutdown
}

func TestCacheLF_SaveDirtyToDB_HandlerFindError(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	itemKey := "item_find_error_lf"
	itemToSave := newtestItemLF(itemKey)
	cache.Save(itemToSave) // Item is now in cache and dirty

	// Configure handler.Find to return an error for this item
	// Default behavior of NoOpReaderWriterLF.Find is to return "NoOp, item not found"
	// unless SetSucceedOnFind(key, true) is called. So, no explicit SetSucceedOnFind(key, false) is needed here.
	// We can also explicitly set an error using SetErrorOnFind if that method existed, but it doesn't for NoOp.

	// Ensure it's marked dirty
	_, isDirty := cache.dirty.Load(itemKey)
	assert.True(t, isDirty)

	// Reset counters before flush
	handler.ResetCountersAndMessages()

	cache.Flush()

	// saveDirtyToDB in lockfree calls Find then Save.
	// If Find returns an error, the current item from cache is used.
	// The item = item.CopyKeyDataFrom(old).(T) line will use 'old' which is the zero value of T
	// if Find fails and returns a zero T.
	// This means the item passed to Save might have its key field, but other fields might be zeroed.
	// The crucial part is that Save IS called.
	assert.Equal(t, int64(1), handler.FindCallCount.Load(), "Find should have been called once for the item")
	assert.Equal(t, int64(1), handler.GetSaveCount(itemKey), "Save should still be called once for the item")

	// Assert no specific warning for find error in this path
	handler.mu.Lock()
	assert.Equal(t, int64(0), handler.WarnCallCount.Load(), "Warn should not have been called for a find error in this specific path")
	handler.mu.Unlock()

	// Item should be cleared from dirty list if Save succeeds (which it does by default in NoOp)
	_, isStillDirty := cache.dirty.Load(itemKey)
	assert.False(t, isStillDirty, "Item should be cleared from dirty list if Save was successful")

	handler.ResetCountersAndMessages()
}

func TestCacheLF_SaveDirtyToDB_HandlerSaveError(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	itemKey := "item_save_error_lf"
	itemToSave := newtestItemLF(itemKey)
	cache.Save(itemToSave) // Item is now in cache and dirty

	expectedError := errors.New("save failed intentionally")
	handler.SetErrorOnSave(itemKey, expectedError)

	// Ensure it's marked dirty
	_, isDirty := cache.dirty.Load(itemKey)
	assert.True(t, isDirty)

	handler.ResetCountersAndMessages() // Reset before Flush to check calls during Flush

	cache.Flush()

	// Assert Save was called
	assert.Equal(t, int64(1), handler.GetSaveCount(itemKey), "Save should have been called once for the item")

	// Assert Warn was called with the specific error message
	handler.mu.Lock()
	assert.True(t, handler.WarnCallCount.Load() > 0, "Warn should have been called")
	foundErrorMessage := false
	// Expected message: "Error saving item_save_error_lf to DB: save failed intentionally"
	expectedLogMessage := "Error saving " + itemKey + " to DB: " + expectedError.Error()
	for _, msg := range handler.WarnMessages {
		if msg == expectedLogMessage {
			foundErrorMessage = true
			break
		}
	}
	assert.True(t, foundErrorMessage, "Expected save error message not found in Warn messages. Got: %v", handler.WarnMessages)
	handler.mu.Unlock()

	// Item should remain dirty
	_, isStillDirty := cache.dirty.Load(itemKey)
	assert.True(t, isStillDirty, "Item should remain dirty after Save failure")

	handler.ResetCountersAndMessages()
	handler.SetErrorOnSave(itemKey, nil) // cleanup
}

func TestCacheLF_SaveDirtyToDB_PurgedItemUpdate(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	itemKey := "purgedItemLF"
	itemToSave := newtestItemLF(itemKey)
	cache.Save(itemToSave) // Item is now in cache and dirty

	// Configure handler for successful Find and Save initially
	handler.SetSucceedOnFind(itemKey, true)
	handler.SetMockedItem(itemKey, itemToSave) // So Find returns the correct item

	// Setup PostSaveCallback to delete the item from cache *after* DB save,
	// but *before* the cache updates its internal state for that item post-save.
	handler.SetPostSaveCallback(func(key string) {
		if key == itemKey {
			cache.cache.Delete(itemKey) // Simulate item being purged/evicted
			// cache.fifo.Remove(itemKey) // .Remove is not available on lockfreequeue, and not essential for this test's core logic
		}
	})

	handler.ResetCountersAndMessages() // Reset before Flush

	cache.Flush() // This will trigger saveDirtyToDB

	// Assert that Warn method was called with the specific message
	handler.mu.Lock()
	assert.True(t, handler.WarnCallCount.Load() > 0, "Warn should have been called")
	foundPurgedMessage := false
	expectedMessage := "Deferred update attempted on purged cache item: " + itemKey
	for _, msg := range handler.WarnMessages {
		if msg == expectedMessage {
			foundPurgedMessage = true
			break
		}
	}
	assert.True(t, foundPurgedMessage, "Expected purged item message not found in Warn messages. Got: %v", handler.WarnMessages)
	handler.mu.Unlock()

	// Item should be cleared from dirty list as Save itself succeeded
	_, isStillDirty := cache.dirty.Load(itemKey)
	assert.False(t, isStillDirty, "Item should be cleared from dirty list as Save itself succeeded")

	// The item should not be in the main cache storage because PostSaveCallback removed it.
	_, inCache := cache.cache.Load(itemKey)
	assert.False(t, inCache, "Item should not be in the cache storage after being purged by callback")

	handler.ResetCountersAndMessages()
	handler.SetPostSaveCallback(nil) // Clean up
}

func TestCacheLF_SaveDirtyToDB_BeginTxError(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	itemKey := "beginTxFailItemLF"
	expectedError := errors.New("begin tx failed LF")

	handler.SetBeginTxError(expectedError)

	cache.Save(newtestItemLF(itemKey)) // Item is now dirty

	// Ensure it's marked dirty
	_, isDirty := cache.dirty.Load(itemKey)
	assert.True(t, isDirty)

	handler.ResetCountersAndMessages() // Reset before Flush

	cache.Flush() // saveDirtyToDB is called by Flush

	// In the lock-free version, saveDirtyToDB logs the BeginTx error but does not return it up the Flush call chain.
	// Assert that Warn method was called with the BeginTx error message
	handler.mu.Lock()
	assert.True(t, handler.WarnCallCount.Load() > 0, "Warn should have been called for BeginTx error")
	foundErrorMessage := false
	// Expected message: "Error beginning transaction: %v", err
	expectedLogMessage := "Error beginning transaction: " + expectedError.Error()
	for _, msg := range handler.WarnMessages {
		if msg == expectedLogMessage {
			foundErrorMessage = true
			break
		}
	}
	assert.True(t, foundErrorMessage, "Expected BeginTx error message not found in Warn messages. Got: %v", handler.WarnMessages)
	handler.mu.Unlock()

	// Item should remain dirty
	_, isStillDirty := cache.dirty.Load(itemKey)
	assert.True(t, isStillDirty, "Item should remain dirty after BeginTx failure")

	// Save should not have been called
	assert.Equal(t, int64(0), handler.GetSaveCount(itemKey), "Save should not have been called")

	handler.ResetCountersAndMessages()
	handler.SetBeginTxError(nil) // cleanup
}

func TestCacheLF_SaveDirtyToDB_CommitTxPanic(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	itemKey := "commitTxPanicItemLF"

	handler.SetCommitTxPanic(true)

	cache.Save(newtestItemLF(itemKey)) // Item is now dirty
	_, isDirty := cache.dirty.Load(itemKey)
	assert.True(t, isDirty)

	handler.ResetCountersAndMessages() // Reset before Flush

	assert.NotPanics(t, func() {
		cache.Flush() // saveDirtyToDB is called by Flush
	}, "Flush should not propagate panic from CommitTx")

	// Assert that Warn method was called with the commit panic message
	handler.mu.Lock()
	assert.True(t, handler.WarnCallCount.Load() > 0, "Warn should have been called")
	foundPanicMessage := false
	// Expected message: "recovered from panic during CommitTx: %v", panicVal
	// Panic message from NoOpReaderWriterLF is "mock CommitTx panic"
	expectedLogMessage := "recovered from panic during CommitTx: mock CommitTx panic"
	for _, msg := range handler.WarnMessages {
		if msg == expectedLogMessage {
			foundPanicMessage = true
			break
		}
	}
	assert.True(t, foundPanicMessage, "Expected commit panic message not found in Warn messages. Got: %v", handler.WarnMessages)
	handler.mu.Unlock()

	// Item should remain dirty because CommitTx failed
	_, isStillDirty := cache.dirty.Load(itemKey)
	assert.True(t, isStillDirty, "Item should remain dirty after CommitTx panic")

	// Save should have been called once, but commit failed
	assert.Equal(t, int64(1), handler.GetSaveCount(itemKey), "Save should have been called once")

	handler.ResetCountersAndMessages()
	handler.SetCommitTxPanic(false) // cleanup
}

func TestCacheLF_EvictionProcessor_ReEnqueueDirty(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cfg.Limit = 1     // Set limit to 1 to force eviction consideration
	cfg.PurgeFreq = 0 // Disable auto-purging for manual test control, evictionProcessor called manually
	cfg.WriteFreq = 0 // Disable auto-writing for manual test control

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled

	cache := NewLazyWriterCacheWithContextLF[testItemLF](ctx, cfg)
	defer cache.Shutdown() // Still good practice

	item1Key := "item1_lf_evict"
	item2Key := "item2_lf_evict"

	// Add two items. Both will be dirty.
	cache.Save(newtestItemLF(item1Key)) // item1 is older, should be at head of FIFO
	cache.Save(newtestItemLF(item2Key))

	_, item1Dirty := cache.dirty.Load(item1Key)
	_, item2Dirty := cache.dirty.Load(item2Key)
	assert.True(t, item1Dirty, "item1 should be dirty")
	assert.True(t, item2Dirty, "item2 should be dirty")
	// assert.Equal(t, int32(2), cache.fifo.Len(), "FIFO queue should have 2 items") // Len() doesn't exist

	// Manually call evictionProcessor
	cache.evictionProcessor()

	// Assert that Warn method was called with "Dirty items at the top of the purge queue"
	handler.mu.Lock()
	assert.True(t, handler.WarnCallCount.Load() > 0, "Warn should have been called")
	foundSkipMessage := false
	expectedMessage := "Dirty items at the top of the purge queue, increase flush rate or db performance"
	for _, msg := range handler.WarnMessages {
		if msg == expectedMessage {
			foundSkipMessage = true
			break
		}
	}
	assert.True(t, foundSkipMessage, "Expected skip dirty message not found in Warn messages. Got: %v", handler.WarnMessages)
	handler.mu.Unlock()

	// Assert that cache size is still 2 (nothing was evicted)
	assert.Equal(t, 2, cache.cache.Size(), "Cache size should still be 2")

	// Assert FIFO still contains 2 items (by successfully dequeueing them), and item1 was re-enqueued (now at the end)
	// No direct Len() method, so we infer by successful Dequeue operations.

	// Verify item1 is now at the end of the queue
	// Dequeue first item (should be item2), then second (should be item1)
	// Dequeue returns only one value (the item). If queue is empty, it's a zero value for T (string here).
	firstDequeued := cache.fifo.Dequeue()
	secondDequeued := cache.fifo.Dequeue()

	assert.NotEqual(t, "", firstDequeued, "Should be able to dequeue first item (item2)")
	assert.Equal(t, item2Key, firstDequeued, "item2 should have been at the head after item1 was re-enqueued")
	assert.NotEqual(t, "", secondDequeued, "Should be able to dequeue second item (item1)")
	assert.Equal(t, item1Key, secondDequeued, "item1 should be at the end after being re-enqueued")

	// Re-enqueue them to not mess up other potential test logic if cache is reused (though it's not here)
	if firstDequeued != "" { // Check if Dequeue returned a valid item before Enqueueing
		cache.fifo.Enqueue(firstDequeued)
	}
	if secondDequeued != "" {
		cache.fifo.Enqueue(secondDequeued)
	}

	handler.ResetCountersAndMessages()
}

func TestCacheLF_SaveDirtyToDB_ItemNotLoadedFromCache(t *testing.T) {
	cfg, handler := newNoOpTestConfigLF()
	cache := NewLazyWriterCacheLF[testItemLF](cfg)
	defer cache.Shutdown()

	phantomItemKey := "phantom_item_lf"

	// Manually add a key to cache.dirty
	// This simulates a scenario where an item was marked dirty but possibly removed from cache
	// before saveDirtyToDB processes it.
	cache.dirty.Store(phantomItemKey, true)

	// Ensure no panic and Save is not called for the phantom item
	assert.NotPanics(t, func() {
		cache.Flush()
	})

	assert.Equal(t, int64(0), handler.GetSaveCount(phantomItemKey), "Save should not have been called for phantom item")

	// The item should be removed from the dirty list by saveDirtyToDB even if not found in cache
	_, isStillDirty := cache.dirty.Load(phantomItemKey)
	assert.False(t, isStillDirty, "Phantom item should be cleared from dirty list")

	handler.ResetCountersAndMessages()
}
