package sqldb

import (
	"log"
	"path"
	"reflect"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
)

//DeployFunc is the format of a function used to deploy the database schema. The type
//is defined for easier use when defining the list of DeployFuncs versus having to
//type "cfg.DeployFuncs = []func(*sqlx.DB) error {...}".
type DeployFunc func(*sqlx.DB) error

//DeploySchemaOptions provides options when deploying a schema.
//
//SkipInsert is used prevent any DeployQueries with "INSERT INTO" statements from
//running. This is used to deploy a completely empty database and is useful for
//migrating data or backups.
//
//CloseConnection determines if the database connection should be closed after this
//func successfully completes. This was added to support SQLite in-memory databases
//since each connection to an im-memory db uses a new database, so if we deploy with
//a connection we need to reuse it to run queries.
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
func (cfg *Config) DeploySchemaWithOps(ops DeploySchemaOptions) (err error) {
	//Make sure a connection isn't already established to prevent overwriting anything.
	//This forces users to call Close() first to prevent any incorrect db usage.
	if cfg.Connected() {
		return ErrConnected
	}

	//Make sure the config is valid.
	err = cfg.validate()
	if err != nil {
		return
	}

	//Get the connection string used to connect to the database. The returned string
	//will not included the db name (for non-sqlite dbs) since the db isn't deployed
	//yet.
	connString := cfg.buildConnectionString(true)

	//Get the correct driver based on the database type. Error should never occur
	//since we already validated the config in validate().
	//
	//We can ignore the error here since an invalid Type would have already been caught
	//in .validate().
	driver, _ := getDriver(cfg.Type)

	//Connect to the database (really just the database server, or file for SQLite,
	//since the specific database itself is not created yet).
	conn, err := sqlx.Open(driver, connString)
	if err != nil {
		return
	}
	defer conn.Close()

	//Create the database.
	//For mariadb/mysql, we need to create the actual database on the server.
	//For SQLite , we need to Ping() the connection so the file is created on disk.
	switch cfg.Type {
	case DBTypeMySQL, DBTypeMariaDB:
		q := `CREATE DATABASE IF NOT EXISTS ` + cfg.Name
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

	err = cfg.Connect()
	if err != nil {
		return
	}

	//Skip closing the connection if user wants to leave connection open. This is
	//mostly used for SQLite in-memory dbs since each time the connection is closed
	//and reopenned, a new db is connected to.
	if ops.CloseConnection {
		defer cfg.Close()
	}

	//Get connection to use for deploying.
	connection := cfg.Connection()

	//Run each deploy query.
	cfg.debugPrintln("sqldb.DeploySchema", "Running DeployQueries...")
	for _, q := range cfg.DeployQueries {
		//Translate the query if needed. This will only translate queries with
		//CREATE TABLE in the text.
		q = cfg.runTranslateCreateTableFuncs(q)

		//Skip queries that insert data if needed.
		if strings.Contains(strings.ToUpper(q), "INSERT INTO") && ops.SkipInsert {
			continue
		}

		//Log out some info about the query being run for diagnostics.
		if strings.Contains(q, "CREATE TABLE") {
			idx := strings.Index(q, "(")
			if idx > 0 {
				cfg.debugPrintln(strings.TrimSpace(q[:idx]) + "...")
			}
		} else {
			cfg.debugPrintln(q)
		}

		//Execute the query. Always log on error so users can identify query that has
		//an error. Connection always gets closed since an error occured.
		_, innerErr := connection.Exec(q)
		if innerErr != nil {
			err = innerErr
			log.Println("sqldb.DeploySchema()", "Error with query.", q)
			cfg.Close()
			return
		}
	}
	cfg.debugPrintln("sqldb.DeploySchema", "Running DeployQueries...done")

	//Run each deploy func.
	cfg.debugPrintln("sqldb.DeploySchema", "Running DeployFuncs...")
	for _, f := range cfg.DeployFuncs {
		//Get function name for diagnostics.
		rawNameWithPath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName := path.Base(rawNameWithPath)

		//Log out some info about the func being run for diagnostics.
		cfg.debugPrintln(funcName)

		//Execute the func. Always log on error so users can identify func that has
		//an error. Connection always gets closed since an error occured.
		innerErr := f(connection)
		if innerErr != nil {
			err = innerErr
			log.Println("sqldb.DeploySchema()", "Error with func.", funcName)
			cfg.Close()
			return innerErr
		}
	}
	cfg.debugPrintln("sqldb.DeploySchema", "Running DeployFuncs...done")

	if ops.CloseConnection {
		//Close() is handled by defer above.
		cfg.debugPrintln("sqldb.DeploySchema()", "Connection closed after success.")
	} else {
		cfg.debugPrintln("sqldb.DeploySchema()", "Connection left open after success.")
	}

	return
}

//DeploySchemaWithOps deploys the database for the default package level config.
func DeploySchemaWithOps(ops DeploySchemaOptions) (err error) {
	return config.DeploySchemaWithOps(ops)
}

//DeploySchema runs DeploySchemaWithOps with some defaults set.
func (cfg *Config) DeploySchema(skipInsert bool) (err error) {
	ops := DeploySchemaOptions{
		SkipInsert:      skipInsert,
		CloseConnection: true,
	}
	return cfg.DeploySchemaWithOps(ops)
}

//DeploySchema runs DeploySchemaWithOps with some defaults set for the default package
//level config.
func DeploySchema(skipInsert bool) (err error) {
	return config.DeploySchema(skipInsert)
}
