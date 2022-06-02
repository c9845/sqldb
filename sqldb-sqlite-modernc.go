//go:build !modernc

package sqldb

import (
	_ "modernc.org/sqlite"
)

const (
	sqliteLibrary    = "modernc.org/sqlite"
	sqliteDriverName = "sqlite"
)
