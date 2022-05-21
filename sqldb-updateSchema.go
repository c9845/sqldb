package sqldb

import (
	"context"
	"log"
	"strings"
)

//UpdateSchema updates an already database by running the list of UpdateQueries defined
//on a database config. Typically this is used to add new columns, alter columns, add
//new indexes, or change values stored in the database.
//
//When each UpdateQuery is run, if an error occurs the error is passed into each defined
//UpdateIgnoreErrorFuncs to determine if and how the error needs to be handled. Sometimes
//an error during a schema update isn't actually an error we need to handle, such as adding
//a column that already exists. Most times these types of errors occur because the
//UpdateSchema func is rerun. The list of funcs you add to UpdateIgnoreErrorFuncs will check
//the returned error message and query and determine if the error can be ignored.
func (c *Config) UpdateSchema() (err error) {
	//Make sure the connection isn't already established to prevent overwriting it. This
	//forces users to call Close() first to prevent any incorrect db usage.
	if c.Connected() {
		return ErrConnected
	}

	//Make sure the config is valid.
	err = c.validate()
	if err != nil {
		return
	}

	//Connect to the database.
	err = c.Connect()
	if err != nil {
		return
	}
	defer c.Close()

	//Get a transaction. We want to update the entire db, or none of it to
	//reduce the chances of odd issues.
	connection := c.Connection()
	ctx := context.Background()
	tx, err := connection.BeginTxx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	//Run each update query.
	if c.Debug {
		log.Println("sqldb.UpdateSchema...")
	}

	for _, q := range c.UpdateQueries {
		const trimLength = 80 //arbitrary number, longer is better
		if len(q) > trimLength {
			log.Println(strings.TrimSpace(q[:trimLength]) + "...")
		} else {
			log.Print(q)
		}

		_, innerErr := tx.ExecContext(ctx, q)
		if innerErr != nil {
			ignore := c.ignoreUpdateSchemaErrors(q, innerErr)
			if !ignore {
				if c.Debug {
					log.Println("sqldb.UpdateSchema() error with query", q, innerErr)
				}

				return innerErr
			}

		}
	} //end for: loop through update queries

	if c.Debug {
		log.Println("sqldb.UpdateSchema...done")
	}

	//Commit now that db has been completely and successfully updated.
	err = tx.Commit()
	if err != nil {
		return
	}

	//Close the connection. We don't want to leave this connection open for further
	//use just so that parent funcs can always assume the connection is closed.
	err = c.Close()

	return
}

//UpdateSchema updates the database for the default package level config.
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
		if c.Debug {
			log.Println("  Ignoring query, " + err.Error())
		}

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
		if c.Debug {
			log.Println("  Ignoring query, " + err.Error())
		}

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
		if c.Debug {
			log.Println("  Ignoring query, " + err.Error())
		}

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
		if c.Debug {
			log.Println("  Ignoring query, " + err.Error())
		}

		return true
	}

	return false
}
