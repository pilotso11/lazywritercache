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
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var ErrConcurrentModification = errors.New("likely concurrent modification detected in LazyWriterCache")
var ErrNotLocked = errors.New("cache not locked on read")

type Cacheable interface {
	Key() any
	CopyKeyDataFrom(from Cacheable) Cacheable // This should copy in DB only ID fields.  If gorm.Model is implemented this is ID, creationTime, updateTime, deleteTime
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
	Info(msg string, action string, item ...T)
	Warn(msg string, action string, item ...T)
}

type Config[K comparable, T Cacheable] struct {
	handler         CacheReaderWriter[K, T]
	Limit           int
	LookupOnMiss    bool // If true, a cache miss will query the DB, with associated performance hit!
	WriteFreq       time.Duration
	PurgeFreq       time.Duration
	DeadlockLimit   int  // Number of times to retry a deadlock before giving up
	SyncWrites      bool // Synchronize cache flush to storage, this is slower but can help with deadlock contention for some use cases, especially where multiple caches may be impacted by the same DB writes because of hooks
	FlushOnShutdown bool // Flush the cache on shutdown
}

func NewDefaultConfig[K comparable, T Cacheable](handler CacheReaderWriter[K, T]) Config[K, T] {
	return Config[K, T]{
		handler:       handler,
		Limit:         10000,
		LookupOnMiss:  true,
		WriteFreq:     500 * time.Millisecond,
		PurgeFreq:     10 * time.Second,
		DeadlockLimit: 5,
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
	ctx    context.Context
	cancel context.CancelFunc
	cache  map[K]T
	dirty  map[K]bool
	mutex  sync.Mutex
	locked atomic.Bool
	fifo   []K
	CacheStats
}

// NewLazyWriterCache creates a new cache and starts up its lazy db writer ticker.
// Users need to pass a DB Find function and ensure their objects implement lazywritercache.Cacheable which has two functions,
// one to return the Key() and the other to copy key variables into the cached item from the DB loaded item. (i.e. the number ID, update time etc.)
// because the lazy write cannot just "Save" the item back to the DB as it might have been updated during the lazy write as its asynchronous.
func NewLazyWriterCache[K comparable, T Cacheable](cfg Config[K, T]) *LazyWriterCache[K, T] {
	return NewLazyWriterCacheWithContext(context.Background(), cfg)
}

// NewLazyWriterCacheWithContext creates a new cache and starts up its lazy db writer ticker and link's its internal cancel context to a parent context passed in.
// Users need to pass a DB Find function and ensure their objects implement lazywritercache.Cacheable which has two functions,
// one to return the Key() and the other to copy key variables into the cached item from the DB loaded item. (i.e. the number ID, update time etc.)
// because the lazy write cannot just "Save" the item back to the DB as it might have been updated during the lazy write as its asynchronous.
func NewLazyWriterCacheWithContext[K comparable, T Cacheable](ctx context.Context, cfg Config[K, T]) *LazyWriterCache[K, T] {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	cache := LazyWriterCache[K, T]{
		Config: cfg,
		cache:  make(map[K]T),
		dirty:  make(map[K]bool),
		cancel: cancel,
		ctx:    ctx,
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
func (c *LazyWriterCache[K, T]) Lock() error {
	// Attempt to set the atomic lock flag FIRST
	if !c.locked.CompareAndSwap(false, true) {
		// If CAS fails, it means it was already locked (or another goroutine just locked it).
		// Return the error immediately WITHOUT acquiring c.mutex.
		return ErrConcurrentModification
	}
	// If CAS was successful, THEN acquire the mutex.
	c.mutex.Lock()
	return nil
}

func (c *LazyWriterCache[K, T]) unlockWithPanic() {
	err := c.Unlock()
	if err != nil {
		panic(err)
	}
}

// GetAndRelease will lock and load an item from the cache and then release the lock.
func (c *LazyWriterCache[K, T]) GetAndRelease(key K) (T, bool, error) {
	defer c.unlockWithPanic() // make sure we release the lock even if there is some kind of panic in the Get after the lock
	return c.GetAndLock(key)
}

// GetAndLock will lock and load an item from the cache.  It does not release the lock so always call Unlock after calling GetAndLock, even if nothing is found
// Useful if you are checking to see if something is there and then planning to update it.
func (c *LazyWriterCache[K, T]) GetAndLock(key K) (T, bool, error) {
	var empty T
	if err := c.Lock(); err != nil {
		return empty, false, err
	}
	return c.GetFromLocked(key)
}

// GetFromLocked will  load an item from a previously locked cache.
func (c *LazyWriterCache[K, T]) GetFromLocked(key K) (T, bool, error) {
	var item T
	if !c.locked.Load() {
		return item, false, ErrNotLocked
	}

	item, ok := c.cache[key]
	if !ok {
		c.Misses.Add(1)
		if c.LookupOnMiss {
			item, err := c.handler.Find(key, nil)
			if err == nil {
				c.Save(item)
				return item, true, nil
			}
		}
		return item, false, nil
	}
	c.Hits.Add(1)
	return item, ok, nil
}

// Save updates an item in the cache.
// The cache must already have been locked, if not we will panic.
//
// The expectation is GetAndLock has been called first, and a Unlock has been deferred.
func (c *LazyWriterCache[K, T]) Save(item T) {
	if !c.locked.Load() {
		panic("Call to Save to LazyWriterCache without locked cache")
	}
	c.Stores.Add(1)
	c.cache[item.Key().(K)] = item          // save in cache
	c.dirty[item.Key().(K)] = true          // add to dirty list
	c.fifo = append(c.fifo, item.Key().(K)) // add to fifo queue
}

// Unlock the Lock.  It will panic if not already locked
func (c *LazyWriterCache[K, T]) Unlock() error {
	if !c.locked.CompareAndSwap(true, false) {
		return ErrConcurrentModification
	}
	c.mutex.Unlock()
	return nil
}

// Get a copy of the dirty records in the cache and clear the dirty record list
// The cache is locked during the copy operations
// The cache objects to be written are copied to the returned list, not their pointers
func (c *LazyWriterCache[K, T]) getDirtyRecords() (dirty []Cacheable, err error) {
	if err := c.Lock(); err != nil {
		return nil, err
	}
	defer c.unlockWithPanic()
	for k := range c.dirty {
		dirty = append(dirty, c.cache[k])
	}
	c.dirty = make(map[K]bool)
	return dirty, nil
}

var globalWriterLock sync.Mutex

// Go routine to Save the dirty records to the DB, this is the lazy writer
func (c *LazyWriterCache[K, T]) saveDirtyToDB() (err error) {
	if c.Config.SyncWrites {
		globalWriterLock.Lock()
		defer globalWriterLock.Unlock()
	}

	// Get all the dirty records
	// return without any locks if there is no work
	dirty, err := c.getDirtyRecords()
	if len(dirty) == 0 {
		return
	}

	c.handler.Info(fmt.Sprintf("Found %d dirty records to write to the DB", len(dirty)), "dirty-write")
	success := 0
	fail := 0

	// Catch any panics
	defer func() {
		if r := recover(); r != nil {
			c.handler.Warn(fmt.Sprintf("Panic in lazy write %v", r), "dirty-write")
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
		var old T
		old, err = c.handler.Find(item.Key().(K), tx)
		if err == nil {
			// Copy the key data in for the existing record into the new record, so we can save it
			item = item.CopyKeyDataFrom(old)
		}

		// Save back the merged item
		err = c.handler.Save(item.(T), tx)
		c.DirtyWrites.Add(1)

		if err != nil {
			if strings.Contains(err.Error(), "deadlock") {
				c.handler.Info(fmt.Sprintf("Deadlock detected, retrying %v", item.Key()), "write", item.(T))
				// Put the items back in the dirty queue
				for _, dirty := range dirty {
					c.dirty[dirty.Key().(K)] = true
				}
				fail++
				return
			}
			// Otherwise just report the error
			val := reflect.ValueOf(item)
			toStr := val.MethodByName("String")
			strVal := ""
			if toStr.IsValid() {
				strVal = toStr.Call([]reflect.Value{})[0].String()
			}
			c.handler.Warn(fmt.Sprintf("Error saving %v to DB: %v (%v)", old.Key(), err, strVal), "write", item.(T))
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
			err = c.Lock()
			defer c.unlockWithPanic()
			rCopy, ok := c.cache[item.Key().(K)]
			if ok {
				// Re-save the item
				c.cache[item.Key().(K)] = rCopy.CopyKeyDataFrom(item).(T)
			} else {
				c.handler.Warn(fmt.Sprintf("Deferred update attempted on purged cache item, saved but not re-added: %v", item.Key()), "write", item.(T))
			}
		}()
		success++
	}

	if fail > 0 {
		c.handler.Warn(fmt.Sprintf("Error syncing cache to DB: %v failures, %v successes", fail, success), "dirty-write")
	} else {
		c.handler.Info(fmt.Sprintf("Completed DB Sync, flushed %d records", success), "dirty-write")
	}
	return
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
	tick := time.NewTicker(c.PurgeFreq)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			err := c.evictionProcessor()
			if err != nil {
				c.handler.Warn(fmt.Sprintf("error from evictionProcessor: %v", err), "eviction")
			}
		case <-c.ctx.Done():
			// Stop
			return
		}
	}
}

// process evictions if the cache is larger than desired
func (c *LazyWriterCache[K, T]) evictionProcessor() error {
	for {
		if err := c.Lock(); err != nil {
			return err
		}
		cLen := len(c.cache)
		if err := c.Unlock(); err != nil {
			return err
		}
		if cLen <= c.Limit {
			return nil
		}
		done, err := func() (bool, error) {
			if err := c.Lock(); err != nil {
				return false, err
			}
			defer c.unlockWithPanic()
			toRemove := c.fifo[0]
			if c.dirty[toRemove] {
				c.handler.Warn("Dirty items at the top of the purge queue, skipping eviction", "eviction")
				return true, nil
			}
			c.fifo = c.fifo[1:] // pop item
			delete(c.cache, toRemove)
			c.Evictions.Add(1)
			return false, nil
		}()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

// this is the lazy writer goroutine
func (c *LazyWriterCache[K, T]) lazyWriter() {
	ticker := time.NewTicker(c.WriteFreq)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := c.saveDirtyToDB()
			if err != nil {
				c.handler.Warn(fmt.Sprintf("error saving dirty records to the db: %v", err), "dirty write")
			}
		case <-c.ctx.Done():
			// Shutdown
			if c.Config.FlushOnShutdown {
				c.handler.Info("flushing cache on shutdown", "shutdown")
				err := c.Flush()
				if err != nil {
					c.handler.Warn(fmt.Sprintf("error flushing cache to DB: %v", err), "flush")
				}
			}
			return
		}
	}
}

// Flush forces all dirty items to be written to the database.
// Flush should be called before exiting the application otherwise dirty writes will be lost.
// As the lazy writer is set up with a timer this should only need to be called at exit.
func (c *LazyWriterCache[K, T]) Flush() error {
	return c.saveDirtyToDB()
}

// get all the keys
// hold a lock while extracting the keys but immediately release it
func (c *LazyWriterCache[K, T]) getKeys() ([]K, error) {
	var keys []K
	err := c.Lock()
	if err != nil {
		return keys, err
	}
	defer c.unlockWithPanic()

	for k := range c.cache {
		keys = append(keys, k)
	}

	return keys, nil
}

// Range over all the keys and maps.
// The cache is locked for the duration of the range function to avoid synchronous access issues.
// If the Range action is expensive, consider using the lock free implementation in lockfree/LazyWriterCacheLF.
//
// As with other Range functions return true to continue iterating or false to stop.
func (c *LazyWriterCache[K, T]) Range(action func(k K, v T) bool) (n int, err error) {
	if err = c.Lock(); err != nil {
		return n, err
	}
	defer c.unlockWithPanic()
	for k, v := range c.cache {
		n++
		if !action(k, v) {
			return
		}
	}
	return
}

// Shutdown signals to the cache it should stop any running goroutines.
// This does not Flush the cache first unless Config.FlushOnShutdown is set to true.
func (c *LazyWriterCache[K, T]) Shutdown() {
	c.cancel()
}

// Invalidate flushes and empties the cache forcing reloads
func (c *LazyWriterCache[K, T]) Invalidate() error {
	if err := c.Flush(); err != nil {
		return err
	}
	if err := c.Lock(); err != nil {
		return err
	}
	defer c.unlockWithPanic()
	c.cache = make(map[K]T)
	c.fifo = make([]K, 0)
	return nil
}
