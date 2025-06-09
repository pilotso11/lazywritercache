package sqlxrw_test // Use _test package for examples that are more like integration tests

import (
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for example
	"github.com/pilotso11/lazywritercache/sqlxrw"
)

// ExampleProduct is a sample struct for the example.
type ExampleProduct struct {
	SKU   string  `db:"sku"`
	Name  string  `db:"name"`
	Price float64 `db:"price"`
}

// Setup a temporary in-memory SQLite DB for the example.
func setupExampleDB() *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("example setup: failed to connect to sqlite: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS products (
		sku TEXT PRIMARY KEY,
		name TEXT,
		price REAL
	);`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("example setup: failed to create schema: %v", err)
	}
	return db
}

func ExampleReadWriterImpl_saveAndFind() {
	db := setupExampleDB()
	defer db.Close()

	// Configuration for ExampleProduct
	config := sqlxrw.Config[string, ExampleProduct]{
		FindQuery:   "SELECT * FROM products WHERE sku = ?",
		UpsertQuery: "INSERT INTO products (sku, name, price) VALUES (?, ?, ?) ON CONFLICT(sku) DO UPDATE SET name = excluded.name, price = excluded.price",
		KeyExtractor: func(item ExampleProduct) string {
			return item.SKU
		},
		UpsertArgsExtractor: func(item ExampleProduct) ([]any, error) {
			return []any{item.SKU, item.Name, item.Price}, nil
		},
		// Provide fallbacks for constructor validation, though UpsertQuery is primary for this example
		InsertQuery:         "INSERT INTO products (sku, name, price) VALUES (?,?,?)",
		InsertArgsExtractor: func(item ExampleProduct) ([]any, error) { return []any{item.SKU, item.Name, item.Price}, nil },
		UpdateQuery:         "UPDATE products SET name=?, price=? WHERE sku=?",
		UpdateArgsExtractor: func(item ExampleProduct) ([]any, error) { return []any{item.Name, item.Price, item.SKU}, nil },
	}

	// Create the ReadWriter
	// Using nil for logger to use default logger
	rw, err := sqlxrw.NewReadWriterImpl(db, config, nil)
	if err != nil {
		log.Fatalf("example: failed to create ReadWriter: %v", err)
	}

	ctx := context.Background()
	product1 := ExampleProduct{SKU: "EX001", Name: "Awesome Widget", Price: 19.99}

	// Save (insert) the product
	if err := rw.Save(ctx, product1, nil); err != nil {
		log.Fatalf("example: Save (insert) failed: %v", err)
	}
	fmt.Printf("Saved: %s, %.2f\n", product1.Name, product1.Price)

	// Find the product
	foundProduct, err := rw.Find(ctx, "EX001", nil)
	if err != nil {
		log.Fatalf("example: Find failed: %v", err)
	}
	fmt.Printf("Found: %s, %.2f\n", foundProduct.Name, foundProduct.Price)

	// Save (update) the product
	product1.Price = 21.99
	if err := rw.Save(ctx, product1, nil); err != nil {
		log.Fatalf("example: Save (update) failed: %v", err)
	}
	fmt.Printf("Saved (updated): %s, %.2f\n", product1.Name, product1.Price)

	foundUpdatedProduct, err := rw.Find(ctx, "EX001", nil)
	if err != nil {
		log.Fatalf("example: Find after update failed: %v", err)
	}
	fmt.Printf("Found (updated): %s, %.2f\n", foundUpdatedProduct.Name, foundUpdatedProduct.Price)

	// Output:
	// Saved: Awesome Widget, 19.99
	// Found: Awesome Widget, 19.99
	// Saved (updated): Awesome Widget, 21.99
	// Found (updated): Awesome Widget, 21.99
}

func ExampleReadWriterImpl_transactions() {
	db := setupExampleDB()
	defer db.Close()

	config := sqlxrw.Config[string, ExampleProduct]{
		FindQuery:    "SELECT * FROM products WHERE sku = ?",
		UpsertQuery:  "INSERT INTO products (sku, name, price) VALUES (?, ?, ?) ON CONFLICT(sku) DO UPDATE SET name = excluded.name, price = excluded.price",
		KeyExtractor: func(item ExampleProduct) string { return item.SKU },
		UpsertArgsExtractor: func(item ExampleProduct) ([]any, error) {
			return []any{item.SKU, item.Name, item.Price}, nil
		},
		InsertQuery:         "INSERT INTO products (sku, name, price) VALUES (?,?,?)",
		InsertArgsExtractor: func(item ExampleProduct) ([]any, error) { return []any{item.SKU, item.Name, item.Price}, nil },
		UpdateQuery:         "UPDATE products SET name=?, price=? WHERE sku=?",
		UpdateArgsExtractor: func(item ExampleProduct) ([]any, error) { return []any{item.Name, item.Price, item.SKU}, nil },
	}
	// Using nil for logger to use default logger
	rw, err := sqlxrw.NewReadWriterImpl(db, config, nil)
	if err != nil {
		log.Fatalf("example: failed to create ReadWriter: %v", err)
	}

	ctx := context.Background()
	productCommit := ExampleProduct{SKU: "TXC001", Name: "Commit Product", Price: 10.00}
	productRollback := ExampleProduct{SKU: "TXR001", Name: "Rollback Product", Price: 20.00}

	// Transaction 1: Commit
	tx, err := rw.BeginTx(ctx)
	if err != nil {
		log.Fatalf("example: BeginTx (commit) failed: %v", err)
	}
	if err := rw.Save(ctx, productCommit, tx); err != nil {
		// If Save fails, we should rollback
		if rbErr := rw.RollbackTx(ctx, tx); rbErr != nil {
			log.Fatalf("example: Save failed and Rollback also failed: Save err: %v, Rollback err: %v", err, rbErr)
		}
		log.Fatalf("example: Save within tx (commit) failed: %v", err)
	}
	if err := rw.CommitTx(ctx, tx); err != nil {
		log.Fatalf("example: CommitTx failed: %v", err)
	}
	fmt.Println("Committed TXC001")

	foundCommitted, err := rw.Find(ctx, "TXC001", nil)
	if err != nil {
		fmt.Printf("Error finding committed product: %v\n", err)
	} else {
		fmt.Printf("Found after commit: %s\n", foundCommitted.Name)
	}

	// Transaction 2: Rollback
	tx, err = rw.BeginTx(ctx)
	if err != nil {
		log.Fatalf("example: BeginTx (rollback) failed: %v", err)
	}
	if err := rw.Save(ctx, productRollback, tx); err != nil {
		// If Save fails, we should rollback
		if rbErr := rw.RollbackTx(ctx, tx); rbErr != nil {
			log.Fatalf("example: Save failed and Rollback also failed: Save err: %v, Rollback err: %v", err, rbErr)
		}
		log.Fatalf("example: Save within tx (rollback) failed: %v", err)
	}
	if err := rw.RollbackTx(ctx, tx); err != nil {
		log.Fatalf("example: RollbackTx failed: %v", err)
	}
	fmt.Println("Rolled back TXR001")

	_, err = rw.Find(ctx, "TXR001", nil)
	if err != nil {
		// Expected: "sqlxrw: item with key TXR001 not found: sql: no rows in result set"
		// We'll just check that an error occurred, indicating not found.
		fmt.Printf("Product TXR001 not found after rollback (expected).\n")
	} else {
		fmt.Println("Product TXR001 found after rollback (unexpected).")
	}

	// Output:
	// Committed TXC001
	// Found after commit: Commit Product
	// Rolled back TXR001
	// Product TXR001 not found after rollback (expected).
}
