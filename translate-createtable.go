/*
translate-createtable.go handles converting a CREATE TABLE query from one database
type to another. This is used when your app supports being deployed with multiple
database types but you only want to write your CREATE TABLE query in one database's
format. This works by rewriting the query into the target database format prior to
running the query with Exec().
*/
package sqldb

import "strings"

//TranslateFunc is the format of functions run against a query to translate it from
//one database format to another format.
type TranslateFunc func(string) string

//TFToSQLiteReformatID reformats the ID column to a SQLite format.
func TFToSQLiteReformatID(in string) (out string) {
	before := "ID INT NOT NULL AUTO_INCREMENT"
	after := "ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL"
	out = strings.Replace(in, before, after, 1)
	return
}

//TFToSQLiteRemovePrimaryKeyDefinition removes the primary key definition
//for a SQLite query since SQLite doesn't use this. We also have to remove
//the comma preceeding this line too.
func TFToSQLiteRemovePrimaryKeyDefinition(in string) (out string) {
	before := "PRIMARY KEY(ID)"
	after := ""
	primaryKeyIndex := strings.Index(in, before)
	choppedQ := out[:primaryKeyIndex]
	lastCommaIndex := strings.LastIndex(choppedQ, ",")
	out = out[:lastCommaIndex] + out[lastCommaIndex+1:]
	out = strings.Replace(out, before, after, 1)
	return
}

//TFHandleSQLiteDefaultTimestamp handles converting UTC_TIMESTAMP values to
//CURRENT_TIMESTAMP values. On MySQL and MariaDB, both UTC_TIMESTAMP and
//CURRENT_TIMESTAMP values exist, with CURRENT_TIMESTAMP returning a datetime
//in the server's local timezone. However, SQLite doesn't have UTC_TIMESTAMP
//and CURRENT_TIMESTAMP returns values in UTC timezone.
func TFHandleSQLiteDefaultTimestamp(in string) (out string) {
	before := "DEFAULT UTC_TIMESTAMP"
	after := "DEFAULT CURRENT_TIMESTAMP"
	out = strings.Replace(in, before, after, -1)
	return
}

//TFHandleSQLiteDatetimeColumns replaces DATETIME columns with TEXT columns. SQLite
//doesn't have a DATETIME column so values stored in these columns can be converted
//oddly. Use TEXT column type for SQLite b/c SQLite golang driver converts DATETIME
//columns in yyyy-mm-dd hh:mm:ss format to yyyy-mm-ddThh:mm:ssZ upon returning value
//which isn'texpected or what we would usually want; instead user can reformat value
//returned from database as needed using time package.
func TFHandleSQLiteDatetimeColumns(in string) (out string) {
	before := "DATETIME"
	after := "TEXT"
	out = strings.Replace(in, before, after, -1)
	return
}

//TranslateCreateTable runs the TranslateCreateTableFuncs funcs defined for a database
//connection. This func would be used prior to Exec-ing the query. Note that this func
//does not check what type of database a query is written in or what type of database
//a query should be translated to, this just runs all the translation funcs defined.
func (c *Config) TranslateCreateTable(in string) (out string) {
	//run the translate funcs
	for _, f := range c.TranslateCreateTableFuncs {
		out = f(in)
	}

	//return the translated query
	return out
}
