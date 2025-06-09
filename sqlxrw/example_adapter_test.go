package sqlxrw_test // Use _test package for examples

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/pilotso11/lazywritercache"
	"github.com/pilotso11/lazywritercache/sqlxrw"
)

// ExampleCacheableItem for the adapter example.
type ExampleCacheableItem struct {
	ID      string `db:"id"` // Primary key
	Content string `db:"content"`
	Version int    `db:"version"` // For optimistic locking or versioning
}

// Key returns the primary key for the ExampleCacheableItem.
func (i ExampleCacheableItem) Key() string {
	return i.ID
}

// CopyKeyDataFrom merges database-managed fields from 'fromDB' into 'i'.
// 'fromDB' is expected to be the state of the item as read from the database.
func (i ExampleCacheableItem) CopyKeyDataFrom(fromDB lazywritercache.CacheableLF[string]) lazywritercache.CacheableLF[string] {
	from, ok := fromDB.(ExampleCacheableItem)
	if !ok {
		// This panic indicates a type mismatch, which should not happen if the cache
		// and its handler are correctly typed and used.
		panic(fmt.Sprintf("CopyKeyDataFrom: expected ExampleCacheableItem, got %T", fromDB))
	}
	// Update the receiver (i) with fields that are managed or modified by the database.
	// In this example, only 'Version' is managed by the DB during updates (auto-increments).
	i.Version = from.Version
	return i
}

// Ensure ExampleCacheableItem implements CacheableLF.
var _ lazywritercache.CacheableLF[string] = ExampleCacheableItem{}

