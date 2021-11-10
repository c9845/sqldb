package sqldb

import (
	"strings"
)

//TranslateFunc is function used to translate a query from one database format
//to another. This is used when you write your queries for one database (ex.: MySQL)
//but you allow your app to be deployed in multiple database formats (ex.: MySQL &
//SQLite). These funcs perform the necessary conversions on a query so that you do
//not need to write your queries in each database format.
type TranslateFunc func(string) string

//translateCreateTable runs the TranslateCreateTableFuncs funcs defined for a database
//connection when DeploySchema() is called.
func (c *Config) translateCreateTable(in string) (out string) {
	//working copy of query to modify as needed.
	query := in

	//Make sure query is a CREATE TABLE query.
	if !strings.Contains(strings.ToUpper(in), "CREATE TABLE") {
		out = query
		return
	}

	//run each translate funcs
	for _, f := range c.TranslateCreateTableFuncs {
		query = f(query)
	}

	//return the translated query
	out = query
	return out
}

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
	out = in

	before := "PRIMARY KEY(ID)"
	after := ""
	primaryKeyIndex := strings.Index(in, before)
	choppedQ := out[:primaryKeyIndex]
	lastCommaIndex := strings.LastIndex(choppedQ, ",")
	out = out[:lastCommaIndex] + out[lastCommaIndex+1:]
	out = strings.Replace(out, before, after, 1)
	return
}

//TFToSQLiteReformatDefaultTimestamp handles converting UTC_TIMESTAMP values to
//CURRENT_TIMESTAMP values. On MySQL and MariaDB, both UTC_TIMESTAMP and
//CURRENT_TIMESTAMP values exist, with CURRENT_TIMESTAMP returning a datetime
//in the server's local timezone. However, SQLite doesn't have UTC_TIMESTAMP
//and CURRENT_TIMESTAMP returns values in UTC timezone.
func TFToSQLiteReformatDefaultTimestamp(in string) (out string) {
	before := "DEFAULT UTC_TIMESTAMP"
	after := "DEFAULT CURRENT_TIMESTAMP"
	out = strings.Replace(in, before, after, -1)
	return
}

//TFToSQLiteReformatDatetime replaces DATETIME columns with TEXT columns. SQLite
//doesn't have a DATETIME column so values stored in these columns can be converted
//oddly. Use TEXT column type for SQLite b/c SQLite golang driver converts DATETIME
//columns in yyyy-mm-dd hh:mm:ss format to yyyy-mm-ddThh:mm:ssZ upon returning value
//which isn'texpected or what we would usually want; instead user can reformat value
//returned from database as needed using time package.
func TFToSQLiteReformatDatetime(in string) (out string) {
	before := "DATETIME"
	after := "TEXT"
	out = strings.Replace(in, before, after, -1)
	return
}
