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

	gormRW := NewGormCacheReaderWriter[string, testDBItem](gDB, newTestDBItem)
	cfg := NewDefaultConfig[string, testDBItem](gormRW)
	cfg.WriteFreq = 100 * time.Millisecond
	cache := NewLazyWriterCache[string, testDBItem](cfg)
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
