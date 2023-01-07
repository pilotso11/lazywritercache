package main

import (
	"flag"
	"math/rand"
	"time"

	"github.com/pilotso11/lazywritercache"
	"github.com/xo/dburl"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Command line
	// -db URL
	dbUrl := flag.String("db", "postgres://postgres:postgres@localhost:5438/test", "Database URL")
	help := flag.Bool("h", false, "Print this help")
	flag.Parse()

	// Setup logger
	root, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(root)
	logger := zap.S()

	// Setup database
	if *help {
		flag.PrintDefaults()
		return
	}
	dsn, err := dburl.Parse(*dbUrl)
	if err != nil {
		logger.Fatalf("%v", err)
	}
	db, err := gorm.Open(postgres.Open(dsn.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatalf("%v", err)
	}
	err = db.AutoMigrate(&Person{})
	if err != nil {
		logger.Fatalf("%v", err)
	}

	// Create the cache
	gormRW := lazywritercache.NewGormCacheReaderWriter(db, zap.L(), "name", NewEmptyPerson, SaverHelper, FinderHelper)
	cacheConfig := lazywritercache.NewDefaultConfig(gormRW)
	cacheConfig.Limit = 8000
	cacheConfig.PurgeFreq = 1 * time.Second // let's get some purges going
	cache := lazywritercache.NewLazyWriterCache(cacheConfig)
	//defer cache.Flush()

	// Do some work
	for i := 1; i < 10000; i++ {
		name := GetRandomName()
		DoWork(cache, name)
	}

	logger.Info("Work done, flushing the cache")
	cache.Flush()

	logger.Info(cache.CacheStats.JSON())
	logger.Info(cache.CacheStats.String())
}

func DoWork(cache *lazywritercache.LazyWriterCache, name string) {
	record, ok := cache.GetAndLock(name)
	defer cache.Release()
	if !ok || rand.Float64() < 0.025 { // new or 2.5% random chance of update
		person := Person{Name: name, City: GetRandomCity()}
		cache.Save(person)
	} else {
		person := record.(Person)
		if rand.Float64() < 0.0001 { // Print out a few random names we found
			zap.S().Info("Found: " + person.Name)
		}
	}

}

func SaverHelper(tx *gorm.DB, item lazywritercache.CacheItem) *gorm.DB {
	person := item.(Person)
	return tx.Save(&person)
}

func FinderHelper(tx *gorm.DB, item lazywritercache.CacheItem) *gorm.DB {
	person := item.(Person)
	return tx.Limit(1).Find(&person, "name=?", person.Name)
}

type Person struct {
	gorm.Model
	Name string
	City string
}

func (p Person) Key() interface{} {
	return p.Name
}

func (p Person) CopyKeyDataFrom(from lazywritercache.CacheItem) lazywritercache.CacheItem {
	fromData := from.(Person)

	p.ID = fromData.ID
	p.CreatedAt = fromData.CreatedAt
	p.UpdatedAt = fromData.UpdatedAt
	p.DeletedAt = fromData.DeletedAt

	return p
}

func NewEmptyPerson(key interface{}) lazywritercache.CacheItem {
	return Person{
		Name: key.(string),
	}
}
