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
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Cacheable interface {
	Key() any
	CopyKeyDataFrom(from Cacheable) Cacheable // This should copy in DB only ID fields.  If gorm.Model is implement this is ID, creationTime, updateTime, deleteTime
}

// EmptyCacheable - placeholder used as a return value if the cache can't find anything
type EmptyCacheable struct {
}

func (i EmptyCacheable) Key() any {
	return ""
}

func (i EmptyCacheable) CopyKeyDataFrom(from Cacheable) Cacheable {
	return from // no op
}

type CacheReaderWriter[K comparable, T Cacheable] interface {
	Find(key K, tx any) (T, error)
	Save(item T, tx any) error
	BeginTx() (tx any, err error)
	CommitTx(tx any)
	Info(msg string)
	Warn(msg string)
}

type Config[K comparable, T Cacheable] struct {
	handler      CacheReaderWriter[K, T]
	Limit        int
	LookupOnMiss bool // If true, a cache miss will query the DB, with associated performance hit!
	WriteFreq    time.Duration
	PurgeFreq    time.Duration
}

func NewDefaultConfig[K comparable, T Cacheable](handler CacheReaderWriter[K, T]) Config[K, T] {
	return Config[K, T]{
		handler:      handler,
		Limit:        10000,
		LookupOnMiss: true,
		WriteFreq:    500 * time.Millisecond,
		PurgeFreq:    10 * time.Second,
	}
}

type CacheStats struct {
	Hits        atomic.Int64
	Misses      atomic.Int64
	Stores      atomic.Int64
	Evictions   atomic.Int64
	DirtyWrites atomic.Int64
}

func (s *CacheStats) String() string {
	return fmt.Sprintf("Hits: %v, Misses %v, Stores %v, Evictions %v, Dirty Writes: %v",
		s.Hits.Load(), s.Misses.Load(), s.Stores.Load(), s.Evictions.Load(), s.DirtyWrites.Load())
}

func (s *CacheStats) JSON() string {
	return fmt.Sprintf(`{"hits": %v, "misses": %v, "stores": %v, "evictions": %v, "dirty-writes": %v}`,
		s.Hits.Load(), s.Misses.Load(), s.Stores.Load(), s.Evictions.Load(), s.DirtyWrites.Load())

}

// LazyWriterCache This cache implementation assumes this process OWNS the database
// There is no synchronisation on Save or any error handling if the DB is in an inconsistent state
// To use this in a distributed mode, we'd need to replace it with something like REDIS that keeps a distributed
// cache for update, and then use a single writer to persist to the DB - with some clustering strategy
type LazyWriterCache[K comparable, T Cacheable] struct {
	Config[K, T]
	cache    map[K]T
	dirty    map[K]bool
	mutex    sync.Mutex
	locked   atomic.Bool
	fifo     []K
	stopping atomic.Bool
	CacheStats
}

// NewLazyWriterCache creates a new cache and starts up its lazy db writer ticker.
// Users need to pass a DB Find function and ensure their objects implement lazywritercache.Cacheable which has two functions,
// one to return the Key() and the other to copy key variables into the cached item from the DB loaded item. (i.e. the number ID, update time etc.)
// because the lazy write cannot just "Save" the item back to the DB as it might have been updated during the lazy write as its asynchronous.
func NewLazyWriterCache[K comparable, T Cacheable](cfg Config[K, T]) *LazyWriterCache[K, T] {
	cache := LazyWriterCache[K, T]{
		Config: cfg,
		cache:  make(map[K]T),
		dirty:  make(map[K]bool),
	}

	if cache.WriteFreq > 0 { // start lazyWriter, use write freq zero for testing
		go cache.lazyWriter()
	}

	if cache.Limit > 0 && cache.PurgeFreq > 0 { // start eviction manager goroutine
		go cache.evictionManager()
	}

	return &cache
}

// Lock the cache. This will panic if the cache is already locked when the mutex is entered.
func (c *LazyWriterCache[K, T]) Lock() {
	c.mutex.Lock()
	if !c.locked.CompareAndSwap(false, true) {
		panic("LazyWriterCache likely subject to concurrent modification exception, locked in incorrect state")
	}
}

