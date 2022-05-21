package sqldb

import (
	"log"
	"path"
	"reflect"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
)

//DeploySchema deploys the database schema by running the list of DeployQueries defined
//on a database config. This will create the database if needed. Typically this is
//used to deploy an empty, or near empty, database.
//
//Typically this func would be called when your app is passed a flag, such as --deploy-db.
//This is so that your database is only deployed when needed, not as part of the regular
//startup of your app.
//
//You should call os.Exit() after this func completes, in most cases, so that you are not
//tempted to call DeploySchema every time your app starts. Even upon successful deployment,
//calling os.Exit() is useful so that if you are using a flag to call this func, the end-user
//doesn't mistakenly put the flag in their script that starts this app.
//
//The dontInsert parameter is used prevent any DeployQueries with "INSERT INTO" statements
//from running. This is used to deploy a completely empty database.
func (c *Config) DeploySchema(dontInsert bool) (err error) {
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

	//Get the connection string used to connect to the database. The returned
	//string will not included the db name (for non-sqlite dbs) since the db
	//isn't deployed yet.
	connString := c.buildConnectionString(true)

	//Get the correct driver based on the database type.
	//This is set based on the empty (_) imported package.
	//Error should never occur this since we already validated the config in validate().
	driver, err := getDriver(c.Type)
	if err != nil {
		return
	}

	//Connect to the database (really just the database server since the specific
	//database itself is not created yet).
	conn, err := sqlx.Open(driver, connString)
	if err != nil {
		return
	}
	defer conn.Close()

	//Handle any database-type specific stuff.
	//For non-sqlite dbs, we need to create the actual database on the server.
	//For sqlite dbs, we need to Ping() the connection so the db file is created on disk.
	switch c.Type {
	case DBTypeMySQL, DBTypeMariaDB:
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

	//Disconnecting from database server since the connection doesn't include the
	//specific database name. We will reconnect utilizing the database name now that
	//it has been created. This really only needs to be done for non-SQLite dbs but
	//it is just easier to do it for all db types.
	err = conn.Close()
	if err != nil {
		return
	}

	//Connect to the database again, this time using the database name. This is the
	//same connection method as used if we aren't deploying. This isn't needed for
	//sqlite but we just do it for all db types since it is cleaner code-wise.
	err = c.Connect()
	if err != nil {
		return
	}
	defer c.Close()

	//Run each deploy query.
	if c.Debug {
		log.Println("sqldb.DeploySchema (DeployQueries)...")
	}
	for _, q := range c.DeployQueries {
		//Translate the query if needed. This will only translate queries with
		//CREATE TABLE in the text.
		q = c.translateCreateTable(q)

		//skip queries that insert data if needed. This will skip any query with
		//INSERT INTO in the text.
		if strings.Contains(strings.ToUpper(q), "INSERT INTO") && dontInsert {
			continue
		}

		if c.Debug {
			if strings.Contains(q, "CREATE TABLE") {
				idx := strings.Index(q, "(")
				log.Println(strings.TrimSpace(q[:idx]) + "...")
			} else {
				log.Println(q)
			}

		}

		//Execute the query.
		//Logging on error so users can identify query in question.
		connection := c.Connection()
		_, innerErr := connection.Exec(q)
		if innerErr != nil {
			err = innerErr
			log.Println("sqldb.DeploySchema() error with query", q)
			return
		}
	}
	if c.Debug {
		log.Println("sqldb.DeploySchema (DeployQueries)...done")
	}

	//Run each deploy func.
	if c.Debug {
		log.Println("sqldb.DeploySchema (DeployFuncs)...")
	}
	for _, f := range c.DeployFuncs {
		//get function name for diagnostics
		rawNameWithPath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName := path.Base(rawNameWithPath)

		if c.Debug {
			log.Println(funcName)
		}

		innerErr := f()
		if innerErr != nil {
			log.Println("Error with deploy func", funcName)
			return innerErr
		}

	}
	if c.Debug {
		log.Println("sqldb.DeploySchema (DeployFuncs)...done")
	}

	//Close the connection. We don't want to leave this connection open for further
	//use just so that parent funcs can always assume the connection is closed.
	err = c.Close()

	return
}

//DeploySchema deploys the database for the default package level config.
func DeploySchema(dontInsert bool) (err error) {
	return config.DeploySchema(dontInsert)
}
