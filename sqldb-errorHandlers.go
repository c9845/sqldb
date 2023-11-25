package sqldb

import "strings"

//This file lists a bunch of example ErrorHandler funcs. These funcs are used to
//ignore/bypass errors returned from running a DeployQuery or UpdateQuery.

// IgnoreErrorDuplicateColumn checks if an error occured because a column with the
// same name already exists. This is useful to for running ALTER TABLE ADD COLUMN or
// RENAME COLUMN.
//
// This error usually occurs because UpdateSchema() is being rerun.
func IgnoreErrorDuplicateColumn(query string, err error) bool {
	if strings.Contains(strings.ToUpper(query), "ADD COLUMN") && strings.Contains(err.Error(), "duplicate column") {
		return true
	}

	return false
}

// IgnoreErrorDropUnknownColumn checks if an error occured because a column you are
// trying to DROP has already been DROPped and does not exist.
//
// This error usually occurs because UpdateSchema() is being rerun.
func IgnoreErrorDropUnknownColumn(query string, err error) bool {
	if strings.Contains(strings.ToUpper(query), "DROP COLUMN") {
		//MariaDB
		if strings.Contains(err.Error(), "check that it exists") {
			return true
		}

		//SQLite
		if strings.Contains(err.Error(), "no such column") {
			return true
		}
	}

	return false
}

// IgnoreErrorSQLiteModify checks if an error occured becaused a query is attempting
// to MODIFY a column for a SQLite database. SQLite does not support MODIFY. This is
// somewhat okay since SQLite allows storing any type of data in any colum (unless
// STRICT tables are used) and the data read from the database will be read into a
// Golang type anyway.
//
// To get around this error, you should create a new table with the new schema, copy
// the old data to the new table, delete the old table, and rename the new table to
// the old table.
func IgnoreErrorSQLiteModify(query string, err error) bool {
	//lint:ignore S1008 - i don't really like "return strings.Contains()", i don't think it is as clear.
	if strings.Contains(query, "MODIFY COLUMN") {
		return true
	}

	return false
}

// IgnoreErrorIndexAlreadyExists checks if an error occured because an index with the
// given name already exists. If you use "IF NOT EXISTS" in your query this error will
// not occur.
func IgnoreErrorIndexAlreadyExists(c Config, query string, err error) bool {
	if strings.Contains(query, "CREATE INDEX") {
		//MariaDB
		if strings.Contains(err.Error(), "duplicate key name") {
			return true
		}

		//SQLite
		if strings.Contains(err.Error(), "already exists") {
			return true
		}
	}

	return false
}