// GetAndRelease will lock and load an item from the cache and then release the lock.
func (c *LazyWriterCache[K, T]) GetAndRelease(key K) (T, bool) {
	defer c.Release() // make sure we release the lock even if there is some kind of panic in the Get after the lock
	item, ok := c.GetAndLock(key)
	return item, ok
}

// GetAndLock will lock and load an item from the cache.  It does not release the lock so always call Release after calling GetAndLock, even if nothing is found
// Useful if you are checking to see if something is there and then planning to update it.
func (c *LazyWriterCache[K, T]) GetAndLock(key K) (T, bool) {
	c.Lock()
	item, ok := c.cache[key]
	if !ok {
		c.Misses.Add(1)
		if c.LookupOnMiss {
			item, err := c.handler.Find(key, nil)
			if err == nil {
				c.Save(item)
				return item, true
			}
		}
		return item, false
	}
	c.Hits.Add(1)
	return item, ok
}

// Save updates an item in the cache.
// The cache must already have been locked, if not we will panic.
//
// The expectation is GetAndLock has been called first, and a Release has been deferred.
func (c *LazyWriterCache[K, T]) Save(item T) {
	if !c.locked.Load() {
		panic("Call to Save to LazyWriterCache without locked cache")
	}
	c.Stores.Add(1)
	c.cache[item.Key().(K)] = item          // save in cache
	c.dirty[item.Key().(K)] = true          // add to dirty list
	c.fifo = append(c.fifo, item.Key().(K)) // add to fifo queue
}

// Release the Lock.  It will panic if not already locked
func (c *LazyWriterCache[K, T]) Release() {
	if !c.locked.CompareAndSwap(true, false) {
		panic("LazyWriterCache likely subject to concurrent modification exception, locked in incorrect state")
	}
	c.mutex.Unlock()
}

// Get a copy of the dirty records in the cache and clear the dirty record list
// The cache is locked during the copy operations
// The cache objects to be written are copied to the returned list, not their pointers
func (c *LazyWriterCache[K, T]) getDirtyRecords() (dirty []Cacheable) {
	c.Lock()
	defer c.Release()
	for k := range c.dirty {
		dirty = append(dirty, c.cache[k])
	}
	c.dirty = make(map[K]bool)
	return dirty
}

// Go routine to Save the dirty records to the DB, this is the lazy writer
func (c *LazyWriterCache[K, T]) saveDirtyToDB() {
	// Get all the dirty records
	// return without any locks if there is no work
	dirty := c.getDirtyRecords()
	if len(dirty) == 0 {
		return
	}

	c.handler.Info(fmt.Sprintf("Found %d dirty records to write to the DB", len(dirty)))
	success := 0
	fail := 0

	// Catch any panics
	defer func() {
		if r := recover(); r != nil {
			c.handler.Warn(fmt.Sprintf("Panic in lazy write %v", r))
		}
	}()

	// We do the whole list of dirty writes
	// as a single DB transaction.
	// For most databases this is a smart choice performance wise
	// even though it holds the lock on the DB side longer.
	tx, err := c.handler.BeginTx()
	if err != nil {
		return
	}
	// Ensure the commit runs
	defer c.handler.CommitTx(tx)

	for _, item := range dirty {

		// Load the item from the DB.
		// If the item already exists in the DB make sure we merge any key data if needed.
		// ORMs will need this if they are managing row ID keys.
		old, err := c.handler.Find(item.Key().(K), tx)
		if err == nil {
			// Copy the key data in for the existing record into the new record, so we can save it
			item = item.CopyKeyDataFrom(old)
		}

		// Save back the merged item
		err = c.handler.Save(item.(T), tx)
		c.DirtyWrites.Add(1)

		if err != nil {
			c.handler.Warn(fmt.Sprintf("Error saving %s to DB: %v", old.Key(), err))
			fail++
			return // don't update cache
		}

		// Briefly lock the cache and update it with the merged data.
		// As we have not held the lock during the DB update, there is a race condition where a
		// new write to the cache in the middle of the update could be overridden by the results.
		// To avoid this we call copy key data back on the saved cache item.  So we only update the
		// cache with the merged key data, assuming the cache user has updated their own data
		// as they desired.
		func() {
			c.Lock()
			defer c.Release()
			rCopy, ok := c.cache[item.Key().(K)]
			if ok {
				// Re-save the item
				c.cache[item.Key().(K)] = rCopy.CopyKeyDataFrom(item).(T)
			} else {
				c.handler.Warn(fmt.Sprintf("Deferred update attempted on purged cache item, saved but not re-added: %v", item.Key()))
			}
		}()
		success++
	}

	if fail > 0 {
		c.handler.Warn(fmt.Sprintf("Error syncing cache to DB: %v failures, %v successes", fail, success))
	} else {
		c.handler.Info(fmt.Sprintf("Completed DB Sync, flushed %d records", success))
	}
}