// setupAdapterExampleDB initializes an in-memory SQLite database for the adapter example.
func setupAdapterExampleDB() *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("adapter example setup: failed to connect to sqlite: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS cacheable_items (
		id TEXT PRIMARY KEY,
		content TEXT,
		version INTEGER DEFAULT 1
	);`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("adapter example setup: failed to create schema: %v", err)
	}
	return db
}

func ExampleSqlxRwToLazyCacheAdapter_integrationWithLazyCache() {
	db := setupAdapterExampleDB()
	defer db.Close()
	ctx := context.Background()

	// 1. Configure sqlxrw.ReadWriterImpl
	sqlxConfig := sqlxrw.Config[string, ExampleCacheableItem]{
		FindQuery: "SELECT id, content, version FROM cacheable_items WHERE id = ?",
		// UpsertQuery for SQLite (increments version on conflict)
		UpsertQuery: "INSERT INTO cacheable_items (id, content, version) VALUES (?, ?, ?) " +
			"ON CONFLICT(id) DO UPDATE SET content = excluded.content, version = version + 1",
		KeyExtractor: func(item ExampleCacheableItem) string { return item.ID },
		UpsertArgsExtractor: func(item ExampleCacheableItem) ([]any, error) {
			// Version passed here is for the INSERT part. DB increments on UPDATE via ON CONFLICT.
			return []any{item.ID, item.Content, item.Version}, nil
		},
		// Provide fallbacks for constructor validation (though UpsertQuery is primary here)
		InsertQuery: "INSERT INTO cacheable_items (id, content, version) VALUES (?, ?, ?)",
		InsertArgsExtractor: func(item ExampleCacheableItem) ([]any, error) {
			return []any{item.ID, item.Content, item.Version}, nil
		},
		UpdateQuery: "UPDATE cacheable_items SET content = ?, version = version + 1 WHERE id = ?",
		UpdateArgsExtractor: func(item ExampleCacheableItem) ([]any, error) {
			// This UpdateQuery is a fallback if UpsertQuery wasn't available/used.
			// It also increments version.
			return []any{item.Content, item.ID}, nil
		},
	}

	sqlxReadWriter, err := sqlxrw.NewReadWriterImpl(db, sqlxConfig)
	if err != nil {
		log.Fatalf("adapter example: failed to create NewReadWriterImpl: %v", err)
	}

	// 2. Create the Adapter
	adapter, err := sqlxrw.NewSqlxRwToLazyCacheAdapter[string, ExampleCacheableItem](sqlxReadWriter)
	if err != nil {
		log.Fatalf("adapter example: failed to create NewSqlxRwToLazyCacheAdapter: %v", err)
	}

	// 3. Configure and Create LazyWriterCacheLF
	cacheCfg := lazywritercache.ConfigLF[string, ExampleCacheableItem]{
		Handler:         adapter,
		LookupOnMiss:    true,
		WriteFreq:       100 * time.Millisecond, // Write frequently for example responsiveness
		PurgeFreq:       500 * time.Millisecond,
		Limit:           1000,
		FlushOnShutdown: true, // Good practice
	}
	lazyCache := lazywritercache.NewLazyWriterCacheLF[string, ExampleCacheableItem](cacheCfg)
	defer lazyCache.Shutdown()

	// 4. Use the LazyWriterCacheLF
	// Save a new item
	item1 := ExampleCacheableItem{ID: "EX_ITEM_001", Content: "Initial content for item 1", Version: 1}
	lazyCache.Save(item1) // Item is now in cache, marked dirty
	fmt.Printf("Saved to cache: ID=%s, Content='%s', Version=%d\n", item1.ID, item1.Content, item1.Version)

	// Flush the cache to ensure it's written to DB
	lazyCache.Flush(ctx)
	fmt.Println("Cache flushed.")

	// Load the item
	// If LookupOnMiss is true, this will hit DB if not in cache.
	// After flush, it should be in DB. Cache state depends on eviction.
	// The fix in lazywritercache ensures CopyKeyDataFrom gets the DB state after save.
	loadedItem1, found := lazyCache.Load(ctx, "EX_ITEM_001")
	if !found {
		fmt.Println("Item EX_ITEM_001 not found after save and flush!") // Should not happen
	} else {
		// Version should be 1 if it was a fresh insert.
		fmt.Printf("Loaded from cache/db: ID=%s, Content='%s', Version=%d\n", loadedItem1.ID, loadedItem1.Content, loadedItem1.Version)
	}

	// Update the item
	// loadedItem1 here has Version 1.
	loadedItem1.Content = "Updated content for item 1"
	lazyCache.Save(loadedItem1) // Save again. The item in cache now has Content "Updated..." and Version 1.
	fmt.Printf("Updated in cache: ID=%s, Content='%s', Version=%d (before flush)\n", loadedItem1.ID, loadedItem1.Content, loadedItem1.Version)

	lazyCache.Flush(ctx) // This will trigger save. DB will set version to 1 (from loadedItem1.Version) + 1 = 2.
	fmt.Println("Cache flushed after update.")

	// Load again. The cache should have been updated with version 2 after the flush.
	reloadedItem1, found := lazyCache.Load(ctx, "EX_ITEM_001")
	if !found {
		fmt.Println("Item EX_ITEM_001 not found after update and flush!") // Should not happen
	} else {
		fmt.Printf("Reloaded after update: ID=%s, Content='%s', Version=%d (after flush)\n", reloadedItem1.ID, reloadedItem1.Content, reloadedItem1.Version)
	}

	// Example of cache miss that loads from DB
	directDbItem := ExampleCacheableItem{ID: "EX_ITEM_002", Content: "Direct DB content", Version: 5}
	_, err = db.ExecContext(ctx, "INSERT INTO cacheable_items (id, content, version) VALUES (?, ?, ?)",
		directDbItem.ID, directDbItem.Content, directDbItem.Version)
	if err != nil {
		log.Fatalf("adapter example: failed to seed direct DB item: %v", err)
	}

	loadedDbItem, found := lazyCache.Load(ctx, "EX_ITEM_002") // Cache miss, should load from DB
	if !found {
		fmt.Println("Item EX_ITEM_002 (direct DB) not found on cache miss!") // Should not happen
	} else {
		// Version should be 5 as loaded from DB.
		fmt.Printf("Loaded on cache miss: ID=%s, Content='%s', Version=%d\n", loadedDbItem.ID, loadedDbItem.Content, loadedDbItem.Version)
	}

	// Output:
	// Saved to cache: ID=EX_ITEM_001, Content='Initial content for item 1', Version=1
	// Cache flushed.
	// Loaded from cache/db: ID=EX_ITEM_001, Content='Initial content for item 1', Version=1
	// Updated in cache: ID=EX_ITEM_001, Content='Updated content for item 1', Version=1 (before flush)
	// Cache flushed after update.
	// Reloaded after update: ID=EX_ITEM_001, Content='Updated content for item 1', Version=2 (after flush)
	// Loaded on cache miss: ID=EX_ITEM_002, Content='Direct DB content', Version=5
}
