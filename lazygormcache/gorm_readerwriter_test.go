package lazygormcache

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/pilotso11/lazywritercache"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	InfoCalled  bool
	WarnCalled  bool
	ErrorCalled bool
	LastMsg     string
	LastAction  string
}

func (m *MockLogger) Info(msg string, action string, item ...lazywritercache.Cacheable) {
	m.InfoCalled = true
	m.LastMsg = msg
	m.LastAction = action
}

func (m *MockLogger) Warn(msg string, action string, item ...lazywritercache.Cacheable) {
	m.WarnCalled = true
	m.LastMsg = msg
	m.LastAction = action
}

func (m *MockLogger) Error(msg string, action string, item ...lazywritercache.Cacheable) {
	m.ErrorCalled = true
	m.LastMsg = msg
	m.LastAction = action
}

type testDBItem struct {
	ID    uint `gorm:"primarykey"`
	Value string
}

var testColumns = []string{"id", "value"}

func (i testDBItem) Key() interface{} {
	return i.Value
}

func (i testDBItem) CopyKeyDataFrom(from lazywritercache.Cacheable) lazywritercache.Cacheable {
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

// TestItem is a struct with associations for testing PreloadAssociations
type TestItem struct {
	ID       uint `gorm:"primarykey"`
	Value    string
	Children []TestChild `gorm:"foreignKey:ParentID"`
}

type TestChild struct {
	ID       uint `gorm:"primarykey"`
	ParentID uint
	Name     string
}

func (i TestItem) Key() interface{} {
	return i.Value
}

func (i TestItem) CopyKeyDataFrom(from lazywritercache.Cacheable) lazywritercache.Cacheable {
	i.Value = from.Key().(string)
	return i
}

func (i TestItem) String() string {
	return i.Value
}

func newTestItem(key string) TestItem {
	return TestItem{
		Value: key,
	}
}

func TestNewReaderWriter(t *testing.T) {
	// Create a mock database
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test creating a new ReaderWriter without logger
	rw := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)

	// Verify the ReaderWriter was created correctly
	assert.NotNil(t, rw.db, "DB should not be nil")
	assert.NotNil(t, rw.getTemplateItem, "getTemplateItem should not be nil")
	assert.True(t, rw.UseTransactions, "UseTransactions should be true by default")
	assert.True(t, rw.PreloadAssociations, "PreloadAssociations should be true by default")
	assert.Nil(t, rw.Logger, "Logger should be nil by default")

	// Test creating a new ReaderWriter with logger
	logger := &MockLogger{}
	rwWithLogger := NewReaderWriter[string, testDBItem](gDB, newTestDBItem, logger)

	// Verify the ReaderWriter was created correctly
	assert.NotNil(t, rwWithLogger.db, "DB should not be nil")
	assert.NotNil(t, rwWithLogger.getTemplateItem, "getTemplateItem should not be nil")
	assert.True(t, rwWithLogger.UseTransactions, "UseTransactions should be true by default")
	assert.True(t, rwWithLogger.PreloadAssociations, "PreloadAssociations should be true by default")
	assert.Equal(t, logger, rwWithLogger.Logger, "Logger should be set correctly")
}

