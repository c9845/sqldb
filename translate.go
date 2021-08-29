/*
translate.go handles converting a CREATE TABLE query from one database type to
another. This is used when your app supports being deployed with multiple database
types but you only want to write your CREATE TABLE query in one database's format.
This works by rewriting the query into the target database format prior to running
the query with Exec().
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

//defaultTranslateCreateFuncs returns the list of translate funcs we have defined
//in this package. This func is used during New...Config() or Default...Config()
//to populate the config's TranslateCreateFuncs field.
func defaultTranslateCreateFuncs() []TranslateFunc {
	return []TranslateFunc{
		TFToSQLiteReformatID,
		TFToSQLiteRemovePrimaryKeyDefinition,
		TFHandleSQLiteDefaultTimestamp,
		TFHandleSQLiteDatetimeColumns,
	}
}

//TranslateCreate handles converting a CREATE query from one database format to
//another. This would be used prior to Exec-ing the query. This function just routes
//to the correct from-to specific database function.
func (c *Config) TranslateCreate(from, to dbType, query string) (out string) {
	if from == DBTypeMySQL && to == DBTypeSQLite {
		out = c.translateCreateFromMySQLToSQLite(query)

	} else if from == DBTypeMariaDB && to == DBTypeSQLite {
		out = c.translateCreateFromMariaDBToSQLite(query)

	} else {
		//unknown translation pair, just return original query
		out = query

	}

	return
}

//translateCreateFromMySQLToSQLite translates a CREATE query from a MySQL format to
//a SQLite format. MySQL and SQLite have some slight differences when it comes to
//creating a table. This func translates a MySQL formatted query into a format that
//will run on SQLite.
func (c *Config) translateCreateFromMySQLToSQLite(query string) string {
	//Don't modify the query if the database in use is in the same format as the
	//query.
	if c.IsMySQLOrMariaDB() {
		return query
	}

	//run the translate funcs
	for _, f := range c.TranslateCreateFuncs {
		query = f(query)
	}

	//return the translated query
	return query
}

//translateCreateFromMariaDBToSQLite translates a CREATE query from a MariaDB format
//to a SQLite format. This just repurposes the mysql -> sqlite translation since the
//mysql and mariadb formats are the same.
func (c *Config) translateCreateFromMariaDBToSQLite(query string) string {
	return c.translateCreateFromMySQLToSQLite(query)
}
