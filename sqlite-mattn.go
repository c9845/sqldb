//go:build !modernc

/*
This file handles the [github.com/mattn/go-sqlite3] SQLite library.

This library is the default SQLite library if no build tags are provided. Note the
"go:build !modernc" line.

This library requires CGO, and therefore requires a bit more work to get cross-
compiling to work properly.
*/

package sqldb

import (
	_ "github.com/mattn/go-sqlite3"
)

const (
	//sqliteLibrary is used in logging.
	sqliteLibrary = sqliteLibraryMattn

	//sqliteDriverName is used in Connect() when calling [database/sql.Open].
	sqliteDriverName = "sqlite3"
)
