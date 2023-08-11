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
	"fmt"
	"time"

	"github.com/pilotso11/lazywritercache"
	"github.com/pilotso11/lazywritercache/lockfree/lockfreequeue"
	"github.com/puzpuzpuz/xsync"
)

type CacheableLF interface {
	Key() string
	CopyKeyDataFrom(from CacheableLF) CacheableLF // This should copy in DB only ID fields.  If gorm.Model is implement this is ID, creationTime, updateTime, deleteTime
}

// EmptyCacheableLF - placeholder used as a return value if the cache can't find anything
type EmptyCacheableLF struct {
}

func (i EmptyCacheableLF) Key() string {
	return ""
}

func (i EmptyCacheableLF) CopyKeyDataFrom(from CacheableLF) CacheableLF {
	return from // no op
}

var _ CacheableLF = (*EmptyCacheableLF)(nil)

type CacheReaderWriterLF[T CacheableLF] interface {
	Find(key string, tx any) (T, error)
	Save(item T, tx any) error
	BeginTx() (tx any, err error)
	CommitTx(tx any)
	Info(msg string, action string, item ...T)
	Warn(msg string, action string, item ...T)
}

type ConfigLF[T CacheableLF] struct {
	handler      CacheReaderWriterLF[T]
	Limit        int
	LookupOnMiss bool // If true, a cache miss will query the DB, with associated performance hit!
	WriteFreq    time.Duration
	PurgeFreq    time.Duration
}

func NewDefaultConfigLF[T CacheableLF](handler CacheReaderWriterLF[T]) ConfigLF[T] {
	return ConfigLF[T]{
		handler:      handler,
		Limit:        10000,
		LookupOnMiss: true,
		WriteFreq:    500 * time.Millisecond,
		PurgeFreq:    10 * time.Second,
	}
}

// LazyWriterCacheLF This cache implementation assumes this process OWNS the database.
// This implementation is lock free which is generally faster for many parallel read use cases but
// is restricted to using string for the keys.
//
// There is no synchronisation on Save or any error handling if the DB is in an inconsistent state
// To use this in a distributed mode, we'd need to replace it with something like REDIS that keeps a distributed
// cache for update, and then use a single writer to persist to the DB - with some clustering strategy
type LazyWriterCacheLF[T CacheableLF] struct {
	ConfigLF[T]
	cache    *xsync.MapOf[string, T]
	dirty    *xsync.MapOf[string, bool]
	fifo     *lockfreequeue.LockFreeQueue[string]
	stopping bool
	lazywritercache.CacheStats
}

// NewLazyWriterCacheLF creates a new cache and starts up its lazy db writer ticker.
// Users need to pass a DB Find function and ensure their objects implement lazywritercache.Cacheable which has two functions,
// one to return the Key() and the other to copy key variables into the cached item from the DB loaded item. (i.e. the number ID, update time etc.)
// because the lazy write cannot just "Save" the item back to the DB as it might have been updated during the lazy write as its asynchronous.
func NewLazyWriterCacheLF[T CacheableLF](cfg ConfigLF[T]) *LazyWriterCacheLF[T] {
	cache := LazyWriterCacheLF[T]{
		ConfigLF: cfg,
		cache:    xsync.NewMapOf[T](),
		dirty:    xsync.NewMapOf[bool](),
		fifo:     lockfreequeue.NewLockFreeQueue[string](),
	}

	if cache.WriteFreq > 0 { // start lazyWriter, use write freq zero for testing
		go cache.lazyWriter()
	}

	if cache.Limit > 0 && cache.PurgeFreq > 0 { // start eviction manager goroutine
		go cache.evictionManager()
	}

	return &cache
}

