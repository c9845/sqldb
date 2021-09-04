/*
deploy-schema.go defines tooling for updating a database schema. Each query used
to update a table, update an index, modify data, or something else must be
encapsulated in a func that runs Exec on the query the returns the error.

It is best to name funcs that deploy the schema in a manner such as follows. This
allows for better organization of code and errors when updating the database.
	- UpdateTableUsers.

This is very similar to deploying the database but specifically handles post-deploy
updates to the schema or data stored in the database.
*/
package sqldb

import (
	"reflect"
	"runtime"

	"github.com/pkg/errors"
)

//UpdateFunc is the format for a function used to update the database schema. A
//func of this format contains a query to update a table's schema or modify data
//in a table and returns the error from calling Exec().
type UpdateFunc func() error

//UpdateSchema updates an already deployed schema by running the update funcs defined
//on a database config.
func (c *Config) UpdateSchema() (err error) {
	//make sure the database isn't already connected, if so, disconnect
	if c.connection != nil {
		err = c.Close()
		if err != nil {
			return
		}
	}

	//connect to the database.
	err = c.Connect()
	if err != nil {
		return
	}
	defer c.Close()

	//run each update func
	for _, f := range c.UpdateFuncs {
		//get name of deploy func for use in error for better diagnostics.
		funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		// funcName := rawNameWithPath[strings.LastIndex(rawNameWithPath, "db"):]

		err = f()
		if err != nil {
			return errors.Wrap(err, "Error in update func "+funcName)
		}
	}

	//close the connection. you need to reestablish the connection separately.
	//this is done so that you don't need to check if the database is already
	//connected.
	err = c.Close()

	return
}

//handleUpdateSchemaErrors handles when an error is returned from an alter
//command to update the db schema.  This func is used so that we do not need
//to copy/paste error handling for each table we update.  All error handling is
//done here (determining what error occured, is error valid, etc.) and the error
//is returned back if needed.
/* func handleUpdateSchemaErrors(alterQuery string, err error) error {
	//make sure an error was provided
	if err == nil {
		return nil
	}

	if isError, isIgnored := isSQLiteModifyTableError(alterQuery, err); isError {
		//this error handler returns true when:
		// - sqlite db is in use, and...
		// - alter command is trying to modify a column, drop a table, add constraint
		// - the --ignore-sqlite-modify-schema-errors flag was NOT provided
		//This err should be returned so user can fix the error or use the --ignore flag.
		if isIgnored {
			log.Println("**Ignoring error (isSQLiteModifyTableError)", err)
			return nil
		}

		return err

	} else if isTableDoesNotExistError(err) {
		//table may have already been dropped by previous run of --update-db.
		log.Println("Does the table need to be created with the --deploy-db flag first?")
		return err

	} else if isCantDropError(err) {
		//column may have already been dropped by previous run of --update-db.
		log.Println("**Ignoring error (isCantDropError)", err)
		return nil

	} else if isUnknownColumnError(err) {
		//column may have been dropped
		log.Println("**Ignoring error (isUnknownColumnError)", err)
		return nil

	} else if isDuplicateColumnError(err) {
		//column already exists
		log.Println("**Ignoring error (isDuplicateColumnError)", err)
		return nil

	} else if isDuplicateConstraintError(alterQuery, err) {
		//key/index already exists
		log.Println("**Ignoring error (isDuplicateConstraintError)", err)
		return nil

	} else {
		//some unknown error occured
		return errors.Wrap(err, "generic db update error")
	}
}
*/
//isSQLiteModifyTableError checks if an error from a sql query to update a table is due to:
// - trying to modify the table, change a column type (sqlite doesn't support).
// - trying to drop foreign key constraint (sqlite doesn't support).
// - trying to add a constraint (sqlite doesn't support).
/* func isSQLiteModifyTableError(alterQuery string, err error) (isErr, isIgnored bool) {
	//only sqlite dbs
	if IsMariaDB() {
		return false, false
	}

	if strings.Contains(err.Error(), `near "CHANGE"`) && strings.Contains(err.Error(), "syntax error") {
		if configSaved.IgnoreSQLiteModifyErrors == false {
			log.Println("SQLite does not allow for modifying column types. Either copy the table's data, destroy and recreate the table with new column type, and copy back data or use old column type (use the --ignore-sqlite-modify-schema-errors flag with --update-db).")
			return true, false
		}

		log.Println("**SQLite cannot modify column types, error ignored.")
		return true, true
	}

	if strings.Contains(err.Error(), `near "FOREIGN"`) && strings.Contains(alterQuery, "DROP FOREIGN KEY") {
		if configSaved.IgnoreSQLiteModifyErrors == false {
			log.Println("SQLite cannot drop foreign keys. Use the --ignore-sqlite-modify-schema-errors flag with --update-db to skip this error.")
			return true, false
		}

		log.Println("**SQLite cannot drop foreign keys, error ignored.")
		return true, true
	}

	if strings.Contains(err.Error(), `near "CONSTRAINT"`) && strings.Contains(alterQuery, "ADD CONSTRAINT") {
		if configSaved.IgnoreSQLiteModifyErrors == false {
			log.Println("SQLite cannot add constraints. Use the --ignore-sqlite-modify-schema-errors flag with --update-db to skip this error).")
			return true, false
		}

		log.Println("**SQLite cannot add constraints, error ignored.")
		return true, true
	}

	return false, false
} */

