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
	//used to match value set for mattn/sqlite3.
	//https://github.com/mattn/go-sqlite3/blob/ae2a61f847e10e6dd771ecd4e1c55e0421cdc7f9/sqlite3.go#L1088
	"PRAGMA busy_timeout = 5000",

	//not setting synchronous mode if journal mode is WAL, per link below, since then
	//we have to handle if user provided another synchronous mode pragma and the default
	//sync mode of FULL should rarely impact performance notably with WAL.
	//https://github.com/mattn/go-sqlite3/blob/ae2a61f847e10e6dd771ecd4e1c55e0421cdc7f9/sqlite3.go#L1308
}