// Load will lock and load an item from the cache and then release the lock.
func (c *LazyWriterCacheLF[T]) Load(key string) (T, bool) {
	item, ok := c.cache.Load(key)
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
func (c *LazyWriterCacheLF[T]) Save(item T) {
	c.Stores.Add(1)
	c.cache.Store(item.Key(), item) // save in cache
	c.dirty.Store(item.Key(), true) // add to dirty list
	c.fifo.Enqueue(item.Key())
}

// Go routine to Save the dirty records to the DB, this is the lazy writer
func (c *LazyWriterCacheLF[T]) saveDirtyToDB() {
	// Get all the dirty records
	// return without any locks if there is no work
	if c.dirty.Size() == 0 {
		return
	}

	c.handler.Info(fmt.Sprintf("Found %d dirty records to write to the DB", c.dirty.Size()), "write-dirty")
	success := 0
	fail := 0

	// Catch any panics
	defer func() {
		if r := recover(); r != nil {
			c.handler.Warn(fmt.Sprintf("Panic in lazy write %v", r), "write-dirty")
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

	c.dirty.Range(func(k string, _ bool) bool {
		c.dirty.Delete(k)
		item, ok := c.cache.Load(k)
		if !ok {
			return true
		}
		// Load the item from the DB.
		// If the item already exists in the DB make sure we merge any key data if needed.
		// ORMs will need this if they are managing row ID keys.
		old, err := c.handler.Find(item.Key(), tx)
		if err == nil {
			// Copy the key data in for the existing record into the new record, so we can save it
			item = item.CopyKeyDataFrom(old).(T)
		}

		// Save back the merged item
		err = c.handler.Save(item, tx)
		c.DirtyWrites.Add(1)

		if err != nil {
			c.handler.Warn(fmt.Sprintf("Error saving %s to DB: %v", old.Key(), err), "write-dirty", item)
			fail++
			return true // don't update cache, move to next item
		}

		// Briefly lock the cache and update it with the merged data.
		// As we have not held the lock during the DB update, there is a race condition where a
		// new write to the cache in the middle of the update could be overridden by the results.
		// To avoid this we call copy key data back on the saved cache item.  So we only update the
		// cache with the merged key data, assuming the cache user has updated their own data
		// as they desired.
		func() {
			rCopy, ok := c.cache.Load(item.Key())
			if ok {
				// Re-save the item
				c.cache.Store(item.Key(), rCopy.CopyKeyDataFrom(item).(T))
			} else {
				c.handler.Warn(fmt.Sprintf("Deferred update attempted on purged cache item, saved but not re-added: %v", item.Key()), "write-dirty", item)
			}
		}()
		success++
		return true
	})

	if fail > 0 {
		c.handler.Warn(fmt.Sprintf("Error syncing cache to DB: %v failures, %v successes", fail, success), "write-dirty")
	} else {
		c.handler.Info(fmt.Sprintf("Completed DB Sync, flushed %d records", success), "write-dirty")
	}
}

// ClearDirty forcefully empties the dirty queue, for example if the cache has just been forcefully loaded from the db, and
// you want to avoid the overhead of retrying to write it all, then ClearDirty may be useful.
// ClearDirty will fail if the cache is not locked
func (c *LazyWriterCacheLF[T]) ClearDirty() {
	c.dirty = xsync.NewMapOf[bool]() // clear dirty list, these all came from the DB
}

// Go routine to evict the cache every few seconds to keep it trimmed to the desired size - or there abouts
func (c *LazyWriterCacheLF[T]) evictionManager() {
	cnt := time.Duration(0)
	for {
		time.Sleep(time.Second) // every second we check for stop
		cnt = cnt + time.Second
		if c.stopping {
			return
		}
		if cnt > c.PurgeFreq {
			c.evictionProcessor()
			cnt = 0
		}
	}
}

// process evictions if the cache is larger than desired
func (c *LazyWriterCacheLF[T]) evictionProcessor() {
	for c.cache.Size() > c.Limit {
		if func() bool {
			toRemove := c.fifo.Dequeue()
			dirty, ok := c.dirty.Load(toRemove)
			if ok && dirty {
				// This is a dirty item, we can't evict it, so we put it back on the queue
				item, _ := c.cache.Load(toRemove)
				c.handler.Warn("Dirty items at the top of the purge queue, skipping eviction", "evict", item)
				c.fifo.Enqueue(toRemove)
				return true
			}

			c.cache.Delete(toRemove)
			c.Evictions.Add(1)
			return false
		}() {
			return
		}
	}
}

// this is the lazy writer goroutine
func (c *LazyWriterCacheLF[T]) lazyWriter() {
	for {
		time.Sleep(c.WriteFreq)
		c.saveDirtyToDB()
		if c.stopping {
			return
		}
	}

}

// Flush forces all dirty items to be written to the database.
// Flush should be called before exiting the application otherwise dirty writes will be lost.
// As the lazy writer is set up with a timer this should only need to be called at exit.
func (c *LazyWriterCacheLF[T]) Flush() {
	c.saveDirtyToDB()
}

// Range over all the keys and maps.
//
// As with other Range functions return true to continue iterating or false to stop.
func (c *LazyWriterCacheLF[T]) Range(action func(k string, v T) bool) (n int) {
	c.cache.Range(func(k string, v T) bool {
		n++
		return action(k, v)
	})
	return n
}

// Shutdown signals to the cache it should stop any running goroutines.
// This does not Flush the cache first, so it is recommended call Flush beforehand.
func (c *LazyWriterCacheLF[T]) Shutdown() {
	c.stopping = true
}
