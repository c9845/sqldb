package sqldb

import (
	"log"
	"path"
	"reflect"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
)

// UpdateFunc is a function used to perform an update schema taks that is more complex
// thatn just a SQL query that could be provided in an UpdateQuery
type UpdateFunc func(*sqlx.DB) error

// UpdateSchemaOptions provides options when updating a schema.
type UpdateSchemaOptions struct {
	// CloseConnection determines if the database connection should be closed after
	// running all the DeployQueries and DeployFuncs.//
	//
	//This was added to support deploying and then using a SQLite in-memory databse.
	//Each connection to an in-memory database references a new database, so to run
	//queries against an in-memory database that was just deployed, we need to keep
	//the connection open.
	CloseConnection bool
}

// UpdateSchema runs the UpdateQueries and UpdateFuncs specified in a config against
// the database noted in the config. Use this to add columns, add indexes, rename
// things, perform data changes, etc.
//
// UpdateQueries will be translated via UpdateQueryTranslators and any UpdateQuery
// errors will be processed by UpdateQueryErrorHandlers. Neither of these steps apply
// to UpdateFuncs.
//
// Typically this func is run when a flag, i.e.: --update-db, is provided.
func (c *Config) UpdateSchemaWithOps(ops UpdateSchemaOptions) (err error) {
	//Check if a connection to the database is already established, and if so, use it.
	//If not, try to connect.
	//
	//This differs from Deploy(), where if a connection already exists, we exit, so
	//that we can support the Deploy option CloseConnection being false. I.e.: we want
	//to use the same connection we deployed with to update the database. This is used
	//mostly for SQLite in-memory dbs where we need to reuse the same connection.
	if !c.Connected() {
		err = c.Connect()
		if err != nil {
			return
		}
	}

	//Make sure the config is valid.
	err = c.validate()
	if err != nil {
		return
	}

	//Check if the connection should be closed after this func completes.
	if ops.CloseConnection {
		defer c.Close()
	}

	//Get connection to use for deploying.
	connection := c.Connection()

	//Run each UpdateQuery.
	c.infoPrintln("sqldb.UpdateSchema", "Running UpdateQueries...")
	for _, q := range c.UpdateQueries {
		//Translate.
		q = c.RunUpdateQueryTranslators(q)

		//Log for diagnostics.
		if len(q) > 50 {
			c.infoPrintln(q[:50])
		} else {
			c.infoPrintln(q)
		}

		//Execute the query. If an error occurs, check if it should be ignored.
		_, innerErr := connection.Exec(q)
		if innerErr != nil && !c.runUpdateQueryErrorHandlers(q, innerErr) {
			err = innerErr
			log.Println("sqldb.UpdateSchema", "Error with query.", q, err)
			c.Close()
			return
		}
	}
	cfg.infoPrintln("sqldb.UpdateSchema", "Running UpdateQueries...done")

	//Run each UpdateFunc.
	cfg.infoPrintln("sqldb.UpdateSchema", "Running UpdateFuncs...")
	for _, f := range cfg.UpdateFuncs {
		//Get function name for diagnostic logging, since for UpdateQueries above we
		//log out some or all of each query.
		rawNameWithPath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName := path.Base(rawNameWithPath)
		cfg.infoPrintln(funcName)

		//Execute the func.
		innerErr := f(connection)
		if innerErr != nil {
			err = innerErr
			cfg.errorPrintln("sqldb.UpdateSchema", "Error with UpdateFunc.", funcName, err)
			cfg.Close()
			return innerErr
		}
	}
	cfg.infoPrintln("sqldb.UpdateSchema", "Running UpdateFuncs...done")

	//Close the connection to the database, if needed.
	if ops.CloseConnection {
		cfg.Close()
		cfg.debugPrintln("sqldb.UpdateSchama", "Connection closed after success.")
	} else {
		cfg.debugPrintln("sqldb.UpdateSchama", "Connection left open after success.")
	}

	return
}

// RunUpdateQueryTranslators runs the list of UpdateQueryTranslators on the provided
// query. This is run in Update().
func (c *Config) RunUpdateQueryTranslators(in string) (out string) {
	for _, t := range c.UpdateQueryTranslators {
		out = t(in)
	}

	return out
}

// runUpdateQueryErrorHandlers runs the list of UpdateQueryErrorHandlers when an error
// occured from running a UpdateQuery. This is run in Update().
func (c *Config) runUpdateQueryErrorHandlers(query string, err error) (ignoreError bool) {
	//Make sure an error occured.
	if err == nil {
		return true
	}

	//Run each UpdateQueryErrorHandler and see if any return true to ignore this error.
	for _, eh := range c.UpdateQueryErrorHandlers {
		ignoreError = eh(query, err)
		if ignoreError {
			return
		}
	}

	return false
}

//
//
//
//
//
//

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
