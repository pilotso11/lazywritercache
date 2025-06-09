package sqlxrw

import (
	"context"
	"database/sql"
	"errors"
	// "fmt" // Removed unused import
	"os"
	"strings" // Added import
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// TestItem is a sample struct for testing
type TestItem struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Value string `db:"value"`
}

// TestItemStrID is a struct for testing finding by string ID
type TestItemStrID struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	Value string `db:"value"`
}

const schema = `
CREATE TABLE IF NOT EXISTS items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	value TEXT
);`

const schemaStrID = `
CREATE TABLE IF NOT EXISTS items_str_id (
    id TEXT PRIMARY KEY,
    name TEXT,
    value TEXT
);`

// setupDB initializes an in-memory SQLite database for testing
func setupDB(t *testing.T, customSchema ...string) *sqlx.DB {
	// Using a file-based DB for easier inspection if needed, but ensure it's cleaned up.
	// Or use "file::memory:?cache=shared" for a shared in-memory connection if multiple goroutines in a test need it.
	// For most unit tests, a fresh in-memory DB per test is fine.
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to sqlite: %v", err)
	}

	actualSchema := schema
	if len(customSchema) > 0 {
		actualSchema = customSchema[0]
	}

	_, err = db.Exec(actualSchema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db
}

// --- Config for TestItem (int key) ---
func getKey(item TestItem) int { return item.ID }
func getInsertArgs(item TestItem) ([]any, error) {
	return []any{item.Name, item.Value}, nil // ID is autoincrement
}
func getUpdateArgs(item TestItem) ([]any, error) {
	return []any{item.Name, item.Value, item.ID}, nil
}

// --- Config for TestItemStrID (string key) ---
func getKeyStr(item TestItemStrID) string { return item.ID }
func getInsertArgsStr(item TestItemStrID) ([]any, error) {
	return []any{item.ID, item.Name, item.Value}, nil
}
func getUpdateArgsStr(item TestItemStrID) ([]any, error) {
	return []any{item.Name, item.Value, item.ID}, nil
}

func TestNewReadWriterImpl_Validation(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	validConfig := Config[int, TestItem]{
		FindQuery:           "SELECT * FROM items WHERE id = ?",
		InsertQuery:         "INSERT INTO items (name, value) VALUES (?, ?)",
		UpdateQuery:         "UPDATE items SET name = ?, value = ? WHERE id = ?",
		KeyExtractor:        getKey,
		InsertArgsExtractor: getInsertArgs,
		UpdateArgsExtractor: getUpdateArgs,
	}

	_, err := NewReadWriterImpl(db, validConfig)
	if err != nil {
		t.Errorf("Expected NewReadWriterImpl with valid config to succeed, but got error: %v", err)
	}

	// Test missing DB
	_, err = NewReadWriterImpl[int, TestItem](nil, validConfig)
	if err == nil {
		t.Error("Expected error for nil DB, got nil")
	}

	// Test missing FindQuery
	badConf1 := validConfig
	badConf1.FindQuery = ""
	_, err = NewReadWriterImpl(db, badConf1)
	if err == nil {
		t.Error("Expected error for missing FindQuery, got nil")
	}

	// Test missing InsertQuery (when no UpsertQuery)
	badConf2 := validConfig
	badConf2.InsertQuery = ""
	_, err = NewReadWriterImpl(db, badConf2)
	if err == nil {
		t.Error("Expected error for missing InsertQuery without UpsertQuery, got nil")
	}

	// Test missing KeyExtractor (when UpdateQuery is present and no UpsertQuery)
	badConf3 := validConfig
	badConf3.KeyExtractor = nil
	_, err = NewReadWriterImpl(db, badConf3)
	if err == nil {
		t.Error("Expected error for missing KeyExtractor with UpdateQuery, got nil")
	}

	// Test UpsertQuery without UpsertArgsExtractor and without InsertArgsExtractor
	badConf4 := validConfig
	badConf4.InsertQuery = "" // Remove normal insert/update
	badConf4.UpdateQuery = ""
	badConf4.InsertArgsExtractor = nil
	badConf4.UpsertQuery = "INSERT OR REPLACE INTO items (id, name, value) VALUES (?, ?, ?)"
	badConf4.UpsertArgsExtractor = nil // Explicitly nil
	_, err = NewReadWriterImpl(db, badConf4)
	if err == nil {
		t.Error("Expected error for UpsertQuery without any valid arg extractor, got nil")
	}
}

