package sqldb

import (
	"strings"
)

/*
This file list a bunch of example Translator funcs. These funcs are used to
translate a DeployQuery or UpdateQuery from one SQL database dialect to another.
*/

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

// TranslateMariaDBToSQLite translates a query written in MariaDB format to SQLite
// format. This translator is meant to be used for CREATE TABLE and ALTER TABLE
// queries only.
func TranslateMariaDBToSQLite(query string) string {
	//Change INT to INTEGER since SQLite defines INTEGER as a data type and INT as a
	//type affinity. This is really just an idiomatic thing and doesn't affect queries
	//at all.
	//
	//Spaces around INT and INTEGER to try and prevent mistaken replacement if the
	//characters "int" end up in another word, i.e.: a column name.
	before := " INT "
	after := " INTEGER "
	query = strings.ReplaceAll(query, before, after)

	//Reformat the ID column.
	before = "ID INTEGER NOT NULL AUTO_INCREMENT"
	after = "ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL"
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

	//Change TINYBLOB, MEDIUMBLOB, and LONGBLOB columns to just BLOB.
	query = strings.ReplaceAll(query, "TINYBLOB", "BLOB")
	query = strings.ReplaceAll(query, "MEDIUMBLOB", "BLOB")
	query = strings.ReplaceAll(query, "LONGBLOB", "BLOB")

	//Convert VARCHAR(...) definitions to TEXT since SQLite doesn't use VARCHAR.
	//Note for/while loop, since a query could use VARCHAR multiple times!
	for {
		const searchFor = "VARCHAR"
		const swapWith = "TEXT"

		if !strings.Contains(query, searchFor) {
			break
		}

		//Get index of start of VARCHAR.
		start := strings.Index(query, searchFor)

		//Get index of closing parenthesis after VARCHAR.
		end := strings.Index(query[start:], ")") + 1 + start

		//Get text between start and end, i.e. the VARCHAR(...) definition.
		definition := query[start:end]

		//Replace VARCHAR(...) with TEXT.
		query = strings.Replace(query, definition, swapWith, 1)
	}

	//Convert DECIMAL(...) definitions to REAL since SQLite doesn't use DECIMAL.
	//Note for/while loop, since a query could use DECIMAL multiple times!
	for {
		const searchFor = "DECIMAL"
		const swapWith = "REAL"

		if !strings.Contains(query, searchFor) {
			break
		}

		//Get index of start of DECIMAL.
		start := strings.Index(query, searchFor)

		//Get index of closing parenthesis after DECIMAL.
		end := strings.Index(query[start:], ")") + 1 + start

		//Get text between start and end, i.e. the DECIMAL(...) definition.
		definition := query[start:end]

		//Replace DECIMAL(...) with TEXT.
		query = strings.Replace(query, definition, swapWith, 1)
	}

	//Convert BOOL or BOOLEAN columns to INTEGER since SQLite doesn't use BOOL.
	query = strings.ReplaceAll(query, " BOOLEAN", " INTEGER")
	query = strings.ReplaceAll(query, " BOOL", " INTEGER")

	//Convert DATE columns to TEXT since SQLite doesn't use DATE.
	query = strings.ReplaceAll(query, " DATE", " TEXT")

	//Convert TIME columns to TEXT since SQLite doesn't use TIME.
	query = strings.ReplaceAll(query, " TIME", " TEXT")

	return query
}