func TestReaderWriter_Find(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	rw := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)

	// Test Find with a successful query
	mock.ExpectQuery(`SELECT * FROM "test_db_items" WHERE "test_db_items"."value" = $1 LIMIT $2`).
		WithArgs("item1", 1).
		WillReturnRows(sqlmock.NewRows(testColumns).AddRow(1, "item1"))

	item, err := rw.Find("item1", nil)
	assert.NoError(t, err, "Find should not return an error")
	assert.Equal(t, "item1", item.Value, "Item value should match")
	assert.Equal(t, uint(1), item.ID, "Item ID should match")

	// Test Find with a query that returns no rows
	mock.ExpectQuery(`SELECT * FROM "test_db_items" WHERE "test_db_items"."value" = $1 LIMIT $2`).
		WithArgs("missing", 1).
		WillReturnRows(sqlmock.NewRows(testColumns))

	_, err = rw.Find("missing", nil)
	assert.Error(t, err, "Find should return an error when no rows are found")
	assert.Equal(t, "not found", err.Error(), "Error message should be 'not found'")

	// Test Find with a query that returns an error
	mock.ExpectQuery(`SELECT * FROM "test_db_items" WHERE "test_db_items"."value" = $1 LIMIT $2`).
		WithArgs("error", 1).
		WillReturnError(sql.ErrConnDone)

	_, err = rw.Find("error", nil)
	assert.Error(t, err, "Find should return an error when the query fails")
	assert.Equal(t, sql.ErrConnDone, err, "Error should be sql.ErrConnDone")

	// Test Find with a transaction
	mock.ExpectBegin()
	tx := gDB.Begin()
	mock.ExpectQuery(`SELECT * FROM "test_db_items" WHERE "test_db_items"."value" = $1 LIMIT $2`).
		WithArgs("item2", 1).
		WillReturnRows(sqlmock.NewRows(testColumns).AddRow(2, "item2"))

	item, err = rw.Find("item2", tx)
	assert.NoError(t, err, "Find should not return an error")
	assert.Equal(t, "item2", item.Value, "Item value should match")
	assert.Equal(t, uint(2), item.ID, "Item ID should match")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriter_Find_PreloadAssociations(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test with PreloadAssociations = true
	rwPreload := NewReaderWriter[string, TestItem](gDB, newTestItem)
	rwPreload.PreloadAssociations = true

	// Mock the query with preloaded associations
	mock.ExpectQuery(`SELECT * FROM "test_items" WHERE "test_items"."value" = $1 LIMIT $2`).
		WithArgs("item1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "value"}).AddRow(1, "item1"))

	// Mock the query for preloading children
	mock.ExpectQuery(`SELECT * FROM "test_children" WHERE "test_children"."parent_id" = $1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id", "name"}).
			AddRow(1, 1, "child1").
			AddRow(2, 1, "child2"))

	_, err = rwPreload.Find("item1", nil)
	assert.NoError(t, err, "Find with PreloadAssociations should not return an error")

	// Test with PreloadAssociations = false
	rwNoPreload := NewReaderWriter[string, TestItem](gDB, newTestItem)
	rwNoPreload.PreloadAssociations = false

	// Mock the query without preloaded associations
	mock.ExpectQuery(`SELECT * FROM "test_items" WHERE "test_items"."value" = $1 LIMIT $2`).
		WithArgs("item2", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "value"}).AddRow(2, "item2"))

	_, err = rwNoPreload.Find("item2", nil)
	assert.NoError(t, err, "Find without PreloadAssociations should not return an error")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriter_Save(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DSN:        "mock",
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	rw := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)

	// Test Save with a successful query
	mock.ExpectBegin()
	tx := gDB.Begin()
	mock.ExpectQuery(`INSERT INTO "test_db_items" ("value") VALUES ($1) RETURNING "id"`).
		WithArgs("item1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	item := newTestDBItem("item1")
	err = rw.Save(item, tx)
	assert.NoError(t, err, "Save should not return an error")

	// Test Save with a query that returns an error
	mock.ExpectBegin()
	tx = gDB.Begin()
	mock.ExpectQuery(`INSERT INTO "test_db_items" ("value") VALUES ($1) RETURNING "id"`).
		WithArgs("error").
		WillReturnError(sql.ErrConnDone)

	item = newTestDBItem("error")
	err = rw.Save(item, tx)
	assert.Error(t, err, "Save should return an error when the query fails")
	assert.Equal(t, sql.ErrConnDone, err, "Error should be sql.ErrConnDone")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriter_BeginTx(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test BeginTx with UseTransactions = true
	rw := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)
	rw.UseTransactions = true
	mock.ExpectBegin()

	tx, err := rw.BeginTx()
	assert.NoError(t, err, "BeginTx should not return an error")
	assert.NotNil(t, tx, "Transaction should not be nil")

	// Test BeginTx with UseTransactions = false
	rw.UseTransactions = false

	tx, err = rw.BeginTx()
	assert.NoError(t, err, "BeginTx should not return an error")
	assert.Equal(t, gDB, tx, "Transaction should be the DB when UseTransactions is false")

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriter_CommitTx(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test CommitTx with UseTransactions = true
	rw := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)
	rw.UseTransactions = true
	mock.ExpectBegin()
	mock.ExpectCommit()

	tx := gDB.Begin()
	rw.CommitTx(tx)

	// Test CommitTx with UseTransactions = false
	rw.UseTransactions = false

	rw.CommitTx(gDB)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "All expectations should be met")
}

func TestReaderWriter_Info(t *testing.T) {
	// Create a mock database
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test Info with a logger
	rw := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)
	logger := &MockLogger{}
	rw.Logger = logger

	// Test Info without an item
	rw.Info("test message", "test action")
	assert.True(t, logger.InfoCalled, "Info should call the logger's Info method")
	assert.Equal(t, "test message", logger.LastMsg, "Message should match")
	assert.Equal(t, "test action", logger.LastAction, "Action should match")

	// Test Info with an item
	item := newTestDBItem("item1")
	rw.Info("test message with item", "test action with item", item)
	assert.Equal(t, "test message with item", logger.LastMsg, "Message should match")
	assert.Equal(t, "test action with item", logger.LastAction, "Action should match")

	// Test Info without a logger
	rw.Logger = nil
	rw.Info("test message without logger", "test action without logger")
	// No assertion needed, just make sure it doesn't panic
}

func TestReaderWriter_Warn(t *testing.T) {
	// Create a mock database
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	gDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating gorm DB: %v", err)
	}

	// Test Warn with a logger
	rw := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)
	logger := &MockLogger{}
	rw.Logger = logger

	// Test Warn without an item
	rw.Warn("test message", "test action")
	assert.True(t, logger.WarnCalled, "Warn should call the logger's Warn method")
	assert.Equal(t, "test message", logger.LastMsg, "Message should match")
	assert.Equal(t, "test action", logger.LastAction, "Action should match")

	// Test Warn with an item
	item := newTestDBItem("item1")
	rw.Warn("test message with item", "test action with item", item)
	assert.Equal(t, "test message with item", logger.LastMsg, "Message should match")
	assert.Equal(t, "test action with item", logger.LastAction, "Action should match")

	// Test Warn without a logger
	rw.Logger = nil
	rw.Warn("test message without logger", "test action without logger")
	// No assertion needed, just make sure it doesn't panic
}

func TestGormReaderWriter(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
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

	gormRW := NewReaderWriter[string, testDBItem](gDB, newTestDBItem)
	cfg := lazywritercache.NewDefaultConfig[string, testDBItem](gormRW)
	cfg.WriteFreq = 100 * time.Millisecond
	cache := lazywritercache.NewLazyWriterCache[string, testDBItem](cfg)
	defer func(db *sql.DB) {
		cache.Shutdown()
		_ = db.Close()
	}(db)

	// First dirty hit should return an empty item
	mock.ExpectQuery(`SELECT * FROM "test_db_items" WHERE "test_db_items"."value" = $1 LIMIT $2`).WithArgs("item1", 1).WillReturnRows(sqlmock.NewRows(testColumns))
	item1, ok, err := cache.GetAndRelease("item1")
	assert.NoError(t, err)
	assert.False(t, ok, "item should not be found")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT * FROM "test_db_items" WHERE "test_db_items"."value" = $1 LIMIT $2`).WithArgs("item1", 1).WillReturnRows(sqlmock.NewRows(testColumns))
	mock.ExpectQuery(`INSERT INTO "test_db_items" ("value") VALUES ($1) RETURNING "id"`).WithArgs("item1").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()
	item1.Value = "item1"
	err = cache.Lock()
	assert.NoError(t, err)
	cache.Save(item1)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.True(t, cache.IsDirty(), "Cache is dirty")

	assert.Eventuallyf(t, func() bool {
		return mock.ExpectationsWereMet() == nil
	}, 200*time.Millisecond, time.Millisecond, "timeout")

	err = cache.Lock()
	assert.NoError(t, err)
	err = cache.Unlock()
	assert.NoError(t, err)
	assert.False(t, cache.IsDirty(), "Cache is dirty")
	assert.Nil(t, mock.ExpectationsWereMet(), "insert was raised")

}
