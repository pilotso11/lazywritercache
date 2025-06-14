package lazygormcache

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/pilotso11/lazywritercache"
)

// testDBItemLF implements the lazywritercache.CacheableLF interface for testing
type testDBItemLF struct {
	ID    uint `gorm:"primarykey"`
	Value string
}

var testColumnsLF = []string{"id", "value"}

func (i testDBItemLF) Key() string {
	return i.Value
}

func (i testDBItemLF) CopyKeyDataFrom(from lazywritercache.CacheableLF[string]) lazywritercache.CacheableLF[string] {
	i.Value = from.Key()
	return i
}

func (i testDBItemLF) String() string {
	return i.Value
}

func newTestDBItemLF(key string) testDBItemLF {
	return testDBItemLF{
		Value: key,
	}
}

// MockLoggerLF implements the LoggerLF interface for testing
type MockLoggerLF struct {
	InfoCalled  bool
	WarnCalled  bool
	ErrorCalled bool
	LastMsg     string
	LastAction  lazywritercache.CacheAction
}

func (m *MockLoggerLF) Info(_ context.Context, msg string, action lazywritercache.CacheAction, _ ...lazywritercache.CacheableLF[string]) {
	m.InfoCalled = true
	m.LastMsg = msg
	m.LastAction = action
}

func (m *MockLoggerLF) Warn(_ context.Context, msg string, action lazywritercache.CacheAction, _ ...lazywritercache.CacheableLF[string]) {
	m.WarnCalled = true
	m.LastMsg = msg
	m.LastAction = action
}

func (m *MockLoggerLF) Error(_ context.Context, msg string, action lazywritercache.CacheAction, _ ...lazywritercache.CacheableLF[string]) {
	m.ErrorCalled = true
	m.LastMsg = msg
	m.LastAction = action
}

func closeIgnore(db *sql.DB) {
	_ = db.Close()
}

func TestNewGormCacheReaderWriteLF(t *testing.T) {
	// Create a mock database
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test creating a new ReaderWriteLF
	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)

	// Verify the ReaderWriteLF was created correctly
	assert.NotNil(t, rw.db, "DB should not be nil")
	assert.NotNil(t, rw.getTemplateItem, "getTemplateItem should not be nil")
	assert.True(t, rw.UseTransactions, "UseTransactions should be true by default")
	assert.Nil(t, rw.Logger, "Logger should be nil by default")
}

