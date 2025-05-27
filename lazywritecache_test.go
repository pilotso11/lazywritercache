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
	"math/rand"
	"strconv"
	"sync"
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
	assert.Len(t, cache.dirty, 2, "dirty records")
	d, err := cache.getDirtyRecords()
	assert.NoError(t, err)
	assert.Contains(t, d, item)
	assert.Contains(t, d, item2)
	assert.Len(t, cache.dirty, 0, "dirty records")

	err = cache.Lock()
	assert.NoError(t, err)
	cache.Save(item2)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.Len(t, cache.dirty, 1, "dirty records")
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
	assert.Len(t, cache.dirty, 2, "dirty records")

	err = cache.Invalidate()
	assert.NoError(t, err)
	assert.Len(t, cache.dirty, 0, "dirty records")
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
