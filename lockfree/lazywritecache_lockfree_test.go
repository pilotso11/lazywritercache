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
	"encoding/json"
	"math/rand"
	"strconv"
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

func newNoOpTestConfigLF(panics ...bool) ConfigLF[testItemLF] {
	doPanics := len(panics) > 0 && panics[0]
	readerWriter := NewNoOpReaderWriterLF[testItemLF](newtestItemLF, doPanics)
	return ConfigLF[testItemLF]{
		handler:      readerWriter,
		Limit:        1000,
		LookupOnMiss: false,
		WriteFreq:    0,
		PurgeFreq:    0,
	}
}
func TestCacheStoreLoadLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())

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
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
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
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())

	assert.NotPanics(t, func() {
		cache.Load("missing")
	}, "get and Release")

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
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
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
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
	for i := 0; i < b.N; i++ {
		id := strconv.Itoa(i % cacheSize)
		item := testItemLF{id: id}
		cache.Save(item)
	}
	b.ReportAllocs()
}

func cacheReadLF(b *testing.B, cacheSize int) {
	// init
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
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

	cfg := newNoOpTestConfigLF()
	cfg.Limit = 20
	cache := NewLazyWriterCacheLF(cfg)

	for i := 0; i < 30; i++ {
		id := strconv.Itoa(i)
		item := testItemLF{id: id}
		cache.Save(item)
	}
	assert.Equal(t, 30, cache.cache.Size())
	cache.evictionProcessor()
	assert.Equal(t, 20, cache.cache.Size())
	_, ok := cache.cache.Load("0")
	assert.Falsef(t, ok, "0 has been evicted")
	_, ok = cache.cache.Load("9")
	assert.Falsef(t, ok, "9 has been evicted")
	_, ok = cache.cache.Load("10")
	assert.Truef(t, ok, "10 has not been evicted")
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
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())

	cache.Save(item)
	cache.Save(itemLF)

	item3, ok := cache.Load("test1")
	assert.Truef(t, ok, "loaded test")
	assert.Equal(t, item, item3)

}

func TestGormLazyCache_GetAndReleaseWithForcedPanicLF(t *testing.T) {
	item := testItemLF{id: "test1"}
	itemLF := testItemLF{id: "testLF"}
	cfg := newNoOpTestConfigLF(true)
	cache := NewLazyWriterCacheLF(cfg)
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
	cache := NewLazyWriterCacheLF(newNoOpTestConfigLF())
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
	cache := NewLazyWriterCacheLF[testItemLF](newNoOpTestConfigLF())
	cache.Save(testItemLF{id: "test"})
	time.Sleep(100 * time.Millisecond)
	cache.Shutdown()
	time.Sleep(2 * time.Second)

}