func TestReaderWriteLF_Find(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)

	// Test Find with a successful query
	mock.ExpectQuery(`SELECT * FROM "test_db_item_lves" WHERE "test_db_item_lves"."value" = $1 LIMIT $2`).
		WithArgs("item1", 1).
		WillReturnRows(sqlmock.NewRows(testColumnsLF).AddRow(1, "item1"))

	item, err := rw.Find(ctx, "item1", nil)
	assert.NoError(t, err, "Find should not return an error")
	assert.Equal(t, "item1", item.Value, "Item value should match")
	assert.Equal(t, uint(1), item.ID, "Item ID should match")

	// Test Find with a query that returns no rows
	mock.ExpectQuery(`SELECT * FROM "test_db_item_lves" WHERE "test_db_item_lves"."value" = $1 LIMIT $2`).
		WithArgs("missing", 1).
		WillReturnRows(sqlmock.NewRows(testColumnsLF))

	_, err = rw.Find(ctx, "missing", nil)
	assert.Error(t, err, "Find should return an error when no rows are found")
	assert.Equal(t, "not found", err.Error(), "Error message should be 'not found'")

	// Test Find with a query that returns an error
	mock.ExpectQuery(`SELECT * FROM "test_db_item_lves" WHERE "test_db_item_lves"."value" = $1 LIMIT $2`).
		WithArgs("error", 1).
		WillReturnError(sql.ErrConnDone)

	_, err = rw.Find(ctx, "error", nil)
	assert.Error(t, err, "Find should return an error when the query fails")
	assert.Equal(t, sql.ErrConnDone, err, "Error should be sql.ErrConnDone")

	// Test Find with a transaction
	mock.ExpectBegin()
	tx := gDB.Begin()
	mock.ExpectQuery(`SELECT * FROM "test_db_item_lves" WHERE "test_db_item_lves"."value" = $1 LIMIT $2`).
		WithArgs("item2", 1).
		WillReturnRows(sqlmock.NewRows(testColumnsLF).AddRow(2, "item2"))

	item, err = rw.Find(ctx, "item2", tx)
	assert.NoError(t, err, "Find should not return an error")
	assert.Equal(t, "item2", item.Value, "Item value should match")
	assert.Equal(t, uint(2), item.ID, "Item ID should match")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriteLF_Save(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)

	// Test Save with a successful query
	mock.ExpectBegin()
	// tx := gDB.Begin()
	mock.ExpectQuery(`INSERT INTO "test_db_item_lves" ("value") VALUES ($1) RETURNING "id"`).
		WithArgs("item1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	item := newTestDBItemLF("item1")
	err = rw.Save(ctx, item, gDB)
	assert.NoError(t, err, "Save should not return an error")

	// Test Save with a query that returns an error
	mock.ExpectBegin()
	// tx = gDB.Begin()
	mock.ExpectQuery(`INSERT INTO "test_db_item_lves" ("value") VALUES ($1) RETURNING "id"`).
		WithArgs("error").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	item = newTestDBItemLF("error")
	err = rw.Save(ctx, item, gDB)
	assert.Error(t, err, "Save should return an error when the query fails")
	assert.Equal(t, sql.ErrConnDone, err, "Error should be sql.ErrConnDone")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriteLF_BeginTx(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test BeginTx with UseTransactions = true
	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)
	rw.UseTransactions = true
	mock.ExpectBegin()

	tx, err := rw.BeginTx(ctx)
	assert.NoError(t, err, "BeginTx should not return an error")
	assert.NotNil(t, tx, "Transaction should not be nil")

	// Test BeginTx with UseTransactions = false
	rw.UseTransactions = false

	tx, err = rw.BeginTx(ctx)
	assert.NoError(t, err, "BeginTx should not return an error")
	assert.Equal(t, gDB, tx, "Transaction should be the DB when UseTransactions is false")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriteLF_CommitTx(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test CommitTx with UseTransactions = true
	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)
	rw.UseTransactions = true
	mock.ExpectBegin()
	mock.ExpectCommit()

	tx := gDB.Begin()
	err = rw.CommitTx(ctx, tx)
	assert.NoError(t, err, "CommitTx should not return an error")

	// Test CommitTx with UseTransactions = false
	rw.UseTransactions = false

	err = rw.CommitTx(ctx, tx)
	assert.NoError(t, err, "CommitTx should not return an error")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriteLF_RollbackTx(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test CommitTx with UseTransactions = true
	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)
	rw.UseTransactions = true
	mock.ExpectBegin()
	mock.ExpectRollback()

	tx := gDB.Begin()
	err = rw.RollbackTx(ctx, tx)
	assert.NoError(t, err, "RollbackTx should not return an error")
	// Test CommitTx with UseTransactions = false
	rw.UseTransactions = false
	err = rw.CommitTx(ctx, gDB)
	assert.NoError(t, err, "RollbackTx with no TX should not return an error")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriteLF_RollbackTxError(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test CommitTx with UseTransactions = true
	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)

	// Force an exception
	rw.UseTransactions = true
	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(gorm.ErrInvalidData)

	tx2, err := rw.BeginTx(ctx)
	assert.NoError(t, err, "BeginTx should not return an error")
	err = rw.RollbackTx(ctx, tx2)
	assert.Error(t, err, "RollbackTx should return an error")
	assert.ErrorIs(t, err, gorm.ErrInvalidData, "Error should be gorm.ErrInvalidData")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriteLF_Info(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test Info with a logger
	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)
	logger := &MockLoggerLF{}
	rw.Logger = logger

	// Test Info without an item
	rw.Info(ctx, "test message", lazywritercache.ActionEvict)
	assert.True(t, logger.InfoCalled, "Info should call the logger's Info method")
	assert.Equal(t, "test message", logger.LastMsg, "Message should match")
	assert.Equal(t, "evict", logger.LastAction.String(), "Action should match")

	// Test Info with an item
	item := newTestDBItemLF("item1")
	rw.Info(ctx, "test message with item", lazywritercache.ActionEvict, item)
	assert.Equal(t, "test message with item", logger.LastMsg, "Message should match")
	assert.Equal(t, lazywritercache.ActionEvict, logger.LastAction, "Action should match")

	// Test Info without a logger
	rw.Logger = nil
	rw.Info(ctx, "test message without logger", lazywritercache.ActionEvict)
	// No assertion needed, just make sure it doesn't panic
}