func TestReadWriterImpl_Find(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	config := Config[int, TestItem]{
		FindQuery:           "SELECT * FROM items WHERE id = ?",
		InsertQuery:         "INSERT INTO items (name, value) VALUES (?, ?)",     // Needed for New validation
		UpdateQuery:         "UPDATE items SET name = ?, value = ? WHERE id = ?", // Needed for New validation
		KeyExtractor:        getKey,
		InsertArgsExtractor: getInsertArgs,
		UpdateArgsExtractor: getUpdateArgs,
	}
	rw, _ := NewReadWriterImpl(db, config)

	// Insert a test item directly
	res, err := db.Exec("INSERT INTO items (name, value) VALUES (?, ?)", "test_find", "initial_val")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	id, _ := res.LastInsertId()

	// Test Find found
	item, err := rw.Find(context.Background(), int(id), nil)
	if err != nil {
		t.Errorf("Find failed: %v", err)
	}
	if item.ID != int(id) || item.Name != "test_find" || item.Value != "initial_val" {
		t.Errorf("Find returned incorrect item: %+v", item)
	}

	// Test Find not found
	_, err = rw.Find(context.Background(), 99999, nil)
	if err == nil {
		t.Error("Expected error for Find not found, got nil")
	} else {
		// Check if the error indicates "not found"
		// This is a bit brittle as it depends on the exact error message from sqlxrw.
		// A more robust way would be to define custom error types/codes.
		if !errors.Is(err, sql.ErrNoRows) && !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, but got: %v", err)
		}
	}
}

func TestReadWriterImpl_Save_InsertUpdate(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	config := Config[int, TestItem]{
		FindQuery:           "SELECT * FROM items WHERE id = ?",
		InsertQuery:         "INSERT INTO items (name, value) VALUES (?, ?)",
		UpdateQuery:         "UPDATE items SET name = ?, value = ? WHERE id = ?",
		KeyExtractor:        getKey,
		InsertArgsExtractor: getInsertArgs,
		UpdateArgsExtractor: getUpdateArgs,
	}
	rw, _ := NewReadWriterImpl(db, config)
	ctx := context.Background()

	// Test Insert
	newItem := TestItem{Name: "save_insert", Value: "value1"}
	err := rw.Save(ctx, newItem, nil)
	if err != nil {
		t.Fatalf("Save (insert) failed: %v", err)
	}

	// Verify insert by finding (need to get ID)
	// This is tricky with autoincrement without fetching ID back from Save.
	// For now, let's assume ID 1 for the first insert in a clean DB.
	// A better Save would return the ID or the saved item.
	var insertedItem TestItem
	err = db.Get(&insertedItem, "SELECT * FROM items WHERE name = ?", "save_insert")
	if err != nil {
		t.Fatalf("Failed to retrieve inserted item for verification: %v", err)
	}
	if insertedItem.Name != "save_insert" || insertedItem.Value != "value1" {
		t.Errorf("Inserted item has wrong data: %+v", insertedItem)
	}

	// Test Update
	updatedItem := TestItem{ID: insertedItem.ID, Name: "save_update", Value: "value2"}
	err = rw.Save(ctx, updatedItem, nil)
	if err != nil {
		t.Fatalf("Save (update) failed: %v", err)
	}

	foundItem, err := rw.Find(ctx, updatedItem.ID, nil)
	if err != nil {
		t.Fatalf("Find after update failed: %v", err)
	}
	if foundItem.Name != "save_update" || foundItem.Value != "value2" {
		t.Errorf("Update did not persist. Expected name 'save_update', value 'value2', got: %+v", foundItem)
	}
}

