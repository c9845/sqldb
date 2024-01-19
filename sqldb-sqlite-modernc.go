//go:build modernc

/*
This file handles the [modernc.org/sqlite] SQLite library.

This library is not the default since it doesn't use the SQLite source C code and
isn't as widely used.

However, this library is straight golang and does not require CGO which makes cross-
compiling much easier.
*/

package sqldb

import (
	_ "modernc.org/sqlite"
)

const (
	//sqliteLibrary is used in logging.
	sqliteLibrary = "modernc.org/sqlite"

	//sqliteDriverName is used in Connect() when calling [database/sql.Open].
	sqliteDriverName = "sqlite"
)

// SQLiteDefaultPragmas defines the list of PRAGMA statments to configure SQLite that
// we use by default.
//
// The [github.com/mattn/go-sqlite3] library sets some PRAGMAs by default. The
// [modernc.org/sqlite] library does not define any default PRAGMAs. However, to
// make switching between the two database libraries/drivers easier, we define
// some PRAGMAs here to make using a SQLite database more consistent.
//
// https://github.com/mattn/go-sqlite3/blob/ae2a61f847e10e6dd771ecd4e1c55e0421cdc7f9/sqlite3.go#L1086
var SQLiteDefaultPragmas = []string{
	"PRAGMA busy_timeout = 5000",
	"PRAGMA synchronous = NORMAL",
}
