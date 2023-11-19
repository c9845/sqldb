package sqldb

import (
	"log"
	"path"
	"reflect"
	"runtime"

	"github.com/jmoiron/sqlx"
)

// DeployFunc is a function used to perform a deployment task that is more complex
// than just a SQL query that could be provided in a DeployQuery.
type DeployFunc func(*sqlx.DB) error

// DeploySchemaOptions provides options when deploying a schema.
type DeploySchemaOptions struct {
	// CloseConnection determines if the database connection should be closed after
	// running all the DeployQueries and DeployFuncs.//
	//
	//This was added to support deploying and then using a SQLite in-memory databse.
	//Each connection to an in-memory database references a new database, so to run
	//queries against an in-memory database that was just deployed, we need to keep
	//the connection open.
	CloseConnection bool
}

// DeploySchema runs the DeployQueries and DeployFuncs specified in a config against
// the database noted in the config. Use this to create your tables, create indexes,
// etc. This will automatically issue a CREATE DATABASE IF NOT EXISTS query.
//
// DeployQueries will be translated via DeployQueryTranslators and any DeployQuery
// errors will be processed by DeployQueryErrorHandlers. Neither of these steps apply
// to DeployFuncs.
//
// Typically this func is run when a flag, i.e.: --deploy-db, is provided.
func (c *Config) DeploySchema(ops DeploySchemaOptions) (err error) {
	//Make sure the connection isn't already established to prevent overwriting it.
	//This forces users to call Close() first to prevent any errors.
	if c.Connected() {
		return ErrConnected
	}

	//Make sure the config is valid.
	err = c.validate()
	if err != nil {
		return
	}

	//Build the connection string used to connect to the database.
	//
	//The returned string will not included the database name (for non-SQLite) since
	//the database may not been deployed yet.
	connString := c.buildConnectionString(true)

	//Get the correct driver based on the database type.
	//
	//If using SQLite, the correct driver is chosen based on build tags.
	driver := getDriver(c.Type)

	//Create the database, if it doesn't already exist.
	//
	//For MariaDB/MySQL, we need to create the actual database on the server.
	//For SQLite , we need to Ping() the connection so the file is created on disk.
	conn, err := sqlx.Open(driver, connString)
	if err != nil {
		return
	}
	defer conn.Close()

	switch c.Type {
	case DBTypeMySQL, DBTypeMariaDB, DBTypeMSSQL:
		q := `CREATE DATABASE IF NOT EXISTS ` + c.Name
		_, innerErr := conn.Exec(q)
		if innerErr != nil {
			err = innerErr
			return
		}
	case DBTypeSQLite:
		err = conn.Ping()
		if err != nil {
			return
		}
	}

	//Reconnect to the database since the previously used connection string did not
	//include the database name (for non-SQLite). This will connect us to the specific
	//database, not just the database server. This connects using Connect(), the same
	//function that would be used to connect to the database for normal usage.
	err = conn.Close()
	if err != nil {
		return
	}

	err = c.Connect()
	if err != nil {
		return
	}

	//Skip closing the connection if user wants to leave connection open. This only
	//matters when an error does not occur, when an error occurs below Close() is
	//manually called.
	if ops.CloseConnection {
		defer c.Close()
	}

	//Get connection to use for deploying.
	connection := c.Connection()

	//Run each DeployQuery.
	c.infoPrintln("sqldb.DeploySchema", "Running DeployQueries...")
	for _, q := range c.DeployQueries {
		//Translate.
		q := c.RunDeployQueryTranslators(q)

		//Log for diagnostics.
		if len(q) > 50 {
			c.infoPrintln(q[:50])
		} else {
			c.infoPrintln(q)
		}

		//Execute the query. If an error occurs, check if it should be ignored.
		_, innerErr := connection.Exec(q)
		if innerErr != nil && !c.runDeployQueryErrorHandlers(q, innerErr) {
			err = innerErr
			log.Println("sqldb.DeploySchema", "Error with query.", q, err)
			c.Close()
			return
		}
	}
	c.infoPrintln("sqldb.DeploySchema", "Running DeployQueries...done")

	//Run each DeployFunc.
	c.infoPrintln("sqldb.DeploySchema", "Running DeployFuncs...")
	for _, f := range c.DeployFuncs {
		//Get function name for diagnostic logging, since for DeployQueries above we
		//log out some or all of each query.
		rawNameWithPath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName := path.Base(rawNameWithPath)
		c.infoPrintln(funcName)

		//Execute the func.
		innerErr := f(connection)
		if innerErr != nil {
			err = innerErr
			log.Println("sqldb.DeploySchema", "Error with DeployFunc.", funcName, err)
			c.Close()
			return
		}
	}
	c.infoPrintln("sqldb.DeploySchema", "Running DeployFuncs...done")

	//Close the connection to the database, if needed.
	if ops.CloseConnection {
		c.Close()
		c.debugPrintln("sqldb.DeploySchema", "Connection closed after successful deploy.")
	} else {
		c.debugPrintln("sqldb.DeploySchema", "Connection left open after successful deploy.")
	}

	return
}

// RunDeployQueryTranslators runs the list of DeployQueryTranslators on the provided
// query. This is run in Deploy().
func (c *Config) RunDeployQueryTranslators(in string) (out string) {
	for _, t := range c.DeployQueryTranslators {
		out = t(in)
	}

	return out
}

// runDeployQueryErrorHandlers runs the list of DeployQueryErrorHandlers when an error
// occured from running a DeployQuery. This is run in Deploy().
func (c *Config) runDeployQueryErrorHandlers(query string, err error) (ignoreError bool) {
	//Make sure an error occured.
	if err == nil {
		return true
	}

	//Run each DeployQueryErrorHandler and see if any return true to ignore this error.
	for _, eh := range c.DeployQueryErrorHandlers {
		ignoreError = eh(query, err)
		if ignoreError {
			return
		}
	}

	return false
}
