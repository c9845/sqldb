package sqldb

import "strings"

// UFAddDuplicateColumn checks if an error was generated because a column already
// exists. This typically happens because you are rerunning UpdateSchema() and the
// column has already been added. This error can be safely ignored since a duplicate
// column won't be create.
func UFAddDuplicateColumn(c Config, query string, err error) bool {
	addCol := strings.Contains(strings.ToUpper(query), "ADD COLUMN")
	dup := strings.Contains(strings.ToLower(err.Error()), "duplicate column")

	if addCol && dup {
		c.debugPrintln("  Ignoring query, " + err.Error())
		return true
	}

	return false
}

// UFDropUnknownColumn checks if an error from was generated because a column does not
// exist. This typically happens because you are rerunning UpdateSchema() and the
// column has already been dropped. This error can be safely ignored in most cases.
func UFDropUnknownColumn(c Config, query string, err error) bool {
	dropCol := strings.Contains(strings.ToUpper(query), "DROP COLUMN")

	//mysql & mariadb
	unknownM := strings.Contains(strings.ToLower(err.Error()), "check that it exists")

	//sqlite
	unknownS := strings.Contains(strings.ToLower(err.Error()), "no such column")

	if dropCol && (unknownM || unknownS) {
		c.debugPrintln("  Ignoring query, " + err.Error())
		return true
	}

	return false
}

// UFModifySQLiteColumn checks if an error occured because you are trying to modify a
// column for a SQLite database. SQLite does not allow modifying columns. In this case,
// we just ignore the error. This is ok since SQLite allows you to store any type of
// value in any column.
//
// To get around this error, you should create a new table with the new schema, copy
// the old data to the new table, delete the old table, and rename the new table to
// the old table.
func UFModifySQLiteColumn(c Config, query string, err error) bool {
	//ignore queries that modify a column for sqlite dbs
	if strings.Contains(strings.ToUpper(query), "MODIFY COLUMN") && c.Type == DBTypeSQLite {
		c.debugPrintln("  Ignoring query, " + err.Error())
		return true
	}

	return false
}

// UFIndexAlreadyExists handles errors when an index already exists. If you use
// "IF NOT EXISTS" in your query to add a column or index this function will not be
// used since IF NOT EXISTS doesn't return an error if the item already exists.
func UFIndexAlreadyExists(c Config, query string, err error) bool {
	createInx := strings.Contains(strings.ToUpper(query), "CREATE INDEX")

	//mysql & mariadb
	existsM := strings.Contains(strings.ToLower(err.Error()), "duplicate key name")

	//sqlite
	existsS := strings.Contains(strings.ToLower(err.Error()), "already exists")

	if createInx && (existsM || existsS) {
		c.debugPrintln("  Ignoring query, " + err.Error())
		return true
	}

	return false
}
