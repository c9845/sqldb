package sqldb

import (
	"log"
	"path"
	"reflect"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
)

//DeployFunc is the format for a function used to deploy part of the schema.
type DeployFunc func(*sqlx.DB) error

//DeploySchemaOptions provides options when deploying a schema.
type DeploySchemaOptions struct {
	SkipInsert      bool
	CloseConnection bool
}

//DeploySchemaWithOps deploys the database schema by running the list of DeployQueries
//and DeployFuncs defined in config. This will create the database if needed. This is
//typically used to deploy an empty, or near empty, database. A database connection
//must not already be established; this func will establish the connection.
//
//Although each DeployQuery and DeployFunc should be indempotent (ex.: using CREATE
//TABLE IF NOT EXISTS), you should still not call this func each time your app starts
//or otherwise. Typically you would check if the database already exists or use a
//flag, such as --deploy-db, to run this func.
//
//skipInsert is used prevent any DeployQueries with "INSERT INTO" statements from
//running. This is used to deploy a completely empty database and is useful for
//migrating data or backups.
//
//closeConnection determines if the database connection should be closed after  this
//func successfully completes. This was added to support SQLite in-memory databases
//since each connection to an im-memory db uses a new database, so if we deploy with
//a connection we need to reuse it to run queries.
func (c *Config) DeploySchemaWithOps(ops DeploySchemaOptions) (err error) {
	//Make sure a connection isn't already established to prevent overwriting anything.
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

	//Create the database.
	//For mariadb/mysql, we need to create the actual database on the server.
	//For SQLite , we need to Ping() the connection so the file is created on disk.
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
	//database, not just the database server. This connects using Connect(), the same
	//func that would be used to connect to the db for normal usage.
	err = conn.Close()
	if err != nil {
		return
	}

	err = c.Connect()
	if err != nil {
		return
	}

	if ops.CloseConnection {
		defer c.Close()
	}

	//Run each deploy query.
	c.debugPrintln("sqldb.DeploySchema (DeployQueries)...")
	connection := c.Connection()
	for _, q := range c.DeployQueries {
		//Translate the query if needed. This will only translate queries with
		//CREATE TABLE in the text.
		q = c.translateCreateTable(q)

		//Skip queries that insert data if needed.
		if strings.Contains(strings.ToUpper(q), "INSERT INTO") && ops.SkipInsert {
			continue
		}

		//Log out some info about the query being run for diagnostics.
		if strings.Contains(q, "CREATE TABLE") {
			idx := strings.Index(q, "(")
			c.debugPrintln(strings.TrimSpace(q[:idx]) + "...")
		} else {
			c.debugPrintln(q)
		}

		//Execute the query. Always log on error so users can identify query that has
		//an error.
		_, innerErr := connection.Exec(q)
		if innerErr != nil {
			err = innerErr
			log.Println("sqldb.DeploySchema() error with query", q)
			c.Close()
			return
		}
	}
	c.debugPrintln("sqldb.DeploySchema (DeployQueries)...done")

	//Run each deploy func.
	c.debugPrintln("sqldb.DeploySchema (DeployFuncs)...")
	for _, f := range c.DeployFuncs {
		//Get function name for diagnostics.
		rawNameWithPath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName := path.Base(rawNameWithPath)

		//Log out some infor about the func being run for diagnostics.
		c.debugPrintln(funcName)

		//Execute the func. Always log on error so users can identify query that has
		//an error.
		innerErr := f()
		if innerErr != nil {
			log.Println("sqldb.DeploySchema() error with deploy func", funcName)
			c.Close()
			return innerErr
		}

	}
	c.debugPrintln("sqldb.DeploySchema (DeployFuncs)...done")

	if ops.CloseConnection {
		//close is handled by defer above.
		c.debugPrintln("Connection closed upon successful deploy.")
	} else {
		c.debugPrintln("Connection left open after successful deploy.")
	}

	return
}

//DeploySchemaWithOps deploys the database for the default package level config.
func DeploySchemaWithOps(ops DeploySchemaOptions) (err error) {
	return config.DeploySchemaWithOps(ops)
}

//DeploySchema runs DeploySchemaWithOps with some defaults set. This was implemented
//to support legacy compatibility while expanding the feature set with deploy options.
func (c *Config) DeploySchema(skipInsert bool) (err error) {
	ops := DeploySchemaOptions{
		SkipInsert:      skipInsert,
		CloseConnection: true, //legacy
	}
	return c.DeploySchemaWithOps(ops)
}

//DeploySchema runs DeploySchemaWithOps with some defaults set for the default package
//level config. This was implemented to support legacy compatibility while expanding
//the feature set with deploy options.
func DeploySchema(skipInsert bool) (err error) {
	return config.DeploySchema(skipInsert)
}
