/*
 * Copyright (c) 2023. Vade Mecum Ltd.  All Rights Reserved.
 */

package lazywritercache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type CacheItem interface {
	Key() interface{}
	CopyKeyDataFrom(from CacheItem) CacheItem // This should copy in DB only ID fields.  If gorm.Model is implement this is ID, creationTime, updateTime, deleteTime
}

// EmptyCacheItem - placeholder used as a return value if the cache can't find anything
type EmptyCacheItem struct {
}

func (i EmptyCacheItem) Key() interface{} {
	return ""
}

func (i EmptyCacheItem) CopyKeyDataFrom(from CacheItem) CacheItem {
	return from // no op
}

type CacheReaderWriter interface {
	Find(key interface{}, tx interface{}) (CacheItem, error)
	Save(item CacheItem, tx interface{}) error
	BeginTx() (tx interface{}, err error)
	CommitTx(tx interface{})
	Info(msg string)
	Warn(msg string)
}

type Config struct {
	handler      CacheReaderWriter
	Limit        int
	LookupOnMiss bool // If true, a cache miss will query the DB, with associated performance hit!
	WriteFreq    time.Duration
	PurgeFreq    time.Duration
}

func NewDefaultConfig(handler CacheReaderWriter) Config {
	return Config{
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
type LazyWriterCache struct {
	Config
	cache  map[interface{}]CacheItem
	dirty  map[interface{}]bool
	mutex  sync.Mutex
	locked atomic.Bool
	fifo   []interface{}
	CacheStats
}

// NewLazyWriterCache creates a new cache and starts up its lazy db writer ticker.
// Users need to pass a DB Find function and ensure their objects implement lazywritercache.CacheItem which has two functions,
// one to return the Key() and the other to copy key variables into the cached item from the DB loaded item. (i.e. the number ID, update time etc.)
// because the lazy write cannot just "Save" the item back to the DB as it might have been updated during the lazy write as its asynchronous.
func NewLazyWriterCache(cfg Config) *LazyWriterCache {
	cache := LazyWriterCache{
		Config: cfg,
		cache:  make(map[interface{}]CacheItem),
		dirty:  make(map[interface{}]bool),
	}

	if cache.WriteFreq > 0 { // start lazyWriter, use write freq zero for testing
		go cache.lazyWriter()
	}

	if cache.Limit > 0 && cache.PurgeFreq > 0 { // start eviction manager go routine
		go cache.evictionManager()
	}

	return &cache
}

// Lock the cache. This will panic if the cache is already locked when the mutex is entered.
func (c *LazyWriterCache) Lock() {
	c.mutex.Lock()
	if !c.locked.CompareAndSwap(false, true) {
		panic("LazyWriterCache likely subject to concurrent modification exception, locked in incorrect state")
	}
}

// GetAndRelease will lock and load an item from the cache and then release the lock.
func (c *LazyWriterCache) GetAndRelease(key string) (CacheItem, bool) {
	defer c.Release() // make sure we release the lock even if there is some kind of panic in the Get after the lock
	item, ok := c.GetAndLock(key)
	return item, ok
}

// GetAndLock will lock and load an item from the cache.  It does not release the lock so always call Release after calling GetAndLock, even if nothing is found
// Useful if you are checking to see if something is there and then planning to update it.
func (c *LazyWriterCache) GetAndLock(key string) (CacheItem, bool) {
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

// Save will ensure the cache is locked via GetAndLock before Save and then released using Release
func (c *LazyWriterCache) Save(item CacheItem) {
	if !c.locked.Load() {
		panic("Call to Save to LazyWriterCache without locked cache")
	}
	c.Stores.Add(1)
	c.cache[item.Key()] = item          // save in cache
	c.dirty[item.Key()] = true          // add to dirty list
	c.fifo = append(c.fifo, item.Key()) // add to fifo queue
}

// Release the Lock.  It will panic if not already locked
func (c *LazyWriterCache) Release() {
	if !c.locked.CompareAndSwap(true, false) {
		panic("LazyWriterCache likely subject to concurrent modification exception, locked in incorrect state")
	}
	c.mutex.Unlock()
}

// Get a copy of the dirty records in the cache and clear the dirty record list
// The cache is locked during the copy operations
// The cache objects to be written are copied to the returned list, not their pointers
func (c *LazyWriterCache) getDirtyRecords() (dirty []CacheItem) {
	c.Lock()
	defer c.Release()
	for k := range c.dirty {
		dirty = append(dirty, c.cache[k])
	}
	c.dirty = make(map[interface{}]bool)
	return dirty
}

// Go routine to Save the dirty records to the DB, this is the lazy writer
func (c *LazyWriterCache) saveDirtyToDB() {
	dirty := c.getDirtyRecords()
	if len(dirty) == 0 {
		return
	} // no work to do

	c.handler.Info(fmt.Sprintf("Found %d dirty records to write to the DB", len(dirty)))
	success := 0
	fail := 0
	func() {

		defer func() {
			if r := recover(); r != nil {
				c.handler.Warn(fmt.Sprintf("Panic in lazy write %v", r))
			}
		}()

		tx, err := c.handler.BeginTx()
		if err != nil {
			return
		}
		defer c.handler.CommitTx(tx)
		for _, item := range dirty {
			// Load it

			// If the item already exists in the DB make sure we merge any key data if needed
			old, err := c.handler.Find(item.Key(), tx)
			if err == nil {
				// Copy the key data in for the existing record into the new record, so we can save it
				item = item.CopyKeyDataFrom(old)
			}

			// and Save
			err = c.handler.Save(item, tx)
			c.DirtyWrites.Add(1)

			if err != nil {
				c.handler.Warn(fmt.Sprintf("Error saving %s to DB: %v", old.Key(), err))
				fail++
				return // don't update cache
			}

			func() {
				// Update the cache with the new key data
				c.Lock()
				defer c.Release()
				rCopy, ok := c.cache[item.Key()]
				if ok {
					// Re-save the item
					c.cache[item.Key()] = rCopy.CopyKeyDataFrom(item)
				} else {
					c.handler.Warn(fmt.Sprintf("Deferred update attempted on purged cache item, saved but not re-added: %v", item.Key()))
				}
			}()
			success++
		}
		return
	}()

	if fail > 0 {
		c.handler.Warn(fmt.Sprintf("Error syncing cache to DB: %v failures, %v successes", fail, success))
	} else {
		c.handler.Info(fmt.Sprintf("Completed DB Sync, flushed %d records", success))
	}
}

// ClearDirty forcefully empties the dirty queue, for example if the cache has just been forcefully loaded from the db, and
// you want to avoid the overhead of retrying to write it all, then ClearDirty may be useful.
// ClearDirty will fail if the cache is not locked
func (c *LazyWriterCache) ClearDirty() {
	if !c.locked.Load() {
		panic("Cache is not locked, cannot ClearDirty")
	}
	c.dirty = make(map[interface{}]bool) // clear dirty list, these all came from the DB
}

// Go routine to evict the cache every few seconds to keep it trimmed to the desired size - or there abouts
func (c *LazyWriterCache) evictionManager() {
	for {
		c.evictionProcessor()
		time.Sleep(c.PurgeFreq) // every 10 seconds seems reasonable
	}
}

func (c *LazyWriterCache) evictionProcessor() {
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

func (c *LazyWriterCache) lazyWriter() {
	for {
		time.Sleep(c.WriteFreq)
		c.saveDirtyToDB()
	}

}

func (c *LazyWriterCache) Flush() {
	c.saveDirtyToDB()
}
