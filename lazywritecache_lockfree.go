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
	"errors"
	"fmt"
	"hash/maphash"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/puzpuzpuz/xsync"

	"github.com/pilotso11/lazywritercache/lockfreequeue"
)

// CacheableLF items must implement a Key method that returns a comparable for use in a map key.
// It must also implement a method to CopyKeyDataFrom another item.  LazyWriterCacheLF must be able to merge any
// key details that might be added externally. For example, a DB table may logically use a string as key (unique constraint)
// but use a sequence as its real primary key.  In this case we need to ensure that if we use the string key internally
// that we can join up with the DB key created on insert.
type CacheableLF[K comparable] interface {
	// Key must return a unique hashable key for the item in order to avoid cache collisions.
	Key() K
	// CopyKeyDataFrom is CRITICAL for data consistency during lazy writes.
	// It should merge ONLY database-managed fields (e.g., auto-increment IDs,
	// created_at/updated_at timestamps managed by the DB) from the `from` item
	// (which is freshly loaded from the DB) into the current item (which is from the cache).
	// This prevents overwriting in-memory changes to other fields made since the item
	// was last loaded or saved, while ensuring DB-generated keys or timestamps are updated.
	// Example: if `from` has a new DB-generated ID or a newer `updated_at` timestamp,
	// those should be copied to the receiver. Other application-specific fields in the
	// receiver should remain untouched by this method.
	CopyKeyDataFrom(from CacheableLF[K]) CacheableLF[K]
}

// EmptyCacheableLF - placeholder used as a return value if the cache can't find anything
type EmptyCacheableLF struct {
}

func (i EmptyCacheableLF) Key() string {
	return ""
}

func (i EmptyCacheableLF) CopyKeyDataFrom(from CacheableLF[string]) CacheableLF[string] {
	return from // no op
}

var _ CacheableLF[string] = (*EmptyCacheableLF)(nil)

// CacheReaderWriterLF defines an interface for managing cache operations and database transactions.
// It supports finding, saving, and transactional operations with recoverable error handling and logging functionality.
type CacheReaderWriterLF[K comparable, T CacheableLF[K]] interface {

	// Find retrieves an item of type T identified by the provided key from the data store.
	// Returns the item and an error if the operation fails.
	Find(ctx context.Context, key K, tx any) (T, error)

	// Save stores the given item in the persistence layer. Save must act as an upsert function, processing both new and existing items..
	Save(ctx context.Context, item T, tx any) error

	// BeginTx initializes and returns a new database transaction along with any potential error encountered.
	BeginTx(ctx context.Context) (tx any, err error)

	// CommitTx commits the provided transaction, completing any changes made within the transaction scope.
	CommitTx(ctx context.Context, tx any) error

	// RollbackTx rolls back the provided transaction, undoing any changes made within the transaction scope.
	RollbackTx(ctx context.Context, tx any) error

	// Info logs an informational message describing a cache action performed on one or more cacheable items provided as input.
	Info(ctx context.Context, msg string, action CacheAction, item ...T)

	// Warn logs a warning with the provided message, cache action, and optional items for additional context.
	Warn(ctx context.Context, msg string, action CacheAction, item ...T)

	// IsRecoverable determines if a database error is transient and the operation
	// should be retried.
	//
	// Recoverable errors are typically:
	//   - Deadlocks (e.g., "database deadlock detected").
	//   - Temporary network issues or timeouts that might resolve on retry.
	//   - Transient transaction serialization failures.
	//
	// Unrecoverable errors typically include:
	//   - Data integrity violations (e.g., unique constraint, foreign key constraint).
	//   - Schema errors or invalid SQL syntax.
	//   - Persistent connection failures or authentication issues.
	//   - "Record not found" errors if the operation expected the record to exist.
	IsRecoverable(_ context.Context, err error) bool

	// Fail logs and handles an irrecoverable error occurring during a flush operation. Affected items are included,
	// and will be supplied unless the failure was triggered by a BeginTx error..
	Fail(ctx context.Context, err error, items ...T)
}

