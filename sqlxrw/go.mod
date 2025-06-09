module github.com/pilotso11/lazywritercache/sqlxrw

go 1.24.3

require (
	github.com/jmoiron/sqlx v1.4.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/pilotso11/lazywritercache v0.1.5
	github.com/stretchr/testify v1.10.0
)

replace github.com/pilotso11/lazywritercache => ../

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/puzpuzpuz/xsync v1.5.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
