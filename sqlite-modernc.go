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
	sqliteLibrary = sqliteLibraryModernc

	//sqliteDriverName is used in Connect() when calling [database/sql.Open].
	sqliteDriverName = "sqlite"
)
