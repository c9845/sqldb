package sqldb

import (
	"path"
	"reflect"
	"runtime"
)

// UpdateSchemaOptions provides options when updating a schema.
type UpdateSchemaOptions struct {
	// CloseConnection determines if the database connection should be closed after
	// running all the DeployQueries and DeployFuncs.//
	//
	//This was added to support deploying and then using a SQLite in-memory databse.
	//Each connection to an in-memory database references a new database, so to run
	//queries against an in-memory database that was just deployed, we need to keep
	//the connection open.
	CloseConnection bool //default true
}

// UpdateSchema runs the UpdateQueries and UpdateFuncs specified in a config against
// the database noted in the config. Use this to add columns, add indexes, rename
// things, perform data changes, etc.
//
// UpdateQueries will be translated via UpdateQueryTranslators and any UpdateQuery
// errors will be processed by UpdateQueryErrorHandlers. Neither of these steps apply
// to UpdateFuncs.
//
// UpdateSchemaOptions is a pointer so that in cases where you do not want to provide
// any options, using the defaults, you can simply provide nil.
//
// Typically this func is run when a flag, i.e.: --update-db, is provided.
func (c *Config) UpdateSchema(opts *UpdateSchemaOptions) (err error) {
	//Set default opts if none were provided.
	if opts == nil {
		opts = &UpdateSchemaOptions{
			CloseConnection: true,
		}
	}

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

	//Run each UpdateQuery.
	c.infoLn("sqldb.UpdateSchema", "Running UpdateQueries...")
	for _, q := range c.UpdateQueries {
		//Translate.
		q = c.RunUpdateQueryTranslators(q)

		//Log for diagnostics. Seeing queries is sometimes nice to see what is
		//happening.
		//
		//Trim logging length just to prevent super long queries from causing long
		//logging entries.
		if len(q) > 50 {
			c.infoLn("UpdateQuery:", q[:70]+"...")
		} else {
			c.infoLn("UpdateQuery:", q)
		}

		//Execute the query. If an error occurs, check if it should be ignored.
		_, innerErr := connection.Exec(q)
		if innerErr != nil && !c.runUpdateQueryErrorHandlers(q, innerErr) {
			err = innerErr
			c.errorLn("sqldb.UpdateSchema", "Error with query.", q, err)
			c.Close()
			return
		}
	}
	c.infoLn("sqldb.UpdateSchema", "Running UpdateQueries...done")

	//Run each UpdateFunc.
	c.infoLn("sqldb.UpdateSchema", "Running UpdateFuncs...")
	for _, f := range c.UpdateFuncs {
		//Get function name for diagnostic logging, since for UpdateQueries above we
		//log out some or all of each query.
		rawNameWithPath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName := path.Base(rawNameWithPath)
		c.infoLn("UpdateFunc:", funcName)

		//Execute the func.
		innerErr := f(connection)
		if innerErr != nil {
			err = innerErr
			c.errorLn("sqldb.UpdateSchema", "Error with UpdateFunc.", funcName, err)
			c.Close()
			return innerErr
		}
	}
	c.infoLn("sqldb.UpdateSchema", "Running UpdateFuncs...done")

	//Close the connection to the database, if needed.
	if opts.CloseConnection {
		c.Close()
		c.debugLn("sqldb.UpdateSchama", "Connection closed after success.")
	} else {
		c.debugLn("sqldb.UpdateSchama", "Connection left open after success.")
	}

	return
}

// UpdateSchema runs the UpdateQueries and UpdateFuncs specified in a config against
// the database noted in the config. Use this to add columns, add indexes, rename
// things, perform data changes, etc.
//
// UpdateQueries will be translated via UpdateQueryTranslators and any UpdateQuery
// errors will be processed by UpdateQueryErrorHandlers. Neither of these steps apply
// to UpdateFuncs.
//
// UpdateSchemaOptions is a pointer so that in cases where you do not want to provide
// any options, using the defaults, you can simply provide nil.
//
// Typically this func is run when a flag, i.e.: --update-db, is provided.
func UpdateSchema(opts *UpdateSchemaOptions) (err error) {
	return cfg.UpdateSchema(opts)
}

// RunUpdateQueryTranslators runs the list of UpdateQueryTranslators on the provided
// query.
//
// This func is called in UpdateSchema().
func (c *Config) RunUpdateQueryTranslators(in string) (out string) {
	out = in
	for _, t := range c.UpdateQueryTranslators {
		out = t(out)
	}

	return out
}

// RunUpdateQueryTranslators runs the list of UpdateQueryTranslators on the provided
// query.
//
// This func is called in UpdateSchema().
func RunUpdateQueryTranslators(in string) (out string) {
	out = in
	for _, t := range cfg.UpdateQueryTranslators {
		out = t(out)
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
