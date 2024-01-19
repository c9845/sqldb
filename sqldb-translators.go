package sqldb

import (
	"strings"
)

//This file list a bunch of example Translator funcs. These funcs are used to
//translate a DeployQuery or UpdateQuery from one SQL database dialect to another.

// Translator is a function that translates a DeployQuery or UpdateQuery from one SQL
// dialect to another. Translators run when DeploySchema() or UpdateSchame() is
// called.
//
// Translators typically have an "is this translator applicable, perform the
// translation" format.
//
// Ex:
//
//	func TranslateDatetimeToText (in query) string {
//	  if !strings.Contains(in, "DATETIME") {
//		    return in
//	  }
//
//	  return strings.Replace(in, "DATETIME", "TEXT")
//	 }
type Translator func(string) string

// TranslateMariaDBToSQLiteCreateTable translates a CREATE TABLE query from MariaDB
// or MySQL to SQLite.
func TranslateMariaDBToSQLiteCreateTable(query string) string {
	//Only applies to CREATE TABLE queries...
	if !strings.Contains(query, "CREATE TABLE") {
		return query
	}

	//Reformat the ID column.
	before := "ID INT NOT NULL AUTO_INCREMENT"
	after := "ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL"
	query = strings.Replace(query, before, after, 1)

	//Remove the PRIMARY KEY definition. SQLite doesn't use PRIMARY KEY(ID), the
	//primary key is defined as part of the column definition (see above).
	before = "PRIMARY KEY(ID)"
	after = ""
	if strings.Contains(query, before) {
		//Find the "PRIMARY KEY(ID)" in the query.
		primaryKeyIndex := strings.Index(query, before)

		//Get the query up to the "PRIMARY KEY(ID)".
		beforePrimaryKeyDeclaration := query[:primaryKeyIndex]

		//Find the last comma before the "PRIMARY KEY(ID)". We need to remove this
		//comma otherwise we will get an invalid query format error, aka an extra
		//comma exists somewhere.
		lastCommaIndex := strings.LastIndex(beforePrimaryKeyDeclaration, ",")

		//Remove the comma.
		query = query[:lastCommaIndex] + query[lastCommaIndex+1:]

		//Remove the "PRIMARY KEY(ID)". We don't need to worry if a comma exists after
		//the "PRIMARY KEY(ID)" since even if one does, we already removed a preceeding
		//comma so we won't end up with two commas in a row.
		query = strings.Replace(query, before, after, 1)
	}

	//Change UTC_TIMESTAMP to CURRENT_TIMESTAMP. SQLite doesn't have UTC_TIMESTAMP,
	//but CURRENT_TIMESTAMP returns a UTC datetime.
	before = "DEFAULT UTC_TIMESTAMP"
	after = "DEFAULT CURRENT_TIMESTAMP"
	query = strings.ReplaceAll(query, before, after)

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
	query = strings.ReplaceAll(query, before, after)

	//Change INT to INTEGER since SQLite defines INTEGER as a data type and INT as a
	//type affinity. This is really just an idiomatic thing and doesn't affect queries
	//at all. Spaces around INT and INTEGER to try and prevent mistaken replacement if
	//the characters "int" end up in another word, i.e.: a column name.
	before = " INT "
	after = " INTEGER "
	query = strings.ReplaceAll(query, before, after)

	//Change TINYBLOB, MEDIUMBLOB, and LONGBLOB columns to just BLOB.
	query = strings.ReplaceAll(query, "TINYBLOB", "BLOB")
	query = strings.ReplaceAll(query, "MEDIUMBLOB", "BLOB")
	query = strings.ReplaceAll(query, "LONGBLOB", "BLOB")

	return query
}
