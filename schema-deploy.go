package sqldb

import (
	"path"
	"reflect"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
)

// DeploySchemaOptions provides options when deploying a schema.
type DeploySchemaOptions struct {
	//CloseConnection determines if the database connection should be closed after
	//running all the DeployQueries and DeployFuncs.//
	//
	//This was added to support deploying and then using a SQLite in-memory databse.
	//Each connection to an in-memory database references a new database, so to run
	//queries against an in-memory database that was just deployed, we need to keep
	//the connection open.
	CloseConnection bool //default true
}

// DeploySchema runs the DeployQueries and DeployFuncs specified in a config against
// the database noted in the config. Use this to create your tables, create indexes,
// etc. This will automatically issue a CREATE DATABASE IF NOT EXISTS query.
//
// DeployQueries will be translated via DeployQueryTranslators and any DeployQuery
// errors will be processed by DeployQueryErrorHandlers. Neither of these steps apply
// to DeployFuncs.
//
// DeploySchemaOptions is a pointer so that in cases where you do not want to provide
// any options, using the defaults, you can simply provide nil.
//
// Typically this func is run when a flag, i.e.: --deploy-db, is provided.
func (c *Config) DeploySchema(opts *DeploySchemaOptions) (err error) {
	//Set default opts if none were provided.
	if opts == nil {
		opts = &DeploySchemaOptions{
			CloseConnection: true,
		}
	}

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
	//the database may not have been deployed yet.
	c.debugLn("sqldb.DeploySchema", "Getting connection string before deploying...")
	connString := c.buildConnectionString(true)

	//Get the correct driver based on the database type.
	//
	//If using SQLite, the correct driver is chosen based on build tags.
	driver := getDriver(c.Type)

	//Create the database, if it doesn't already exist.
	//
	//For MariaDB/MySQL, we need to create the actual database on the server.
	//For SQLite, we need to Ping() the connection so the file is created on disk.
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

	c.debugLn("sqldb.DeploySchema", "Connecting to deployed database...")
	err = c.Connect()
	if err != nil {
		return
	}

	//Skip closing the connection if user wants to leave connection open after this
	//function completes. Leaving the connection open is important for handling
	//SQLite in-memory databases.
	//
	//This is only effective when an error does not occur in the below code. When an
	//error occurs, Close() is always called.
	if opts.CloseConnection {
		defer c.Close()
	}

	//Get connection to use for deploying.
	connection := c.Connection()

	//Run each DeployQuery.
	c.infoLn("sqldb.DeploySchema", "Running DeployQueries...")
	for _, q := range c.DeployQueries {
		//Translate.
		q := c.RunDeployQueryTranslators(q)

		//Log for diagnostics. Seeing queries is sometimes nice to see what is
		//happening.
		//
		//Trim logging length just to prevent super long queries from causing long
		//logging entries.
		ql, _, found := strings.Cut(strings.TrimSpace(q), "\n")
		if found {
			c.infoLn("DeployQuery:", ql)
		} else if maxLen := 70; len(q) > maxLen {
			c.infoLn("DeployQuery:", q[:maxLen]+"...")
		} else {
			c.infoLn("DeployQuery:", q)
		}

		//Execute the query. If an error occurs, check if it should be ignored.
		_, innerErr := connection.Exec(q)
		if innerErr != nil && !c.runDeployQueryErrorHandlers(q, innerErr) {
			err = innerErr
			c.errorLn("sqldb.DeploySchema", "Error with query.", q, err)
			c.Close()
			return
		}
	}
	c.infoLn("sqldb.DeploySchema", "Running DeployQueries...done")

	//Run each DeployFunc.
	c.infoLn("sqldb.DeploySchema", "Running DeployFuncs...")
	for _, f := range c.DeployFuncs {
		//Get function name for diagnostic logging, since for DeployQueries above we
		//log out some or all of each query.
		rawNameWithPath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName := path.Base(rawNameWithPath)
		c.infoLn("DeployFunc:", funcName)

		//Execute the func.
		innerErr := f(connection)
		if innerErr != nil {
			err = innerErr
			c.errorLn("sqldb.DeploySchema", "Error with DeployFunc.", funcName, err)
			c.Close()
			return
		}
	}
	c.infoLn("sqldb.DeploySchema", "Running DeployFuncs...done")

	//Close the connection to the database, if needed.
	if opts.CloseConnection {
		c.Close()
		c.debugLn("sqldb.DeploySchema", "Connection closed after successful deploy.")
	} else {
		c.debugLn("sqldb.DeploySchema", "Connection left open after successful deploy.")
	}

	return
}

// DeploySchema runs the DeployQueries and DeployFuncs specified in a config against
// the database noted in the config. Use this to create your tables, create indexes,
// etc. This will automatically issue a CREATE DATABASE IF NOT EXISTS query.
//
// DeployQueries will be translated via DeployQueryTranslators and any DeployQuery
// errors will be processed by DeployQueryErrorHandlers. Neither of these steps apply
// to DeployFuncs.
//
// DeploySchemaOptions is a pointer so that in cases where you do not want to provide
// any options, using the defaults, you can simply provide nil.
//
// Typically this func is run when a flag, i.e.: --deploy-db, is provided.
func DeploySchema(opts *DeploySchemaOptions) (err error) {
	return cfg.DeploySchema(opts)
}

// RunDeployQueryTranslators runs the list of DeployQueryTranslators on the provided
// query.
//
// This func is called in DeploySchema() but can also be called manually when you want
// to translate a DeployQuery (for example, running a specific DeployQuery as part of
// UpdateSchema).
func (c *Config) RunDeployQueryTranslators(in string) (out string) {
	out = in
	for _, t := range c.DeployQueryTranslators {
		out = t(out)
	}

	return out
}

// RunDeployQueryTranslators runs the list of DeployQueryTranslators on the provided
// query.
//
// This func is called in DeploySchema() but can also be called manually when you want
// to translate a DeployQuery (for example, running a specific DeployQuery as part of
// UpdateSchema).
func RunDeployQueryTranslators(in string) (out string) {
	out = in
	for _, t := range cfg.DeployQueryTranslators {
		out = t(out)
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
