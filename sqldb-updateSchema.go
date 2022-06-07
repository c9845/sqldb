package sqldb

import (
	"context"
	"log"
	"strings"
)

//UpdateSchemaOptions provides options when updating a schema.
type UpdateSchemaOptions struct {
	CloseConnection bool
}

//UpdateSchemaWithOps updates a database by running the list of UpdateQueries defined
//in config. This is typically used to add new colums, alter columns, add indexes, or
//updates values stored in the database.
//
//Although each UpdateQuery should be indempotent, you should still not call this func
//each time your app starts or otherwise. Typically you would check if the database
//has already been updated or use a flag, such as  --update-db, to run this func.
//
//When each UpdateQuery is run, if an error occurs the error is passed into each defined
//UpdateIgnoreErrorFuncs to determine if and how the error needs to be handled.
//Sometimes an error during a schema update isn't actually an error we need to handle,
//such as adding a column that already exists. Most times these types of errors occur
//because the UpdateSchema func is being rerun. The list of funcs you add to0
//UpdateIgnoreErrorFuncs will check the returned error message and query and determine
//if the error can be ignored.
func (c *Config) UpdateSchemaWithOps(ops UpdateSchemaOptions) (err error) {
	//Check if a connection to the database is already established, and if so, use it.
	if !c.Connected() {
		err = c.Connect()
		if err != nil {
			return
		}
	}

	//Check if the connection should be closed after this func completes.
	if ops.CloseConnection {
		defer c.Close()
	}

	//Make sure the config is valid.
	err = c.validate()
	if err != nil {
		return
	}

	//Start a transaction. We use a transaction to update the schema so that either
	//the entire database is updated successfully or none of the database is updated.
	//This prevents the database from being in a half-updated state.
	ctx := context.Background()
	connection := c.Connection()
	tx, err := connection.BeginTxx(ctx, nil)
	if err != nil {
		c.Close()
		return
	}
	defer tx.Rollback()

	//Run each update query.
	c.debugPrintln("sqldb.UpdateSchema...")
	for _, q := range c.UpdateQueries {
		//Log out some info about the query being run for diagnostics.
		const trimLength = 80 //arbitrary number, longer shows more info but can clog up terminal output.
		if len(q) > trimLength {
			c.debugPrintln(strings.TrimSpace(q[:trimLength]) + "...")
		} else {
			c.debugPrintln(q)
		}

		//Execute the query. Always log on error so users can identify query that has
		//an error.
		_, innerErr := tx.ExecContext(ctx, q)
		if innerErr != nil && !c.ignoreUpdateSchemaErrors(q, innerErr) {
			log.Println("sqldb.UpdateSchema() error with query", q, innerErr)
			c.Close()
			return innerErr
		}
	}
	c.debugPrintln("sqldb.UpdateSchema...done")

	//Commit transaction now that all UpdateQueries have been run successfully..
	err = tx.Commit()
	if err != nil {
		c.Close()
		return
	}

	if ops.CloseConnection {
		//close is handeld by defer above.
		c.debugPrintln("Connection closed upon successful deploy.")
	} else {
		c.debugPrintln("Connection left open after successful deploy.")
	}

	return
}

//UpdateSchemaWithOps updates the database for the default package level config.
func UpdateSchemaWithOps(ops UpdateSchemaOptions) (err error) {
	return config.UpdateSchemaWithOps(ops)
}

//UpdateSchema runs UpdateSchemaWithOps with some defaults set. This was implemented
//to support legacy compatibility while expanding the feature set with update options.
func (c *Config) UpdateSchema() (err error) {
	ops := UpdateSchemaOptions{
		CloseConnection: true,
	}
	return c.UpdateSchemaWithOps(ops)
}

//UpdateSchema runs UpdateSchemaWithOps with some defaults set for the default package
//level config. This was implemented to to support legacy compatibility while expanding
//the feature set with update options.
func UpdateSchema() (err error) {
	return config.UpdateSchema()
}

//ignoreUpdateSchemaErrors handles when an error is returned from an UpdateQuery when
//run from UpdateSchema(). This is used to handle queries that can fail and aren't really
//an error (i.e.: adding a column that already exists). Excusable errors can happen
//because UpdateQueries should be able to run more than once (i.e.: if you run UpdateSchema()
//each time your app starts).
//
//The query to update the schema is passed in so that we can check what an error is in
//relation to. Sometimes the error returned doesn't provide enough context.
func (c *Config) ignoreUpdateSchemaErrors(query string, err error) bool {
	//make sure an error was provided
	if err == nil {
		return true
	}

	//Run each UpdateIngoreErrorFunc. This will check if the error returned from running
	//the query can be safely ignored. Once one function returns "true" (to ignore the
	//error that occured), the other functions are skipped.
	for _, f := range c.UpdateIgnoreErrorFuncs {
		ignore := f(*c, query, err)
		if ignore {
			return true
		}
	}

	return false
}

//UpdateIgnoreErrorFunc is function for handling errors returned when trying to update
//the schema of your database using UpdateSchema(). The query being run, as well as the
//error from running the query, are passed in so that the function can determine if this
//error can be ignored for this query. Each function of this type, and used for this
//purpose should be very narrowly focused so as not to ignore errors by mistake (false
//positives).
type UpdateIgnoreErrorFunc func(Config, string, error) bool

//UFAddDuplicateColumn checks if an error was generated because a column already exists.
//This typically happens because you are rerunning UpdateSchema() and the column has
//already been added. This error can be safely ignored since a duplicate column won't
//be create.
func UFAddDuplicateColumn(c Config, query string, err error) bool {
	addCol := strings.Contains(strings.ToUpper(query), "ADD COLUMN")
	dup := strings.Contains(strings.ToLower(err.Error()), "duplicate column")

	if addCol && dup {
		c.debugPrintln("  Ignoring query, " + err.Error())
		return true
	}

	return false
}

//UFDropUnknownColumn checks if an error from was generated because a column does not exist.
//This typically happens because you are rerunning UpdateSchema() and the column has
//already been dropped. This error can be safely ignored in most cases.
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

//UFModifySQLiteColumn checks if an error occured because you are trying to modify a
//column for a SQLite database. SQLite does not allow modifying columns. In this case,
//we just ignore the error. This is ok since SQLite allows you to store any type of value
//in any column.
//
//To get around this error, you should create a new table with the new schema, copy the
//old data to the new table, delete the old table, and rename the new table to the old
//table.
func UFModifySQLiteColumn(c Config, query string, err error) bool {
	//ignore queries that modify a column for sqlite dbs
	if strings.Contains(strings.ToUpper(query), "MODIFY COLUMN") && c.Type == DBTypeSQLite {
		c.debugPrintln("  Ignoring query, " + err.Error())
		return true
	}

	return false
}

//UFAlreadyExists handles errors when an index already exists. This may also work for
//other thngs that already exist (columns). If you use "IF NOT EXISTS" in your query to
//add a column or index this function will not be used since IF NOT EXISTS doesn't return
//an error if the item already exists.
func UFAlreadyExists(c Config, query string, err error) bool {
	if strings.Contains(err.Error(), "already exists") {
		c.debugPrintln("  Ignoring query, " + err.Error())
		return true
	}

	return false
}
