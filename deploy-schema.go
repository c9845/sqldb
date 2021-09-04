/*
deploy-schema.go defines tooling for deploying a database schema. Each query used
to create a table, create an index, insert initial data, or something else must be
encapsulated in a func that runs Exec on the query the returns the error.

It is best to name funcs that deploy the schema in a manner such as follows. This
allows for better organization of code and errors when deploying the database.
	- CreateTableUsers, CreateTableAccounts.
	- CreateIndexOnTableUsers.
	- InsertInitialUsers, InsertInitialAccount.
*/
package sqldb

import (
	"reflect"
	"runtime"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

//DeployFunc is the format for a function used to deploy the database schema. A
//func of this format contains a query to deploy a table's schema and returns the
//error from calling Exec().
type DeployFunc func() error

//DeploySchema deploys the database schema by running the deploy funcs defined on a
//database config.
func (c *Config) DeploySchema() (err error) {
	//get the connection string used to connect to the database, making sure to
	//handle the fact that the database doesn't exist yet.
	connString := c.buildConnectionString(true)

	//connect to the database (really just the database server since the specific
	//database itself is not created yet)
	conn, err := sqlx.Open(c.driver, connString)
	if err != nil {
		return
	}
	defer conn.Close()

	//handle any database-type specific stuff
	switch c.Type {
	case DBTypeMySQL, DBTypeMariaDB:
		err = createDatabase(conn, c.Name)
		if err != nil {
			return
		}

	case DBTypeSQLite:
		//test connection
		//This actually establishes the connection so the database file is created
		//on disk. The file must exist so that the schema can be deployed.
		err = conn.Ping()
		if err != nil {
			return
		}
	}

	//disconnecting from database server since the connection doesn't include the
	//specific database name. we will reconnect utilizing the database name now that
	//it has been created (really only needed for non-SQLite dbs)
	err = conn.Close()
	if err != nil {
		return
	}

	//connect to the database again, this time using the database name. this is the
	//same connection method as used if we aren't deploying.
	err = c.Connect()
	if err != nil {
		return
	}
	defer c.Close()

	//run each deploy func
	for _, f := range c.DeployFuncs {
		//get name of deploy func for use in error for better diagnostics.
		funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		// funcName := rawNameWithPath[strings.LastIndex(rawNameWithPath, "db"):]

		err = f()
		if err != nil {
			return errors.Wrap(err, "Error in deploy func "+funcName)
		}
	}

	//close the connection. you need to reestablish the connection separately.
	//this is done so that you don't need to check if the database is already
	//connected.
	err = c.Close()

	return
}

//createDatabase creates a new database if a database with the same name doesn't
//already exist.
func createDatabase(c *sqlx.DB, dbName string) error {
	q := `CREATE DATABASE IF NOT EXISTS ` + dbName
	_, err := c.Exec(q)
	return err
}
