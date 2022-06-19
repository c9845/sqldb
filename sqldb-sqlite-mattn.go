//!modernc causes this file to be included by default if no -tags are provided to
//go build or go run. AKA, default to using mattn if no build tag is provided.

//mattn requires CGO. modernc does not which allows for easier cross compiled builds.

//go:build !modernc

package sqldb

import (
	_ "github.com/mattn/go-sqlite3"
)

const (
	sqliteLibrary    = "github.com/mattn/go-sqlite3"
	sqliteDriverName = "sqlite3"
)

//Placeholder so that this variable is defined for this SQLite library.
var sqliteDefaultPragmas = []string{}
