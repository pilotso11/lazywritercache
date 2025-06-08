# In memory GOLANG Lazy Writer Cache  
![Coverage](https://img.shields.io/badge/Coverage-54.1%25-yellow)
A lazy writer cache is useful in situations where you don't want to block a low latency operation 
with something expensive, such as a database write, but you want the consistency of a single version 
of truth if you need to look for the object again.   A simple goroutine spun off write satisfies the first
condition of unblocking the operation, but doesn't provide any mechanism to resolve the lookup.

A lazy writer cache has the following properties:
* An in memory cache of objects
* Store operations are fast, objects are marked as dirty for lazy writing
* Read objets are also fast as they are coming from the in memory cache of objects
* A goroutine provides background services to write the cache to some durable store (Postgres etc).
* A separate goroutine provides cache pruning to the configured size
* There is no guarantee of object persistence between lazy writes.   If your system crashes in between, your data is lost.  You must be comfortable with this tradeoff.
* To keep writes fast, eviction is also a lazy operation so the size limit is a soft limit that can be breached.   The lazy purge will do a good job of keeping memory usage under control as long as you don't create objects faster than it can purge them.

# Dependencies

The basic lazywritercache has no dependencies outside of core go packages.   The dependencies in go.mod are for the GORM example in [/example](/example).

A second lock free implementation which performs slightly better under highly parallel read workloads is
in the lockfree sub-folder as lockfree.LazyWriterCacheLF and has a dependence on xsync.MapOf.

# Why not just use REDIS?
Good question.  In fact, if you need to live in a distributed world then REDIS is probably a good solution. But it's still much 
slower than an in memory map.  You've got to make hops to a server etc.   This in memory cache is 
much faster even than REDIS - roughly 100x.  The benchmark results below (and in the tests) show performance nearly as good as an in memory map.  Keep in mind that the lazy writer db time is still a practical limit on how fast you can update the cache.   A network based solution will have times measured
in 10's of microseconds at best.   A database write is another 2 orders of magnitude slower. 

If you are really sensitive about nonblocking performance then
you could conceivably put a lazy write cache in front of redis with a fairly quick sync loop.

Benchmark results for cache size of 20k items and 100k items on macbook apple silicon.

The lock-free implementation has roughy 2x performance pickup for many parallel reads at a roughly 2x write penalty.
Benchmark [tracking results on GH Pages.](https://pilotso11.github.io/lazywritercache/dev/bench/)

| Benchmark                | ns/op | b/op | allocs/op |
|--------------------------|------:|-----:|----------:|
| CacheWrite 20k           |   137 |  149 |         3 |
| CacheWrite 100k          |   168 |  141 |         3 |
| CacheRead 20k            |    48 |    0 |         0 |
| CacheRead 100k           |    57 |    1 |         0 |
| CacheRead 20k 5 threads  |   726 |    4 |         0 |
| CacheRead 20k 10 threads |  1505 |    4 |         0 |

| Benchmark Lock Free      | ns/op | b/op | allocs/op |
|--------------------------|------:|-----:|----------:|
| CacheWrite 20k           |   214 |  109 |         6 |
| CacheWrite 100k          |   253 |  113 |         7 |
| CacheRead 20k            |    42 |    0 |         0 |
| CacheRead 100k           |    56 |    1 |         0 |
| CacheRead 20k 5 threads  |   292 |    4 |         0 |
| CacheRead 20k 10 threads |   706 |    4 |         0 |


# How do I use it?

You need to be at least on go1.18 as we use generics.  If you've not tried out generics, now is great time to do it.

```shell
go get github.com/pilotso11/lazywritercache
```

You will need to provide 2 structures to the cache. 

### 1. CacheReaderWriter
You will need a CacheReaderWriter implementation.   This interface abstracts the cache away from any specific persistence mechanism.
It has functions to look up cache items, store cache items and manage transactions.
Any of these can be no-op operations if you don't need them.
A GORM implementation is provided. 
```go
import "github.com/pilotso11/lazywritercache"

type CacheReaderWriter[K,T] interface {
	Find(key K, tx any) (T, error)
	Save(item T, tx any) error
	BeginTx() (tx any, err error)
	CommitTx(tx any)
	Info(msg string)
	Warn(msg string)
}
```
### 2. A Cachable structure
Cacheable strucs must implement a function to return their key and a function to sync up their data post storage.  If your database schema as dynamically allocated primary keys you need this because
you won't know the key at save time (I'm assuming you have a secondary unique key that you are using to lookup the items).
```go
type Cacheable interface {
	Key() any
	CopyKeyDataFrom(from Cacheable) Cacheable // This should copy in DB only ID fields.  If gorm.Model is implement this is ID, creationTime, updateTime, deleteTime
}
```
For example, if you were using GORM and your type implements `gorm.Model` your copy function would look like:
```go
func (data MyData) CopyKeyDataFrom(from lazywritercache.Cacheable) lazywritercache.CacheItem {
    fromData := from.(MyData)  
    
    data.ID = fromData.ID
    data.CreatedAt = fromData.CreatedAt
    data.UpdatedAt = fromData.UpdatedAt
    data.DeletedAt = fromData.DeletedAt
    
    return data
}

```
Two sample `CacheReaderWriter` implementations are provided.  A no-op implementation for unit testing
and a simple [GORM](https://gorm.io) implementation that looks up items with a key field and saves them back.

The test code is an illustration of the no-op implementation.  The [example](/example) folder has
as sample that use GORM and postgres.

Assuming you have a `CacheReaderWriter` implementation, and you've created your `Cacheable`, then creating your cache is 
straightforward.
```go
readerWriter := lazywritercache.NewGormCacheReaderWriter[string, Person](db, NewEmptyPerson)
cacheConfig := lazywritercache.NewDefaultConfig[string, Person](readerWriter)
cache := lazywritercache.NewLazyWriterCache[string, Person](cacheConfig)
```
