package sqldb

import (
	"strings"
)

// runTranslateCreateTableFuncs runs the list of TranslateCreateTableFuncs funcs
// defined on a query when Deploy() or Update() is called.
func (cfg *Config) runTranslateCreateTableFuncs(originalQuery string) (translatedQuery string) {
	//Make sure query is a CREATE TABLE query.
	if !strings.Contains(strings.ToUpper(originalQuery), "CREATE TABLE") {
		return originalQuery
	}

	//Run each translate func. A query may be translated by multiple funcs.
	workingQuery := originalQuery
	for _, f := range cfg.TranslateCreateTableFuncs {
		workingQuery = f(workingQuery)
	}

	//Return the completely translated query.
	translatedQuery = workingQuery
	return
}

// TFMySQLToSQLiteReformatID reformats the ID column from a MySQL format to a SQLite
// format.
func TFMySQLToSQLiteReformatID(in string) (out string) {
	before := "ID INT NOT NULL AUTO_INCREMENT"
	after := "ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL"
	out = strings.Replace(in, before, after, 1)
	return
}

// TFMySQLToSQLiteRemovePrimaryKeyDefinition removes the primary key definition from a
// MySQL query for use in SQLite. SQLite doesn't use this PRIMARY KEY(ID), the PRIMARY
// KEY note is assigned as part of the column definition. We also have to remove the
// comma preceeding this line too since a trailing comma creates a bad query!
func TFMySQLToSQLiteRemovePrimaryKeyDefinition(in string) (out string) {
	out = in

	before := "PRIMARY KEY(ID)"
	after := ""
	primaryKeyIndex := strings.Index(in, before)
	if primaryKeyIndex == -1 {
		return
	}

	choppedQ := out[:primaryKeyIndex]
	lastCommaIndex := strings.LastIndex(choppedQ, ",")
	out = out[:lastCommaIndex] + out[lastCommaIndex+1:]
	out = strings.Replace(out, before, after, 1)

	return
}

// TFMySQLToSQLiteReformatDefaultTimestamp handles converting UTC_TIMESTAMP values to
// CURRENT_TIMESTAMP values. On MySQL and MariaDB, both UTC_TIMESTAMP and CURRENT_TIMESTAMP
// values exist, with CURRENT_TIMESTAMP returning a datetime in the server's local
// timezone. However, SQLite doesn't have UTC_TIMESTAMP and CURRENT_TIMESTAMP is
// different, it returns values in UTC timezone.
func TFMySQLToSQLiteReformatDefaultTimestamp(in string) (out string) {
	before := "DEFAULT UTC_TIMESTAMP"
	after := "DEFAULT CURRENT_TIMESTAMP"
	out = strings.Replace(in, before, after, -1)
	return
}

// TFMySQLToSQLiteReformatDatetime replaces DATETIME columns with TEXT columns. SQLite
// doesn't have a DATETIME column type so values stored in these columns can be converted
// oddly. Just use TEXT column type for SQLite for ease of use.
//
// The mattn/go-sqlite3 library converts DATETIME columns in yyyy-mm-dd hh:mm:ss format
// to yyyy-mm-ddThh:mm:ssZ upon returning values (via a SELECT query) which is unexpected
// since that is is not what is stored in the database when using the sqlite3 command
//
//line tool.
func TFMySQLToSQLiteReformatDatetime(in string) (out string) {
	before := "DATETIME"
	after := "TEXT"
	out = strings.Replace(in, before, after, -1)
	return
}
