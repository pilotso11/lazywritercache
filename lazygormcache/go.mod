module github.com/pilotso11/lazywritercache/lazygormcache

go 1.23.10

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/pilotso11/lazywritercache v0.1.5
	github.com/puzpuzpuz/xsync v1.5.2
	github.com/stretchr/testify v1.10.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.30.0
)

replace github.com/pilotso11/lazywritercache => ../

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.5 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