func TestReaderWriteLF_Warn(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test Warn with a logger
	rw := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)
	logger := &MockLoggerLF{}
	rw.Logger = logger

	// Test Warn without an item
	rw.Warn(ctx, "test message", lazywritercache.ActionWriteDirty)
	assert.True(t, logger.WarnCalled, "Warn should call the logger's Warn method")
	assert.Equal(t, "test message", logger.LastMsg, "Message should match")
	assert.Equal(t, "write-dirty", logger.LastAction.String(), "Action should match")

	// Test Warn with an item
	item := newTestDBItemLF("item1")
	rw.Warn(ctx, "test message with item", lazywritercache.ActionWriteDirty, item)
	assert.Equal(t, "test message with item", logger.LastMsg, "Message should match")
	assert.Equal(t, lazywritercache.ActionWriteDirty, logger.LastAction, "Action should match")

	// Test Warn without a logger
	rw.Logger = nil
	rw.Warn(ctx, "test message without logger", lazywritercache.ActionWriteDirty)
	// No assertion needed, just make sure it doesn't panic
}

func TestGormReaderWriterLF_Integration(t *testing.T) {
	ctx := context.Background()
	// Create a mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer closeIgnore(db)

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Create a ReaderWriteLF
	gormRW := NewReaderWriterLF[string, testDBItemLF](gDB, newTestDBItemLF)

	// Create a LazyWriterCacheLF
	cfg := lazywritercache.NewDefaultConfigLF[string, testDBItemLF](gormRW)
	cfg.WriteFreq = 100 * time.Millisecond
	cache := lazywritercache.NewLazyWriterCacheLF[string, testDBItemLF](cfg)
	defer cache.Shutdown()

	// Test loading a non-existent item
	mock.ExpectQuery(`SELECT * FROM "test_db_item_lves" WHERE "test_db_item_lves"."value" = $1 LIMIT $2`).
		WithArgs("item1", 1).
		WillReturnRows(sqlmock.NewRows(testColumnsLF))

	_, ok := cache.Load(ctx, "item1")
	assert.False(t, ok, "Item should not be found")

	// Test saving and loading an item
	item := testDBItemLF{Value: "item1"}
	cache.Save(item)

	// The item should be in the cache now
	loadedItem, ok := cache.Load(ctx, "item1")
	assert.True(t, ok, "Item should be found")
	assert.Equal(t, item.Value, loadedItem.Value, "Item value should match")

	// Test that the lazy writer saves the item to the database
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT * FROM "test_db_item_lves" WHERE "test_db_item_lves"."value" = $1 LIMIT $2`).
		WithArgs("item1", 1).
		WillReturnRows(sqlmock.NewRows(testColumnsLF))
	mock.ExpectQuery(`INSERT INTO "test_db_item_lves" ("value") VALUES ($1) RETURNING "id"`).
		WithArgs("item1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	assert.Eventuallyf(t, func() bool {
		return mock.ExpectationsWereMet() == nil
	}, 200*time.Millisecond, 10*time.Millisecond, "All expectations should be met within 200ms ")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}
