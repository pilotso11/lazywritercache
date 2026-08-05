module github.com/pilotso11/lazywritercache/examples

go 1.23.10

replace github.com/pilotso11/lazywritercache => ../

replace github.com/pilotso11/lazywritercache/lazygormcache => ../lazygormcache

require (
	github.com/pilotso11/lazywritercache v0.1.5
	github.com/pilotso11/lazywritercache/lazygormcache v0.0.0-00010101000000-000000000000
	github.com/xo/dburl v0.23.8
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.30.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.5 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/puzpuzpuz/xsync v1.5.2 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/text v0.26.0 // indirect
)