func TestReadWriterImpl_Save_UpsertQuery(t *testing.T) {
	db := setupDB(t) // Uses default int ID schema "items"
	defer db.Close()

	upsertConf := Config[int, TestItem]{
		FindQuery: "SELECT * FROM items WHERE id = ?",
		// For SQLite, an Upsert can be INSERT ... ON CONFLICT DO UPDATE
		// Note: This requires ID to be part of insert normally.
		// We'll adjust TestItem and args for this specific test.
		UpsertQuery:  "INSERT INTO items (id, name, value) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = excluded.name, value = excluded.value",
		KeyExtractor: func(item TestItem) int { return item.ID }, // Still needed for some internal logic or if Find is used
		UpsertArgsExtractor: func(item TestItem) ([]any, error) {
			return []any{item.ID, item.Name, item.Value}, nil
		},
		// Provide fallbacks for constructor validation if needed, though Upsert should take precedence
		InsertQuery:         "INSERT INTO items (id, name, value) VALUES (?,?,?)", // Fallback
		InsertArgsExtractor: func(item TestItem) ([]any, error) { return []any{item.ID, item.Name, item.Value}, nil },
		UpdateQuery:         "UPDATE items SET name=?, value=? WHERE id=?", // Fallback
		UpdateArgsExtractor: func(item TestItem) ([]any, error) { return []any{item.Name, item.Value, item.ID}, nil },
	}

	rw, err := NewReadWriterImpl(db, upsertConf)
	if err != nil {
		t.Fatalf("Failed to create ReadWriter with upsert config: %v", err)
	}
	ctx := context.Background()

	// Upsert - Insert
	item1 := TestItem{ID: 100, Name: "upsert_insert", Value: "val_ins"}
	err = rw.Save(ctx, item1, nil)
	if err != nil {
		t.Fatalf("Save (upsert-insert) failed: %v", err)
	}
	found1, err := rw.Find(ctx, 100, nil)
	if err != nil {
		t.Fatalf("Find after upsert-insert failed: %v", err)
	}
	if found1.Name != "upsert_insert" || found1.Value != "val_ins" {
		t.Errorf("Upsert-insert resulted in wrong data: %+v", found1)
	}

	// Upsert - Update
	item2 := TestItem{ID: 100, Name: "upsert_update", Value: "val_upd"}
	err = rw.Save(ctx, item2, nil)
	if err != nil {
		t.Fatalf("Save (upsert-update) failed: %v", err)
	}
	found2, err := rw.Find(ctx, 100, nil)
	if err != nil {
		t.Fatalf("Find after upsert-update failed: %v", err)
	}
	if found2.Name != "upsert_update" || found2.Value != "val_upd" {
		t.Errorf("Upsert-update resulted in wrong data: %+v", found2)
	}
}

func TestReadWriterImpl_Save_UpsertQuery_StringKey(t *testing.T) {
	db := setupDB(t, schemaStrID) // Use string ID schema
	defer db.Close()

	upsertConfStr := Config[string, TestItemStrID]{
		FindQuery:    "SELECT * FROM items_str_id WHERE id = ?",
		UpsertQuery:  "INSERT INTO items_str_id (id, name, value) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = excluded.name, value = excluded.value",
		KeyExtractor: getKeyStr,
		UpsertArgsExtractor: func(item TestItemStrID) ([]any, error) {
			return []any{item.ID, item.Name, item.Value}, nil
		},
		InsertQuery:         "INSERT INTO items_str_id (id, name, value) VALUES (?,?,?)",
		InsertArgsExtractor: getInsertArgsStr,
		UpdateQuery:         "UPDATE items_str_id SET name=?, value=? WHERE id=?",
		UpdateArgsExtractor: getUpdateArgsStr,
	}

	rw, err := NewReadWriterImpl(db, upsertConfStr)
	if err != nil {
		t.Fatalf("Failed to create ReadWriter with string key upsert config: %v", err)
	}
	ctx := context.Background()

	// Upsert - Insert (string key)
	item1 := TestItemStrID{ID: "key1", Name: "upsert_str_insert", Value: "val_str_ins"}
	err = rw.Save(ctx, item1, nil)
	if err != nil {
		t.Fatalf("Save (upsert-insert string key) failed: %v", err)
	}
	found1, err := rw.Find(ctx, "key1", nil)
	if err != nil {
		t.Fatalf("Find after upsert-insert string key failed: %v", err)
	}
	if found1.Name != "upsert_str_insert" || found1.Value != "val_str_ins" {
		t.Errorf("Upsert-insert string key resulted in wrong data: %+v", found1)
	}

	// Upsert - Update (string key)
	item2 := TestItemStrID{ID: "key1", Name: "upsert_str_update", Value: "val_str_upd"}
	err = rw.Save(ctx, item2, nil)
	if err != nil {
		t.Fatalf("Save (upsert-update string key) failed: %v", err)
	}
	found2, err := rw.Find(ctx, "key1", nil)
	if err != nil {
		t.Fatalf("Find after upsert-update string key failed: %v", err)
	}
	if found2.Name != "upsert_str_update" || found2.Value != "val_str_upd" {
		t.Errorf("Upsert-update string key resulted in wrong data: %+v", found2)
	}
}

