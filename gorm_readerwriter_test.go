package lazywritercache

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type testDBItem struct {
	ID    uint `gorm:"primarykey"`
	Value string
}

var testColumns = []string{"id", "value"}

func (i testDBItem) Key() interface{} {
	return i.Value
}

func (i testDBItem) CopyKeyDataFrom(from Cacheable) Cacheable {
	i.Value = from.Key().(string)
	return i
}
func (i testDBItem) String() string {
	return i.Value
}

func newTestDBItem(key string) testDBItem {
	return testDBItem{
		Value: key,
	}
}

func TestGormReaderWriter_WithTransactions_True_Integration(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	// defer db.Close() // The cache shutdown handles this

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Errorf("Error setting up gorm mock: %v", err)
		t.Fail()
		return
	}

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	gormRW.UseTransactions = true // Explicitly set for clarity
	cfg := NewDefaultConfig[string, testDBItem](gormRW)
	cfg.WriteFreq = 100 * time.Millisecond
	cache := NewLazyWriterCache[string, testDBItem](cfg)
	defer func(db *sql.DB) {
		cache.Shutdown()
		_ = db.Close()
	}(db)

	// First dirty hit should return an empty item
	mock.ExpectQuery(`SELECT \* FROM "test_db_items" WHERE "test_db_items"\."value" = \$1 LIMIT \$2`).WithArgs("item1", 1).WillReturnRows(sqlmock.NewRows(testColumns))
	item1, ok, err := cache.GetAndRelease("item1")
	assert.NoError(t, err)
	assert.False(t, ok, "item should not be found")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "test_db_items" WHERE "test_db_items"\."value" = \$1 LIMIT \$2`).WithArgs("item1", 1).WillReturnRows(sqlmock.NewRows(testColumns))
	mock.ExpectQuery(`INSERT INTO "test_db_items" \("value"\) VALUES \(\$1\) RETURNING "id"`).WithArgs("item1").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()
	item1.Value = "item1"
	err = cache.Lock()
	assert.NoError(t, err)
	cache.Save(item1)
	assert.Equal(t, 1, len(cache.dirty), "Cache is dirty")
	err = cache.Unlock()
	assert.NoError(t, err)

	time.Sleep(1 * time.Second) // time to write the cache
	err = cache.Lock()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(cache.dirty), "Cache is dirty")
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.Nil(t, mock.ExpectationsWereMet(), "insert was raised")

}

func TestGormReaderWriter_UseTransactions_True_Direct(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	assert.NoError(t, err)

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	gormRW.UseTransactions = true // Explicitly set

	// Test BeginTx
	mock.ExpectBegin()
	tx, err := gormRW.BeginTx()
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.NotEqual(t, gormRW.db, tx, "BeginTx should return a new transaction object")

	// Test CommitTx
	mock.ExpectCommit()
	gormRW.CommitTx(tx) // CommitTx does not return an error

	// Test Save operation with an explicit transaction
	itemToSave := newTestDBItem("save_item_tx_true")
	itemToSave.ID = 3

	mock.ExpectBegin() // Expect Begin for the new transaction txForSave
	txForSave, err := gormRW.BeginTx()
	assert.NoError(t, err)
	assert.NotNil(t, txForSave)

	// The Save method itself does not emit Begin/Commit, it uses the passed transaction
	// Corrected SQL for UPDATE and arguments for GORM v2
	mock.ExpectExec(`UPDATE "test_db_items" SET "value"=\$1 WHERE "id" = \$2`). // Single escape for $
		WithArgs(itemToSave.Value, itemToSave.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = gormRW.Save(itemToSave, txForSave)
	assert.NoError(t, err)

	mock.ExpectCommit() // Expect Commit for txForSave
	gormRW.CommitTx(txForSave)


	// Further, test that RollbackTx works as expected.
	mock.ExpectBegin()
	txToRollback, err := gormRW.BeginTx()
	assert.NoError(t, err)
	assert.NotNil(t, txToRollback)
	mock.ExpectRollback()
	gormRW.RollbackTx(txToRollback) // RollbackTx does not return an error

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormReaderWriter_PreloadAssociations_True(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	assert.NoError(t, err)

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	gormRW.PreloadAssociations = true // Enable preloading

	expectedItem := newTestDBItem("preload_item_true")
	expectedItem.ID = 1

	// Expectation for Find with Preload(clause.Associations)
	// The actual SQL for preload can be complex and dialect-specific.
	// GORM might issue multiple queries for preloading.
	// For this test, we'll focus on the primary Find query containing Preload.
	// We assume `clause.Associations` will result in `Preload("Associations")` call on GORM chain.
	// The `Find` method in `gorm_readerwriter.go` does:
	//   `q := g.db.Limit(1)`
	//   `if g.PreloadAssociations { q = q.Preload(clause.Associations) }`
	//   `res := q.Find(&item, query, key)`
	// So, the mock should reflect this.
	// sqlmock might not easily mock the .Preload() part of the chain directly unless it translates to a specific SQL hint
	// that we can match. Often, Preload results in separate queries.
	// Let's assume for now GORM translates Preload into some SQL that we can match in the main query,
	// or that `Preload(clause.Associations)` doesn't change the main SELECT query text for this simple case
	// but GORM handles it internally. If Preload issues separate queries, this mock will need adjustment.

	mock.ExpectQuery(`SELECT \* FROM "test_db_items" WHERE "test_db_items"\."value" = \$1 LIMIT \$2`).
		WithArgs(expectedItem.Value, 1).
		WillReturnRows(sqlmock.NewRows(testColumns).AddRow(expectedItem.ID, expectedItem.Value))
	// If `Preload(clause.Associations)` adds specific SQL like `PRELOAD ("associations")` this query would need to change.
	// Or, if it issues secondary queries, those would need to be mocked.
	// For now, assuming the Find part is the primary check.
	// A more robust way would be to use a GORM specific testing tool or inspect the GORM query object if possible.

	item, err := gormRW.Find(expectedItem.Value, nil) // Pass nil for tx when not using an explicit one

	assert.NoError(t, err)
	// 'found' is not returned by gormRW.Find; error presence indicates found/not found
	// For a successful find, err is nil. If not found, err is "not found" or gorm.ErrRecordNotFound
	assert.Equal(t, expectedItem.ID, item.ID)
	assert.Equal(t, expectedItem.Value, item.Value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormReaderWriter_PreloadAssociations_False(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	assert.NoError(t, err)

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	gormRW.PreloadAssociations = false // Disable preloading

	expectedItem := newTestDBItem("preload_item_false")
	expectedItem.ID = 1

	// Expectation for Find without Preload
	mock.ExpectQuery(`SELECT \* FROM "test_db_items" WHERE "test_db_items"\."value" = \$1 LIMIT \$2`).
		WithArgs(expectedItem.Value, 1).
		WillReturnRows(sqlmock.NewRows(testColumns).AddRow(expectedItem.ID, expectedItem.Value))

	item, err := gormRW.Find(expectedItem.Value, nil) // Pass nil for tx

	assert.NoError(t, err)
	// 'found' is not returned by gormRW.Find
	assert.Equal(t, expectedItem.ID, item.ID)
	assert.Equal(t, expectedItem.Value, item.Value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// mockLogger is a mock implementation of the Logger interface for testing.
type mockLogger struct {
	InfoMessages []string
	WarnMessages []string
	ErrorMessages []string
	InfoActions []string
	WarnActions []string
	ErrorActions []string
	InfoItems []Cacheable
	WarnItems []Cacheable
	ErrorItems []Cacheable
}

func (m *mockLogger) Info(msg string, action string, item ...Cacheable) {
	m.InfoMessages = append(m.InfoMessages, msg)
	m.InfoActions = append(m.InfoActions, action)
	if len(item) > 0 {
		m.InfoItems = append(m.InfoItems, item[0])
	}
}

func (m *mockLogger) Warn(msg string, action string, item ...Cacheable) {
	m.WarnMessages = append(m.WarnMessages, msg)
	m.WarnActions = append(m.WarnActions, action)
	if len(item) > 0 {
		m.WarnItems = append(m.WarnItems, item[0])
	}
}

func (m *mockLogger) Error(msg string, action string, item ...Cacheable) {
	m.ErrorMessages = append(m.ErrorMessages, msg)
	m.ErrorActions = append(m.ErrorActions, action)
	if len(item) > 0 {
		m.ErrorItems = append(m.ErrorItems, item[0])
	}
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		InfoMessages:  []string{},
		WarnMessages:  []string{},
		ErrorMessages: []string{},
		InfoActions: []string{},
		WarnActions: []string{},
		ErrorActions: []string{},
		InfoItems: []Cacheable{},
		WarnItems: []Cacheable{},
		ErrorItems: []Cacheable{},
	}
}

func TestGormReaderWriter_Logging(t *testing.T) {
	// No DB interaction needed for this test, so sqlmock setup is omitted.
	gDB, _ := gorm.Open(postgres.New(postgres.Config{DSN: "mock"}), &gorm.Config{}) // Minimal GORM setup

	ml := newMockLogger()
	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem, ml)

	// Test Info logging without item
	gormRW.Info("test info message", "test_action_info")
	assert.Len(t, ml.InfoMessages, 1, "Expected one info message")
	assert.Equal(t, "test info message", ml.InfoMessages[0])
	assert.Len(t, ml.InfoActions, 1)
	assert.Equal(t, "test_action_info", ml.InfoActions[0])
	assert.Len(t, ml.InfoItems, 0)


	// Test Warn logging without item
	gormRW.Warn("test warn message", "test_action_warn")
	assert.Len(t, ml.WarnMessages, 1, "Expected one warn message")
	assert.Equal(t, "test warn message", ml.WarnMessages[0])
	assert.Len(t, ml.WarnActions, 1)
	assert.Equal(t, "test_action_warn", ml.WarnActions[0])
	assert.Len(t, ml.WarnItems, 0)

	// Test Info logging with item
	testItemInfo := newTestDBItem("info_item")
	gormRW.Info("info with item", "test_action_info_item", testItemInfo)
	assert.Len(t, ml.InfoMessages, 2)
	assert.Equal(t, "info with item", ml.InfoMessages[1])
	assert.Len(t, ml.InfoActions, 2)
	assert.Equal(t, "test_action_info_item", ml.InfoActions[1])
	assert.Len(t, ml.InfoItems, 1)
	assert.Equal(t, testItemInfo, ml.InfoItems[0])

	// Test Warn logging with item
	testItemWarn := newTestDBItem("warn_item")
	gormRW.Warn("warn with item", "test_action_warn_item", testItemWarn)
	assert.Len(t, ml.WarnMessages, 2)
	assert.Equal(t, "warn with item", ml.WarnMessages[1])
	assert.Len(t, ml.WarnActions, 2)
	assert.Equal(t, "test_action_warn_item", ml.WarnActions[1])
	assert.Len(t, ml.WarnItems, 1)
	assert.Equal(t, testItemWarn, ml.WarnItems[0])

	// Note: GormCacheReaderWriter itself doesn't have an Error method that directly calls logger.Error.
	// The Logger interface has Error, but GormCacheReaderWriter only has Info and Warn methods.
	// So, we don't test gormRW.Error() here.
}

func TestGormReaderWriter_Find_NotFound_RowsAffectedZero(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	assert.NoError(t, err)

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	keyToFind := "not_found_item"
	expectedTemplateItem := newTestDBItem(keyToFind)

	// Mock GORM's Find to return 0 rows affected, no error
	mock.ExpectQuery(`SELECT \* FROM "test_db_items" WHERE "test_db_items"\."value" = \$1 LIMIT \$2`).
		WithArgs(keyToFind, 1).
		WillReturnRows(sqlmock.NewRows(testColumns)) // No rows added, so RowsAffected will be 0

	returnedItem, err := gormRW.Find(keyToFind, nil) // Pass nil for tx

	assert.Error(t, err)
	assert.Equal(t, "not found", err.Error(), "Error should be the specific 'not found' error for RowsAffected=0")
	// Check that the returned item is the template item.
	// The newTestDBItem function creates the template.
	assert.Equal(t, expectedTemplateItem.Value, returnedItem.Value, "Returned item should be the template item based on the key")
	assert.Equal(t, expectedTemplateItem.ID, returnedItem.ID, "ID should be the default for a template") // Assuming ID is zero/default for a template from newTestDBItem

	assert.NoError(t, mock.ExpectationsWereMet())
}


func TestGormReaderWriter_Find_DBError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	assert.NoError(t, err)

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)

	expectedError := gorm.ErrRecordNotFound
	mock.ExpectQuery(`SELECT \* FROM "test_db_items" WHERE "test_db_items"\."value" = \$1 LIMIT \$2`).
		WithArgs("test_item", 1).
		WillReturnError(expectedError)

	_, err = gormRW.Find("test_item", nil) // Pass nil for tx, remove 'found'

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	// found variable removed
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormReaderWriter_Save_DBError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	assert.NoError(t, err)

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	itemToSave := newTestDBItem("test_save_item")
	itemToSave.ID = 1 // Assume item already exists for an update-like Save

	expectedError := gorm.ErrInvalidDB
	mock.ExpectBegin()
	// Assuming Save performs an UPSERT or UPDATE. Adjust if it's purely an INSERT.
	// For an existing item (ID=1), GORM would typically do an UPDATE.
	// If it's a new item, it would be an INSERT.
	// Let's assume it tries to save (which could be insert or update) and fails.
	// The actual SQL might vary based on GORM's dialects and Save behavior for new/existing.
	// We'll use a general exec that matches what GORM would do for Save.
	// This regex tries to match either an INSERT or an UPDATE.
	// Corrected SQL for UPDATE and arguments for GORM v2
	mock.ExpectExec(`UPDATE "test_db_items" SET "value"=\$1 WHERE "id" = \$2`). // Single escape for $
		WithArgs(itemToSave.Value, itemToSave.ID).
		WillReturnError(expectedError)

	// The test should explicitly manage the transaction for Save if it's testing transactional save errors.
	tx, errTx := gormRW.BeginTx() // This will be the transaction for the Save operation
	assert.NoError(t, errTx)
	assert.NotNil(t, tx)

	err = gormRW.Save(itemToSave, tx) // Pass the transaction

	mock.ExpectRollback() // Expect a rollback due to the error
	gormRW.RollbackTx(tx) // Explicitly rollback

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormReaderWriter_UseTransactions_False(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)) // Changed to Regexp
	assert.NoError(t, err)
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	assert.NoError(t, err)

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	gormRW.UseTransactions = false // Disable transactions

	// Test BeginTx
	tx, err := gormRW.BeginTx() // Returns (any, error)
	assert.NoError(t, err)
	assert.Same(t, gormRW.db, tx, "BeginTx should return the original db instance when UseTransactions is false")

	// Test CommitTx - should be a no-op
	// No direct mock expectation here as it should not interact with the DB transaction capabilities
	gormRW.CommitTx(tx) // Does not return error

	// Test Find operation
	expectedItem := newTestDBItem("find_item_no_tx")
	expectedItem.ID = 1
	mock.ExpectQuery(`SELECT \* FROM "test_db_items" WHERE "test_db_items"\."value" = \$1 LIMIT \$2`).
		WithArgs(expectedItem.Value, 1).
		WillReturnRows(sqlmock.NewRows(testColumns).AddRow(expectedItem.ID, expectedItem.Value))

	item, err := gormRW.Find(expectedItem.Value, nil) // Pass nil for tx, 'found' not returned
	assert.NoError(t, err)
	// found variable removed
	assert.Equal(t, expectedItem.ID, item.ID)
	assert.Equal(t, expectedItem.Value, item.Value)

	// Test Save operation
	itemToSave := newTestDBItem("save_item_no_tx")
	itemToSave.ID = 2
	// Expect direct exec, not begin/commit
	// Corrected SQL for UPDATE and arguments for GORM v2
	// When UseTransactions is false, GORM might still wrap this in its own implicit transaction for some drivers if not careful.
	// However, sqlmock should just see the raw SQL.
	// If GORM itself adds BEGIN/COMMIT, the mock needs to expect that.
	// GORM's Save() on a base DB connection usually wraps the operation in a transaction.
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "test_db_items" SET "value"=\$1 WHERE "id" = \$2`). // Single escape for $
		WithArgs(itemToSave.Value, itemToSave.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = gormRW.Save(itemToSave, gormRW.db) // Save needs a tx, pass gormRW.db when UseTransactions=false
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
