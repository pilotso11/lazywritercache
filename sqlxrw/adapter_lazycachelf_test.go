package sqlxrw

import (
	"context"
	"errors"
	"fmt"
	// "os"    // Not used after TestMain removal from this file
	// "strconv" // Not used
	// "strings" // Not used directly in this file's test logic
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/pilotso11/lazywritercache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Struct Implementing CacheableLF ---
type CacheableProduct struct {
	SKU         string  `db:"sku"` // Primary key for the cache
	ProductName string  `db:"name"`
	Price       float64 `db:"price"`
	Version     int     `db:"version"` // For optimistic locking or versioning
}

// Key returns the primary key for the CacheableProduct.
func (p CacheableProduct) Key() string {
	return p.SKU
}

// CopyKeyDataFrom merges database-managed fields from 'from' into 'p'.
func (p CacheableProduct) CopyKeyDataFrom(fromDB lazywritercache.CacheableLF[string]) lazywritercache.CacheableLF[string] {
	from, ok := fromDB.(CacheableProduct)
	if !ok {
		// This should ideally not happen if types are consistent.
		// If fromDB could be a pointer, additional checks might be needed:
		// if fromPtr, isPtr := fromDB.(*CacheableProduct); isPtr { from = *fromPtr } else ...
		panic(fmt.Sprintf("CopyKeyDataFrom: expected CacheableProduct, got %T", fromDB))
	}
	// 'p' is the cached version, 'from' is the DB version.
	p.Version = from.Version // Example: DB updates version on save
	// If SKU could be DB generated (not the case here as it's the key), copy it.
	// p.SKU = from.SKU
	return p
}

var _ lazywritercache.CacheableLF[string] = CacheableProduct{}

// --- Database Setup ---
const productSchema = `
CREATE TABLE IF NOT EXISTS products (
	sku TEXT PRIMARY KEY,
	name TEXT,
	price REAL,
	version INTEGER DEFAULT 1
);`

func setupAdapterTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err, "Failed to connect to sqlite for adapter test")
	_, err = db.Exec(productSchema)
	require.NoError(t, err, "Failed to create product schema for adapter test")
	return db
}

// --- sqlxrw.Config for CacheableProduct ---
func getProductSqlxConfig(t *testing.T) Config[string, CacheableProduct] {
	return Config[string, CacheableProduct]{
		FindQuery: "SELECT sku, name, price, version FROM products WHERE sku = ?",
		// UpsertQuery for SQLite:
		UpsertQuery: "INSERT INTO products (sku, name, price, version) VALUES (?, ?, ?, ?) " +
			"ON CONFLICT(sku) DO UPDATE SET name = excluded.name, price = excluded.price, version = excluded.version + 1",
		KeyExtractor: func(item CacheableProduct) string { return item.SKU },
		UpsertArgsExtractor: func(item CacheableProduct) ([]any, error) {
			// For initial insert, version might be 1. DB increments on update via ON CONFLICT.
			// The version passed here is for the INSERT part.
			return []any{item.SKU, item.ProductName, item.Price, item.Version}, nil
		},
		// Provide fallbacks for constructor validation, though UpsertQuery is primary
		InsertQuery: "INSERT INTO products (sku, name, price, version) VALUES (?, ?, ?, ?)",
		InsertArgsExtractor: func(item CacheableProduct) ([]any, error) {
			return []any{item.SKU, item.ProductName, item.Price, item.Version}, nil
		},
		UpdateQuery: "UPDATE products SET name = ?, price = ?, version = version + 1 WHERE sku = ?",
		UpdateArgsExtractor: func(item CacheableProduct) ([]any, error) {
			return []any{item.ProductName, item.Price, item.SKU}, nil
		},
	}
}