func TestReadWriterImpl_Transactions(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	config := Config[int, TestItem]{
		FindQuery:           "SELECT * FROM items WHERE id = ?",
		InsertQuery:         "INSERT INTO items (name, value) VALUES (?, ?)",
		UpdateQuery:         "UPDATE items SET name = ?, value = ? WHERE id = ?",
		KeyExtractor:        getKey,
		InsertArgsExtractor: getInsertArgs,
		UpdateArgsExtractor: getUpdateArgs,
	}
	rw, _ := NewReadWriterImpl(db, config)
	ctx := context.Background()

	// Test Commit
	tx, err := rw.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}
	itemToCommit := TestItem{Name: "commit_test", Value: "val_commit"}
	// Simulate insert within transaction (Save needs to handle ID for real use)
	// For this test, we'll use raw exec in tx then try to find via rw.Find with tx
	_, err = tx.ExecContext(ctx, "INSERT INTO items (name, value) VALUES (?, ?)", itemToCommit.Name, itemToCommit.Value)
	if err != nil {
		t.Fatalf("tx.ExecContext for commit test failed: %v", err)
	}
	var committedItem TestItem
	err = tx.GetContext(ctx, &committedItem, "SELECT * FROM items WHERE name = ?", itemToCommit.Name)
	if err != nil {
		t.Fatalf("GetContext within tx for commit test failed: %v", err)
	}

	err = rw.CommitTx(ctx, tx)
	if err != nil {
		t.Fatalf("CommitTx failed: %v", err)
	}

	foundAfterCommit, err := rw.Find(ctx, committedItem.ID, nil) // Find without tx
	if err != nil {
		t.Errorf("Find after commit failed: %v", err)
	}
	if foundAfterCommit.Name != itemToCommit.Name {
		t.Errorf("Item not found or incorrect after commit. Expected %s, Got %s", itemToCommit.Name, foundAfterCommit.Name)
	}

	// Test Rollback
	tx, err = rw.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx for rollback test failed: %v", err)
	}
	itemToRollback := TestItem{Name: "rollback_test", Value: "val_rollback"}
	_, err = tx.ExecContext(ctx, "INSERT INTO items (name, value) VALUES (?, ?)", itemToRollback.Name, itemToRollback.Value)
	if err != nil {
		t.Fatalf("tx.ExecContext for rollback test failed: %v", err)
	}
	// Get the ID to ensure we're checking for absence later
	var tempRolledbackItem TestItem
	err = tx.GetContext(ctx, &tempRolledbackItem, "SELECT * FROM items WHERE name = ?", itemToRollback.Name)
	if err != nil {
		t.Fatalf("GetContext within tx for rollback test failed: %v", err)
	}

	err = rw.RollbackTx(ctx, tx)
	if err != nil {
		t.Fatalf("RollbackTx failed: %v", err)
	}

	_, err = rw.Find(ctx, tempRolledbackItem.ID, nil) // Should not be found
	if err == nil {
		t.Error("Expected error when finding rolled-back item, but it was found.")
	}
}

// Helper to import strings for the test
func stringsContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestMain(m *testing.M) {
	// This is to ensure strings.Contains is linked for the subtask if it's not used elsewhere directly in test code
	// but by helper functions.
	_ = stringsContains("", "")
	os.Exit(m.Run())
}