// ClearDirty forcefully empties the dirty queue, for example if the cache has just been forcefully loaded from the db, and
// you want to avoid the overhead of retrying to write it all, then ClearDirty may be useful.
// ClearDirty will fail if the cache is not locked
func (c *LazyWriterCache[K, T]) ClearDirty() {
	if !c.locked.Load() {
		panic("Cache is not locked, cannot ClearDirty")
	}
	c.dirty = make(map[K]bool) // clear dirty list, these all came from the DB
}

// Go routine to evict the cache every few seconds to keep it trimmed to the desired size - or there abouts
func (c *LazyWriterCache[K, T]) evictionManager() {
	cnt := time.Duration(0)
	for {
		time.Sleep(time.Second) // every second we check for stop
		cnt = cnt + time.Second
		if c.stopping.Load() {
			return
		}
		if cnt > c.PurgeFreq {
			c.evictionProcessor()
			cnt = 0
		}
	}
}

// process evictions if the cache is larger than desired
func (c *LazyWriterCache[K, T]) evictionProcessor() {
	for len(c.cache) > c.Limit {
		func() {
			c.Lock()
			defer c.Release()
			toRemove := c.fifo[0]
			c.fifo = c.fifo[1:] // pop item
			delete(c.cache, toRemove)
			c.Evictions.Add(1)
		}()
	}
}

// this is the lazy writer goroutine
func (c *LazyWriterCache[K, T]) lazyWriter() {
	for {
		time.Sleep(c.WriteFreq)
		c.saveDirtyToDB()
		if c.stopping.Load() {
			return
		}
	}

}

// Flush forces all dirty items to be written to the database.
// Flush should be called before exiting the application otherwise dirty writes will be lost.
// As the lazy writer is set up with a timer this should only need to be called at exit.
func (c *LazyWriterCache[K, T]) Flush() {
	c.saveDirtyToDB()
}

// get all the keys
// hold a lock while extracting the keys but immediately release it
func (c *LazyWriterCache[K, T]) getKeys() []K {
	c.Lock()
	defer c.Release()

	var keys []K
	for k := range c.cache {
		keys = append(keys, k)
	}

	return keys
}

// Range over all the keys and maps.
// The cache is locked for the duration of the range function to avoid synchronous access issues.
// If the Range action is expensive, consider using the lock free implementation in lockfree/LazyWriterCacheLF.
//
// As with other Range functions return true to continue iterating or false to stop.
func (c *LazyWriterCache[K, T]) Range(action func(k K, v T) bool) (n int) {
	c.Lock()
	defer c.Release()
	for k, v := range c.cache {
		n++
		if !action(k, v) {
			return
		}
	}
	return
}

// Shutdown signals to the cache it should stop any running goroutines.
// This does not Flush the cache first, so it is recommended call Flush beforehand.
func (c *LazyWriterCache[K, T]) Shutdown() {
	c.stopping.Store(true)
}

// Invalidate flushes and empties the cache forcing reloads
func (c *LazyWriterCache[K, T]) Invalidate() {
	c.Flush()
	c.Lock()
	defer c.Release()
	c.cache = make(map[K]T)
	c.fifo = make([]K, 0)
}
