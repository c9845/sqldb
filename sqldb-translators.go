package sqldb

import (
	"strings"
)

//This file list a bunch of example Translator funcs.

// TranslateMariaDBToSQLiteCreateTable translates a CREATE TABLE query from MariaDB
// or MySQL to SQLite.
func TFMySQLToSQLiteReformatID(query string) string {
	//Only applies to CREATE TABLE queries...
	if !strings.Contains(query, "CREATE TABLE") {
		return query
	}

	//Reformat the ID column.
	before := "ID INT NOT NULL AUTO_INCREMENT"
	after := "ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL"
	if strings.Contains(query, before) {
		query = strings.Replace(query, before, after, 1)
	}

	//Remove the PRIMARY KEY definition. SQLite doesn't use PRIMARY KEY(ID), the
	//primary key is defined as part of the column definition (see above).
	before = "PRIMARY KEY(ID)"
	after = ""
	if strings.Contains(query, before) {
		primaryKeyIndex := strings.Index(query, before)

		choppedQ := query[:primaryKeyIndex]
		lastCommaIndex := strings.LastIndex(choppedQ, ",")
		query = query[:lastCommaIndex] + query[lastCommaIndex+1:]
		query = strings.Replace(query, before, after, 1)
	}

	//Change UTC_TIMESTAMP to CURRENT_TIMESTAMP. SQLite doesn't have UTC_TIMESTAMP,
	//but CURRENT_TIMESTAMP returns a UTC datetime.
	before = "DEFAULT UTC_TIMESTAMP"
	after = "DEFAULT CURRENT_TIMESTAMP"
	if strings.Contains(query, before) {
		query = strings.ReplaceAll(query, before, after)
	}

	//Change DATETIME columns to TEXT. SQLite doesn't have a DATETIME column type,
	//and using it (as column type affinity) can cause issues due to the way each
	//SQLite library handles data conversion.
	//
	//The mattn/go-sqlite3 library converts DATETIME columns with data stored in
	//yyyy-mm-dd hh:mm:ss format to yyyy-mm-ddThh:mm:ssZ upon returning values (via
	//a SELECT query) which is unexpected. There may be other ways around preventing
	//this conversion, but converting to TEXT works and is inline with SQLite's column
	//types.
	before = "DATETIME"
	after = "TEXT"
	if strings.Contains(query, before) {
		strings.ReplaceAll(query, before, after)
	}

	return query
}
