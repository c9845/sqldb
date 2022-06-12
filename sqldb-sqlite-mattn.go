//!modernc causes this file to be included by default if no -tags are provided to
//go build or go run. AKA, default to using mattn if no build tag is provided.

//mattn required CGO. modernc does not which allows for easier cross compiled builds.

//go:build !modernc

package sqldb

import (
	_ "github.com/mattn/go-sqlite3"
)

const (
	sqliteLibrary    = "github.com/mattn/go-sqlite3"
	sqliteDriverName = "sqlite3"
)

//mattn/sqlite3 already sets some default PRAGMAS as noted in the following link.
//https://github.com/mattn/go-sqlite3/blob/ae2a61f847e10e6dd771ecd4e1c55e0421cdc7f9/sqlite3.go#L1086
var sqliteDefaultPragmas = []string{}