// --- Test Suite ---
func TestLazyCacheWithSqlxAdapter(t *testing.T) {
	db := setupAdapterTestDB(t)
	defer db.Close()

	sqlxRWConfig := getProductSqlxConfig(t)
	sqlxRwImpl, err := NewReadWriterImpl(db, sqlxRWConfig)
	require.NoError(t, err, "Failed to create SqlxReadWriterImpl")

	adapter, err := NewSqlxRwToLazyCacheAdapter[string, CacheableProduct](sqlxRwImpl)
	require.NoError(t, err, "Failed to create SqlxRwToLazyCacheAdapter")

	cacheCfg := lazywritercache.ConfigLF[string, CacheableProduct]{
		Handler:               adapter,
		Limit:                 100,
		LookupOnMiss:          true,
		WriteFreq:             50 * time.Millisecond, // Frequent writes for testing
		PurgeFreq:             100 * time.Millisecond,
		FlushOnShutdown:       true,
		AllowConcurrentWrites: false, // Simpler for testing, ensure sequential writes
	}
	cache := lazywritercache.NewLazyWriterCacheLF[string, CacheableProduct](cacheCfg)
	defer cache.Shutdown() // Ensure flush and shutdown

	ctx := context.Background()

	t.Run("SaveNewItemAndFlush", func(t *testing.T) {
		product1 := CacheableProduct{SKU: "ADPT001", ProductName: "Adapter Widget", Price: 29.99, Version: 1}
		cache.Save(product1)

		// Wait for flush or explicitly flush
		time.Sleep(cacheCfg.WriteFreq * 2) // Wait for potential auto-flush
		cache.Flush(ctx)                   // Explicit flush

		var dbProduct CacheableProduct
		err := db.GetContext(ctx, &dbProduct, "SELECT * FROM products WHERE sku = ?", "ADPT001")
		require.NoError(t, err, "Product ADPT001 not found in DB after save and flush")
		assert.Equal(t, product1.ProductName, dbProduct.ProductName)
		assert.Equal(t, product1.Price, dbProduct.Price)
		assert.Equal(t, product1.Version, dbProduct.Version, "Initial version should match")
	})

	t.Run("LoadItem_CacheHit", func(t *testing.T) {
		// ADPT001 should be in cache from previous sub-test if tests run sequentially in Go < 1.22 or if state persists.
		// To be safe, save it again and ensure it's there.
		productHit := CacheableProduct{SKU: "ADPT_HIT", ProductName: "Cache Hit Product", Price: 10.50, Version: 1}
		cache.Save(productHit)
		cache.Flush(ctx) // Ensure it's in DB for consistency if needed by other parts

		loadedItem, found := cache.Load(ctx, "ADPT_HIT")
		assert.True(t, found, "Expected ADPT_HIT to be found in cache")
		assert.Equal(t, productHit.ProductName, loadedItem.ProductName)
	})

	t.Run("LoadItem_CacheMiss_LookupInDB", func(t *testing.T) {
		// Ensure item is in DB but not in cache initially for this test
		missProduct := CacheableProduct{SKU: "ADPT_MISS", ProductName: "Cache Miss Product", Price: 99.00, Version: 1}
		_, err := db.ExecContext(ctx, "INSERT INTO products (sku, name, price, version) VALUES (?, ?, ?, ?)",
			missProduct.SKU, missProduct.ProductName, missProduct.Price, missProduct.Version)
		require.NoError(t, err, "Failed to insert ADPT_MISS directly into DB")

		// Clear the item from cache if it somehow got there from another test or previous Load.
		// This is tricky as LazyWriterCacheLF doesn't have a direct Delete.
		// For a clean test, use a unique key or re-initialize cache.
		// Here, we rely on it not being in cache due to unique key.

		loadedItem, found := cache.Load(ctx, "ADPT_MISS")
		assert.True(t, found, "Expected ADPT_MISS to be found via DB lookup")
		assert.Equal(t, missProduct.ProductName, loadedItem.ProductName)
		assert.Equal(t, missProduct.Price, loadedItem.Price)

		// Verify it's now in cache
		// Note: The loaded item might have a different version if the DB updated it and CopyKeyDataFrom was effective.
		// Let's check the version from the DB directly for comparison.
		var dbMissProduct CacheableProduct
		err = db.GetContext(ctx, &dbMissProduct, "SELECT * FROM products WHERE sku = ?", "ADPT_MISS")
		require.NoError(t, err)
		assert.Equal(t, dbMissProduct.Version, loadedItem.Version, "Version in cache should match DB after lookup")

		_, foundInCacheAfterLoad := cache.Load(ctx, "ADPT_MISS") // This should be a cache hit now
		assert.True(t, foundInCacheAfterLoad, "ADPT_MISS should be in cache after DB lookup")
	})

	t.Run("UpdateItemAndFlush", func(t *testing.T) {
		productUpdate := CacheableProduct{SKU: "ADPT_UPD", ProductName: "Product to Update", Price: 50.00, Version: 1}
		cache.Save(productUpdate)
		cache.Flush(ctx) // Initial save

		// Load, modify, save
		loaded, found := cache.Load(ctx, "ADPT_UPD")
		require.True(t, found, "Failed to load product ADPT_UPD from cache for update")

		// The version loaded from cache should be the DB version if lookup happened, or initial if direct save.
		// If it was just saved and flushed, its version in cache is 1. DB version is 1.
		// If it was loaded by miss, its version is DB version.
		// Let's check current DB version to be sure.
		var dbProductBeforeUpdate CacheableProduct
		err := db.GetContext(ctx, &dbProductBeforeUpdate, "SELECT * FROM products WHERE sku = ?", "ADPT_UPD")
		require.NoError(t, err)

		loaded.ProductName = "Updated Product Name"
		loaded.Price = 55.55
		// Important: The 'loaded' item version is what's in cache.
		// The UpsertArgsExtractor will use this version for the INSERT part of ON CONFLICT.
		// The DB's ON CONFLICT clause will then do `version = excluded.version + 1`.
		// So, if loaded.Version is 1, excluded.version becomes 1, and new DB version becomes 2.
		// If CopyKeyDataFrom is not called or cache is not updated after this Save,
		// the cache might hold stale version. But LazyCacheLF should update it.

		cache.Save(loaded) // This 'loaded' still has its original version from cache
		cache.Flush(ctx)

		var dbProductAfterUpdate CacheableProduct
		err = db.GetContext(ctx, &dbProductAfterUpdate, "SELECT * FROM products WHERE sku = ?", "ADPT_UPD")
		require.NoError(t, err)
		assert.Equal(t, "Updated Product Name", dbProductAfterUpdate.ProductName)
		assert.Equal(t, 55.55, dbProductAfterUpdate.Price)
		// Version in DB should be dbProductBeforeUpdate.Version + 1 due to "version = excluded.version + 1"
		// and excluded.version being the version from the item passed to UpsertArgsExtractor.
		assert.Equal(t, dbProductBeforeUpdate.Version+1, dbProductAfterUpdate.Version, "Version should have increased by 1 in DB")

		// Verify cache holds the new DB version after save & flush due to CopyKeyDataFrom
		finalCachedItem, found := cache.Load(ctx, "ADPT_UPD")
		require.True(t, found, "Item ADPT_UPD not found in cache after update and flush")
		assert.Equal(t, dbProductAfterUpdate.Version, finalCachedItem.Version, "Cached item version should match the new DB version")
	})

	t.Run("IsRecoverable_BasicChecks", func(t *testing.T) {
		// Test the adapter's IsRecoverable directly
		assert.False(t, adapter.IsRecoverable(ctx, nil), "nil error should not be recoverable")
		assert.True(t, adapter.IsRecoverable(ctx, errors.New("database deadlock detected")), "deadlock should be recoverable")
		assert.True(t, adapter.IsRecoverable(ctx, errors.New("SQLITE_BUSY: database is locked")), "sqlite busy should be recoverable")
		assert.False(t, adapter.IsRecoverable(ctx, errors.New("unique constraint failed")), "constraint violation should not be recoverable")
	})
}

// TestMain function was removed from this file to avoid "multiple definitions of TestMain" error,
// as sqlxrw_test.go already contains a TestMain. A package can only have one TestMain.
// func TestMain(m *testing.M) {
//     os.Exit(m.Run())
// }
