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
//on a database config. This will create the database if needed. Typically this is used
//to deploy an empty, or near empty, database. A database connection must not already
//be established; this func will establish the connection the leave it open for further
//use.
//
//Typically this func would be called when your app is passed a flag, such as --deploy-db,
//so that your database is only deployed when needed, not as part of every start of
//your app.
//
//The dontInsert parameter is used prevent any DeployQueries with "INSERT INTO" statements
//from running. This is used to deploy a completely empty database.
func (c *Config) DeploySchema(dontInsert bool) (err error) {
	//Make sure the connection isn't already established to prevent overwriting anything.
	//This forces users to call Close() first to prevent any incorrect db usage.
	if c.Connected() {
		return ErrConnected
	}

	//Make sure the config is valid.
	err = c.validate()
	if err != nil {
		return
	}

	//Get the connection string used to connect to the database. The returned string
	//will not included the db name (for non-sqlite dbs) since the db isn't deployed
	//yet.
	connString := c.buildConnectionString(true)

	//Get the correct driver based on the database type.
	//Error should never occur this since we already validated the config in validate().
	driver, err := getDriver(c.Type)
	if err != nil {
		return
	}

	//Connect to the database (really just the database server, or file for sqlite,
	//since the specific database itself is not created yet).
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

	//Reconnect to the database since the previously used connection didn't include
	//the database name in the connection string. This will connect us to the specific
	//database, not the database server. This connects using Connect(), the same func
	//that would be used to connect to the db for normal usage.
	//
	//This is not necessary for SQLite since SQLite always connects to the filepath
	//that was provided, not a server.
	if !c.IsSQLite() {
		err = conn.Close()
		if err != nil {
			return
		}

		//Note, no `defer Close()` since we want to leave the connection to the db
		//open upon successfully deploying the db so that db can be used without
		//calling `Connect()` after this func.
		err = c.Connect()
		if err != nil {
			return
		}
	}

	//Run each deploy query.
	c.log("sqldb.DeploySchema (DeployQueries)...")
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

	//Not closing the connection upon success since user may want to start interacting
	//with the db right away and this removes the need to call Connect() right after
	//this func.

	return
}

//DeploySchema deploys the database for the default package level config.
func DeploySchema(dontInsert bool) (err error) {
	return config.DeploySchema(dontInsert)
}
