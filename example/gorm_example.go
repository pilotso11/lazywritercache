// MIT License
//
// # Copyright (c) 2023 Seth Osher
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
package main

import (
	"flag"
	"log"
	"math/rand"

	"github.com/pilotso11/lazywritercache"
	"github.com/pilotso11/lazywritercache/gormcache"

	"github.com/xo/dburl"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Command line
	// -db URL
	// dbUrl := flag.String("db", "sqlite:test.db", "Database URL")
	dbUrl := flag.String("db", "postgres://postgres:postgres@localhost:5438/test", "Database URL")
	help := flag.Bool("h", false, "Print this help")
	flag.Parse()

	// Setup logger
	// Setup database
	if *help {
		flag.PrintDefaults()
		return
	}
	dsn, err := dburl.Parse(*dbUrl)
	if err != nil {
		log.Fatalf("%v", err)
	}

	var db *gorm.DB
	switch dsn.Driver {
	case "postgres":
		db, err = gorm.Open(postgres.Open(dsn.DSN), &gorm.Config{})
		/*case "sqlite3": // this cause cgo issues for some, especially on windows
		db, err = gorm.Open(sqlite.Open(dsn.DSN), &gorm.Config{}) */
	}
	if err != nil {
		log.Fatalf("%v", err)
	}
	err = db.AutoMigrate(&Person{})
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Create the cache
	gormRW := gormcache.NewReaderWriter[string, Person](db, NewEmptyPerson)
	cacheConfig := lazywritercache.NewDefaultConfig[string, Person](gormRW)
	cache := lazywritercache.NewLazyWriterCache[string, Person](cacheConfig)
	defer cache.Shutdown()

	defer cache.Flush()

	// Do some work
	for i := 1; i < 10000; i++ {
		name := GetRandomName()
		doWork(cache, name)
	}

	log.Println("Work done, flushing the cache")
	cache.Flush()

	log.Println(cache.CacheStats.JSON())
	log.Println(cache.CacheStats.String())
}

// This is not a great example as it has rather high read/write ratio, but its good enough
// to illustrate the api
func doWork(cache *lazywritercache.LazyWriterCache[string, Person], name string) {
	record, ok, _ := cache.GetAndLock(name)
	defer cache.Unlock()
	if !ok || rand.Float64() < 0.025 { // new or 2.5% random chance of update
		person := Person{Name: name, City: GetRandomCity()}
		cache.Save(person)
	} else {
		if rand.Float64() < 0.0001 { // Print out a few random names we found
			log.Println("Found: " + record.Name)
		}
	}

}

type Person struct {
	gorm.Model
	Name string
	City string
}

func (p Person) Key() interface{} {
	return p.Name
}

func (p Person) CopyKeyDataFrom(from lazywritercache.Cacheable) lazywritercache.Cacheable {
	fromData := from.(Person)

	p.ID = fromData.ID
	p.CreatedAt = fromData.CreatedAt
	p.UpdatedAt = fromData.UpdatedAt
	p.DeletedAt = fromData.DeletedAt

	return p
}

func NewEmptyPerson(key string) Person {
	return Person{
		Name: key,
	}
}