//isDuplicateColumnError checks if an error was generated while updating the schema because
//a column already exists. These errors can be safely ignored since a duplicate column won't
//be create.
/* func isDuplicateColumnError(err error) bool {
	//strings.ToLower needed b/c sqlite uses lowercase vs mariadb use uppercase D in "duplicate".
	return strings.Contains(strings.ToLower(err.Error()), "duplicate column")
} */

//isUnknownColumnError checks if an error from a sql query is being kicked out because
//the column doensn't exist in the table. This is usually triggered when dropping a column
//that has already been dropped.
/* func isUnknownColumnError(err error) bool {
	if IsMariaDB() {
		if strings.Contains(err.Error(), "Unknown column") {
			return true
		}
	}

	//for sqlite, you should use the isSQLiteModifyTableError func to handle this

	return false
} */

//isCantDropError checks if an error was generated because something cannot be dropped. Typically
//this is used for dropping columns and handles columns that have already been dropped. Can also
//be used for dropping foreign keys.
/* func isCantDropError(err error) bool {
	if strings.Contains(err.Error(), "Can't DROP COLUMN") && strings.Contains(err.Error(), "check that it exists") {
		return true
	}

	if strings.Contains(err.Error(), "Can't DROP FOREIGN KEY") && strings.Contains(err.Error(), "check that it exists") {
		return true
	}

	if strings.Contains(err.Error(), "no such column") {
		return true
	}

	return false
} */

//isColumnDoesNotExistError checks if an error was generated because a column doesn't not exist.
//This may occur when trying to drop a column that has already been dropped (rerunning --update-db)
//or something more serious like a column was dropped by mistake by user interacting with database
//directly.
//The difficulty with this func is determining when a missing column is supposed to be missing, and
//thus we can ignore the error, or when a missing column is important and we should not ignore the
//error.
// func isColumnDoesNotExistError(err error) bool {
// 	return false
// }

//isTableDoesNotExistError checks if a table does not exist.
//The difficulty with this func is determining when a missing table is supposed to be missing (db update
//dropped it) and user is reruning update-db and we can ignore the error, or when missing table is
//important and we should not ignore the error.
/* func isTableDoesNotExistError(err error) bool {
	if strings.Contains(err.Error(), "Table") && strings.Contains(err.Error(), "doesn't exist") {
		return true
	}

	return false
} */

//isTableAlreadyExistsError checks if an error from a sql query to rename a table is being
//kicked out because the new table name already exists.
// func isTableAlreadyExistsError(err error) bool {
// 	if IsMariaDB() {
// 		if strings.Contains(err.Error(), "Table") && strings.Contains(err.Error(), "already exists") {
// 			return true
// 		}
// 	}

// 	return false
// }

//isDuplicateConstraintError checks if we are trying to add a constraint that already exists
//constraint was most likely created in db deploy or a previous update
//Adding index/key.
/* func isDuplicateConstraintError(alterQuery string, err error) bool {
	if strings.Contains(alterQuery, "ADD CONSTRAINT") == true && strings.Contains(err.Error(), "Duplicate key on write or update") == true {
		return true

	} else if strings.Contains(alterQuery, "ADD CONSTRAINT") == true && strings.Contains(err.Error(), "Duplicate key name") == true {
		return true

	} else if strings.Contains(alterQuery, "ADD INDEX") == true && strings.Contains(err.Error(), "Duplicate key name") == true {
		return true
	}

	return false
} */