// ConfigLF defines the configuration options for a lazy write cache mechanism, including performance and behavior controls.
type ConfigLF[K comparable, T CacheableLF[K]] struct {
	// Limit defines the maximum number of items in the cache.  This is a hint and can be violated if items are added faster than they can be saved.
	Limit int
	// LookupOnMiss controls whether a cache miss will query the DB,  If false the cache only handles items explicitly put into it.
	LookupOnMiss bool
	// WriteFreq specifies the interval for periodic writes to the database in a lazy write cache mechanism.
	WriteFreq time.Duration
	// PurgeFreq specifies the interval for periodic evictions from the cache in a lazy write cache mechanism.
	PurgeFreq time.Duration
	// FlushOnShutdown determines if the cache should write all pending changes to the database before shutting down.
	FlushOnShutdown bool
	// AllowConcurrentWrites determines if multiple write operations to the database can occur concurrently in the cache.
	AllowConcurrentWrites bool
	// Handler specifies the CacheReaderWriterLF interface for managing cache operations and database transactions in the lazy write cache.
	Handler CacheReaderWriterLF[K, T]
}

// NewDefaultConfigLF initializes a ConfigLF with default settings for cache limits, frequencies, and behavior parameters.
func NewDefaultConfigLF[K comparable, T CacheableLF[K]](handler CacheReaderWriterLF[K, T]) ConfigLF[K, T] {
	return ConfigLF[K, T]{
		Handler:      handler,
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
type LazyWriterCacheLF[K comparable, T CacheableLF[K]] struct {
	ConfigLF[K, T]
	cancel  context.CancelFunc
	cache   *xsync.MapOf[K, T]
	dirty   *xsync.MapOf[K, bool]
	fifo    *lockfreequeue.LockFreeQueue[K]
	writing *atomic.Bool
	hasher  func(k K) uint64
	CacheStats
}

// NewLazyWriterCacheLF creates a new, and starts up its lazy db writer ticker.
// Users need to pass a DB Find function and ensure their objects implement lazywritercache.Cacheable which has two functions,
// one to return the Key() and the other to copy key variables into the cached item from the DB loaded item. (i.e. the number ID, update time etc.)
// because the lazy write cannot just "Save" the item back to the DB as it might have been updated during the lazy write as its asynchronous.
func NewLazyWriterCacheLF[K comparable, T CacheableLF[K]](cfg ConfigLF[K, T]) *LazyWriterCacheLF[K, T] {
	return NewLazyWriterCacheWithContextLF(context.Background(), cfg)
}

// NewLazyWriterCacheWithContextLF creates a new cache passing in a parent context, and starts up its lazy db writer ticker.
// Users need to pass a DB Find function and ensure their objects implement lazywritercache.Cacheable which has two functions,
// one to return the Key() and the other to copy key variables into the cached item from the DB loaded item. (i.e. the number ID, update time etc.)
// because the lazy write cannot just "Save" the item back to the DB as it might have been updated during the lazy write as its asynchronous.
func NewLazyWriterCacheWithContextLF[K comparable, T CacheableLF[K]](ctx context.Context, cfg ConfigLF[K, T]) *LazyWriterCacheLF[K, T] {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Hash func required for xsync.NewTypedMapOf.
	seed := maphash.MakeSeed()
	hasher := func(k K) uint64 {
		return maphash.Comparable(seed, k)
	}

	cache := LazyWriterCacheLF[K, T]{
		ConfigLF: cfg,
		cancel:   cancel,
		cache:    xsync.NewTypedMapOf[K, T](hasher),
		dirty:    xsync.NewTypedMapOf[K, bool](hasher),
		fifo:     lockfreequeue.NewLockFreeQueue[K](),
		writing:  &atomic.Bool{},
		hasher:   hasher,
	}

	if cache.WriteFreq > 0 { // start lazyWriter, use write freq zero for testing
		go cache.lazyWriter(ctx)
	}

	if cache.Limit > 0 && cache.PurgeFreq > 0 { // start eviction manager goroutine
		go cache.evictionManager(ctx)
	}

	return &cache
}

// Load will lock and load an item from the cache and then release the lock.
func (c *LazyWriterCacheLF[K, T]) Load(ctx context.Context, key K) (T, bool) {
	item, ok := c.cache.Load(key)
	if ok {
		c.Hits.Add(1)
		return item, true
	}

	// Cache miss
	c.Misses.Add(1)
	if c.LookupOnMiss {
		// Assuming Handler.Find can take a nil transaction context if not in a transaction
		itemFromDB, err := c.Handler.Find(ctx, key, nil)
		if err == nil {
			// Found in DB, save to cache and return.
			// Save will mark it dirty and add to FIFO for potential eviction.
			c.Save(itemFromDB)
			return itemFromDB, true
		}
		// If err != nil, item was not found in DB or an error occurred.
		// Proceed to return not found.
	}

	// Not found in cache, and either LookupOnMiss is false,
	// or DB lookup failed/returned no item.
	var zeroT T // Ensure a true zero value for T is returned
	return zeroT, false
}

// Save updates an item in the cache.
func (c *LazyWriterCacheLF[K, T]) Save(item T) {
	c.Stores.Add(1)
	c.cache.Store(item.Key(), item) // save in cache
	c.dirty.Store(item.Key(), true) // add to dirty list
	c.fifo.Enqueue(item.Key())
}

// Go routine to Save the dirty records to the DB, this is the lazy writer
func (c *LazyWriterCacheLF[K, T]) saveDirtyToDB(ctx context.Context) {
	// Get all the dirty records
	// return without any locks if there is no work
	if c.dirty.Size() == 0 {
		return
	}

	// Check for concurrent writes
	if !c.AllowConcurrentWrites {
		if !c.writing.CompareAndSwap(false, true) {
			return
		}
		defer c.writing.Store(false)
	}

	c.Handler.Info(ctx, fmt.Sprintf("Found %d dirty records to write to the DB", c.dirty.Size()), ActionWriteDirty)
	success := 0
	fail := 0

	// Catch any panics
	defer func() {
		if r := recover(); r != nil {
			c.Handler.Warn(ctx, fmt.Sprintf("Panic in lazy write %q, data loss possible.", r), ActionWriteDirty)
		}
	}()

	// We do the whole list of dirty writes
	// as a single DB transaction.
	// For most databases this is a smart choice performance wise
	// even though it holds the lock on the DB side longer.
	tx, err := c.Handler.BeginTx(ctx)
	if err != nil && c.Handler.IsRecoverable(ctx, err) {
		c.Handler.Info(ctx, fmt.Sprintf("Recoverable error with BeginTx, batch will be retried: %v", err), ActionWriteDirty)
		return
	} else if err != nil {
		c.Handler.Warn(ctx, fmt.Sprintf("Unrecoverable error from BeginTx: %q, batch will be aborted", err), ActionWriteDirty)
		c.Handler.Fail(ctx, err)
		return
	}
	// Ensure the commit runs

	unCommitted := make([]T, 0)

	c.dirty.Range(func(k K, _ bool) bool {
		c.dirty.Delete(k) // Optimistically remove from dirty; will be re-added if write fails recoverably.
		item, ok := c.cache.Load(k)
		if !ok {
			// item no longer in cache, skip it
			return true
		}
		// Load the item from the DB.
		// If the item already exists in the DB make sure we merge any key data if needed.
		// ORMs will need this if they are managing row ID keys.
		old, err := c.Handler.Find(ctx, item.Key(), tx)
		if err == nil {
			// Copy the key data in for the existing record into the new record, so we can save it
			item = item.CopyKeyDataFrom(old).(T)
		}

		// Save back the merged item
		err = c.Handler.Save(ctx, item, tx)

		if err != nil {
			// Use the Handler's IsRecoverable method to check error type
			if c.Handler.IsRecoverable(ctx, err) {
				c.Handler.Info(ctx, fmt.Sprintf("Recoverable error saving %v to DB, batch will be retried: %v", item.Key(), err), ActionWriteDirty, item)
				unCommitted = append(unCommitted, item) // Add original item from cache for retry
				fail++
				return false // Stop processing this batch, it will be retried.
			}
			c.Handler.Warn(ctx, fmt.Sprintf("Unrecoverable error saving %v to DB: %v", item.Key(), err), ActionWriteDirty, item)
			c.Handler.Fail(ctx, err, item)
			fail++
			// For unrecoverable errors, we don't re-add to unCommitted for this transaction's retry.
			// The item remains out of the dirty list. Consider if this is the desired behavior or if it should be re-added to dirty for a *future* attempt.
			return true // Continue with other items in the batch if possible, though the transaction will likely be rolled back.
		}
		c.DirtyWrites.Add(1)
		// Item successfully saved in DB transaction, keep it for potential commit.
		// The `item` here is the one potentially merged with DB data.
		unCommitted = append(unCommitted, item)

		// Briefly lock the cache and update it with the merged data.
		// The cache operates without locks during the DB update. This means a concurrent write
		// to the same item in the cache could occur. The `CopyKeyDataFrom` call below attempts
		// to merge DB-generated fields (like IDs, timestamps) from the `item` (that was just saved to DB)
		// back into the potentially newer cache item (`rCopy`).
		// This relies heavily on `CopyKeyDataFrom` being correctly implemented to only update
		// specific DB-managed fields, preserving other in-memory changes in `rCopy`.
		func() {
			rCopy, ok := c.cache.Load(item.Key())
			if ok {
				// Re-save the item into the cache, merging DB-generated fields from `item`
				// into the current cache item `rCopy`.
				c.cache.Store(item.Key(), rCopy.CopyKeyDataFrom(item).(T))
			} else {
				// Item was purged from cache between DB save and this update.
				// It was saved to DB, but won't be re-added to cache here.
				c.Handler.Info(ctx, fmt.Sprintf("DB-saved item %v was purged from cache before post-save update.", item.Key()), ActionWriteDirty, item)
			}
		}()
		success++
		return true
	})

	if fail > 0 { // If any save within the batch failed (recoverably or unrecoverably for that item)
		errRollback := c.Handler.RollbackTx(ctx, tx)
		if errRollback != nil && c.Handler.IsRecoverable(ctx, errRollback) {
			c.Handler.Info(ctx, fmt.Sprintf("Error rolling back transaction after failures, will retry: %v", errRollback), ActionWriteDirty)
		} else if errRollback != nil {
			c.Handler.Warn(ctx, fmt.Sprintf("Error rolling back transaction after failures, batch aborted: %v", errRollback), ActionWriteDirty)
			fail += len(unCommitted)
			// unCommitted contains all successfully saved items in this batch.
			// notify the Handler they are lost.
			c.Handler.Fail(ctx, errors.Join(err, errRollback), unCommitted...)
			return
		}
		// Re-mark all items that were part of this transaction attempt as dirty if the transaction failed.
		// This includes items that might have individually "succeeded" but the overall commit failed,
		// or items that had a recoverable error.
		for _, itemFromBatch := range unCommitted { // unCommitted now holds all items attempted in this batch
			c.dirty.Store(itemFromBatch.Key(), true)
		}

		c.Handler.Info(ctx, fmt.Sprintf("Transaction rolled back. Error syncing cache to DB: %v failures, %v successes within batch. All %d items in batch marked for retry.", fail, success, len(unCommitted)), ActionWriteDirty)
		return
	}

	// All individual saves were successful, try to commit.
	err = c.Handler.CommitTx(ctx, tx)
	if err != nil {
		fail++ // Mark a failure for the commit itself.
		// If commit fails, all items in the batch should be retried if the error is recoverable.
		if c.Handler.IsRecoverable(ctx, err) {
			c.Handler.Warn(ctx, fmt.Sprintf("Recoverable error from CommitTx, batch will be retried: %v", err), ActionWriteDirty)
			for _, item := range unCommitted { // unCommitted contains all successfully saved items in this batch
				c.dirty.Store(item.Key(), true)
			}
		} else {
			// Unrecoverable commit error. Data might be in an inconsistent state.
			// Items were individually saved but commit failed. DB state is uncertain.
			// These items are NOT re-added to dirty list, effectively lost from cache's perspective of being dirty.
			c.Handler.Warn(ctx, fmt.Sprintf("Unrecoverable error from CommitTx, batch data might be inconsistent in DB and will NOT be retried by cache: %v", err), ActionWriteDirty)
			// unCommitted items are not re-added to dirty list.
			// notify Handler of data loss.
			c.Handler.Fail(ctx, err, unCommitted...)
		}
		c.Handler.Warn(ctx, fmt.Sprintf("Error committing transaction: %v failures, %v successes", fail, success), ActionWriteDirty)
		return
	}

	// Transaction committed successfully. Items are no longer dirty.
	c.Handler.Info(ctx, fmt.Sprintf("Completed DB Sync, flushed %d records successfully in transaction.", success), ActionWriteDirty)
}

// ClearDirty forcefully empties the dirty queue.
// This is useful if, for example, the cache has just been loaded from the DB,
// and you want to avoid the overhead of retrying to write it all back.
// Note: This only clears the 'dirty' tracking map. Items might still exist in the
// FIFO eviction queue. If an item cleared from 'dirty' is still in the FIFO queue,
// the evictionProcessor will see it as non-dirty and may evict it.
func (c *LazyWriterCacheLF[K, T]) ClearDirty() {
	c.dirty = xsync.NewTypedMapOf[K, bool](c.hasher) // clear dirty list
}

// Go routine to evict the cache every few seconds to keep it trimmed to the desired size - or there abouts
func (c *LazyWriterCacheLF[K, T]) evictionManager(ctx context.Context) {
	tick := time.NewTicker(c.PurgeFreq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c.evictionProcessor(ctx)
		}
	}
}

// process evictions if the cache is larger than desired.
// Note: If the cache is full of dirty items, this processor will repeatedly
// dequeue a dirty item, find it's dirty, re-enqueue it, and then return.
// This is by design, as dirty items should not be evicted before being written.
// This might appear as a busy-loop for the eviction processor if WriteFreq is too slow
// or DB writes are failing, but it prevents data loss.
func (c *LazyWriterCacheLF[K, T]) evictionProcessor(ctx context.Context) {
	for c.cache.Size() > c.Limit {
		keyToEvict, fifoOk := c.fifo.Peek()
		if !fifoOk {
			// FIFO queue is empty, but cache size > limit.
			// This might happen if all remaining items are dirty and constantly re-queued,
			// or if there's a discrepancy. Stop eviction for this cycle.
			c.Handler.Info(ctx, "Eviction attempted on empty FIFO queue while cache size > limit.", ActionEvict)
			return
		}

		isDirty, dirtyLoaded := c.dirty.Load(keyToEvict)
		if dirtyLoaded && isDirty {
			// Item is dirty, cannot evict. Put it back at the end of the queue.
			// Log if the item is actually still in the cache.
			if item, itemInCache := c.cache.Load(keyToEvict); itemInCache {
				c.Handler.Warn(ctx, "Dirty item at head of purge queue; eviction paused for this cycle.", ActionEvict, item)
			} else {
				// It's unusual for a key to be in 'dirty' but not in 'cache'.
				// It might have been deleted from cache but not yet from dirty list by another process.
				// In this case, just remove it from dirty list as it's not in cache.
				c.Handler.Warn(ctx, fmt.Sprintf("Dirty key %v (not in cache) at head of purge queue; removing from dirty list.", keyToEvict), ActionEvict)
				c.dirty.Delete(keyToEvict) // Remove from dirty as it's not in cache.
				_, _ = c.fifo.Dequeue()    // remove the head of the fifo
				// there is no need to check the dequeued item is also the item we peeked at because
				// evictionProcessor() is only called sequentially from the goroutine loop.
				continue // Try next item from FIFO.
			}
			return // Stop this eviction cycle; wait for next PurgeFreq or for item to become non-dirty.
		}

		// Item is not dirty (or not in dirty map), proceed with eviction and remove from the fifi..
		_, _ = c.fifo.Dequeue()
		// there is no need to check the dequeued head of the fifo item is also the item we peeked at because
		// evictionProcessor() is only called sequentially from the goroutine loop.
		c.cache.Delete(keyToEvict)
		// The key might still be in the dirty map if `dirtyLoaded` was false (meaning it was never marked dirty or cleared).
		// If it was truly non-dirty, it shouldn't be in the dirty map.
		// If `dirtyLoaded` was false, `isDirty` is also false, so this path is correct.
		// If `dirtyLoaded` was true but `isDirty` was false (e.g. marked non-dirty by a successful write),
		// then it's also fine to evict.
		c.Evictions.Add(1)
		// Loop continues to check if cache size is still > Limit.
	}
}

// this is the lazy writer goroutine
func (c *LazyWriterCacheLF[K, T]) lazyWriter(ctx context.Context) {
	ticker := time.NewTicker(c.WriteFreq)
	for {
		select {
		case <-ctx.Done():
			if c.ConfigLF.FlushOnShutdown {
				c.Flush(ctx)
			}
			return
		case <-ticker.C:
			c.saveDirtyToDB(ctx)

		}
	}
}

// Flush forces all dirty items to be written to the database.
// Flush should be called before exiting the application otherwise dirty writes will be lost.
// As the lazy writer is set up with a timer this should only need to be called at exit.
func (c *LazyWriterCacheLF[K, T]) Flush(ctx context.Context) {
	c.saveDirtyToDB(ctx)
}

// Range over all the keys and maps.
//
// As with other Range functions return true to continue iterating or false to stop.
func (c *LazyWriterCacheLF[K, T]) Range(action func(k K, v T) bool) (n int) {
	c.cache.Range(func(k K, v T) bool {
		n++
		return action(k, v)
	})
	return n
}

// Shutdown signals to the cache it should stop any running goroutines.
// This does not Flush the cache first unless ConfigLF.FlushOnShutdown is true.
func (c *LazyWriterCacheLF[K, T]) Shutdown() {
	c.cancel()
}

func (c *LazyWriterCacheLF[K, T]) IsDirty() bool {
	return c.dirty.Size() > 0
}

// CacheAction represents an enumerated type indicating specific actions to be performed in a caching mechanism.
type CacheAction int64

const (
	ActionWriteDirty CacheAction = iota
	ActionEvict
)

func (a CacheAction) String() string {
	switch a {
	case ActionEvict:
		return "evict"
	case ActionWriteDirty:
		return "write-dirty"
	default:
		return "unknown"
	}
}
