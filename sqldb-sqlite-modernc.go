//go:build modernc

package sqldb

import (
	_ "modernc.org/sqlite"
)

const (
	sqliteLibrary    = "modernc.org/sqlite"
	sqliteDriverName = "sqlite"
)

var sqliteDefaultPragmas = []string{
	//The mattn/go-sqlite3 sets this value by default. Use this for modernc/sqlite as
	//well.
	//https://github.com/mattn/go-sqlite3/blob/ae2a61f847e10e6dd771ecd4e1c55e0421cdc7f9/sqlite3.go#L1086
	"PRAGMA busy_timeout = 5000",
}
