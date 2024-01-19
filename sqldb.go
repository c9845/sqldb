/*
Package sqldb provides tooling to make connecting to, deploying, updating, and using
a SQL database easier. This provides some wrapping around the [database/sql] package.

The initial purpose behind this package was to encapsulate commonly used database
connection, schema deploying and updating, and other boilerplate tasks.

# Global or Local DB Connection

You can use this package in two methods: as a singleton with the database configuration
and connection stored within this package in a globally accessible variable, or store
the configuration and connection somewhere else in your application. Storing the data
yourself allows for connecting to multiple databases at once.

# Usage:

	  //Build a config:
	  cfg := &sqldb.Config{
		Type:       sqldb.DBTypeSqlite,
		SQLitePath: "/path/to/sqlite.db",
	  }

	  //Use the config as a singleton.
	  sqldb.Use(cfg)
	  err := sqldb.Connect()
	  if err != nil {
		log.Fatalln(err)
		return
	  }

	  c := sqldb.Connection()
	  err := c.Exec("SELECT * FROM my_table")
	  if err != nil {
		log.Fatalln(err)
		return
	  }

# Deploying a Database

Deploying of a schema is done via DeployQueries and DeployFuncs, along with the
associated DeployQueryTranslators and DeployQueryErrorHandlers. DeployQueries are
just SQL query strings while DeployFuncs are used for more complicated deployment
scenarios, such as INSERTs that rely on some sort of non-SQL data (ex: data
calculated by golang code in your application).

DeployQueryTranslators translate DeployQueries from one database type to another
(ex: MariaDB to SQLite) since different databases support slightly different
SQL dialects. This allows you to write your CREATE TABLE or other deployment
related queries in one database type's format, but then modify the query
programatically to the format required for another database type. This is extremely
useful if your application supports multiple database types. Note that
DeployQueryTranslators do not apply to DeployFuncs since DeployFuncs are more than
just a SQL query.

DeployQueryErrorHandlers is a list of functions that are run when any DeployQuery
results in an error (as returned by [sql.Exec]). These funcs are used to evaluate,
and if appropriate, ignore the error.

DeployQueries and DeployFuncs should be safe to be rerun multiple times, particularly
without INSERTing duplicate data. Use IF NOT EXISTS or check if something exists
before INSERTing in DeployFuncs.

# Updating a Schema

Updating an existing database schema is done via UpdateQueries and UpdateFuncs, along
with the associated UpdateQueryTranslators and UpdateQueryErrorHandlers. This
functionality is similar to the deploy schema tooling.

UpdateQueryErrorHandlers are very useful for handling queries that when run multiple
times would result in an error, especially when the IF EXISTS syntax is not
available (see SQLite for ALTER TABLE...DROP COLUMN).

# SQLite Library

This package support two SQLite libraries, [github.com/mattn/go-sqlite3] and
[gitlab.com/cznic/sqlite]. The mattn library requires CGO which can be troublesome
for cross-compiling. The modernc library is pure golang, however, it is a translation,
not the original SQLite code, and therefore does not have the same level of
trustworthiness or extent of testing.

As of now, mattn is the default if no build tags are provided. This is simply due to
the longer history of this library being available and the fact that this uses the
SQLite C source code.

Use either library with build tags:

	go build -tags mattn ...
	go build -tags modernc ...
	go run -tags mattn ...
	go run -tags modernc ...

The mattn library sets some default PRAGMA values, as noted in the source code at
https://github.com/mattn/go-sqlite3/blob/ae2a61f847e10e6dd771ecd4e1c55e0421cdc7f9/sqlite3.go#L1086.
Some of these are just safe defaults, for example, busy_timeout. In order to treat
the mattn and modernc libraries more similarly, some of these mattn PRAGMAs are also
set when using the modernc library. This is done to make using both libraries act in
the same manner, make them more interchangable with the same result.

# Notes

This package uses [github.com/jmoiron/sqlx] instead of the go standard library
[database/sql] package because sqlx provides some additional tooling which makes using
a database a bit easier (i.e.: Get(), Select(), and StructScan()).

Could possible remove sqlx and required users of this package to call
[github.com/jmoiron/sqlx.NewDb] if a [sqlx.DB] is needed. Could also just do this
internally, via Config.Connection() and Config.Connectionx()
*/
package sqldb

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"

	//MySQL/MariaDB driver import is not an empty import b/c we use it the driver
	//library to generate the connection string.
	"github.com/go-sql-driver/mysql"

	//SQLite driver is imported in other sqldb-sqlite-*.go files based upon build
	//tag to handle mattn or modernc library being used.

	//MS SQL Server.
	_ "github.com/denisenkom/go-mssqldb"
)

// Config is the details used for establishing and using a database connection.
type Config struct {
	//Type represents the type of database to use.
	Type dbType

	//Connection information for a non-SQLite database.
	Host     string
	Port     uint
	Name     string
	User     string
	Password string

	//ConnectionOptions is a list of key-value pairs of options used when building
	//the connection string used to connect to a database. Each driver/database type
	//will handle these differently. Use AddConnectionOption() instead of having to
	//do Config.ConnectionOptions = map[string]string{"key", "value"}.
	ConnectionOptions map[string]string

	//SQLitePath is the path where the SQLite database file is located.
	SQLitePath string

	//SQLitePragmas is a list of PRAGMAs to apply when connecting to a SQLite
	//database. Typically this is used to set the journal mode or busy timeout.
	//PRAGMAs provided here are in SQLite query format with an equals sign
	//(ex.: PRAGMA busy_timeout=5000).
	//
	//Both the mattn and modernc packages allow setting of PRAGMAs in the database
	//filename path. See the below links. PRAGMA statements here will be appended to
	//the SQLitePath, after translating to the correct library's format, so that the
	//PRAGMAs are set properly for the database upon initially opening it.
	//
	//https://github.com/mattn/go-sqlite3#connection-string)
	//https://pkg.go.dev/modernc.org/sqlite#Driver.Open
	SQLitePragmas []string

	//MapperFunc is used to override the mapping of database column names to struct
	//field names or struct tags. Mapping of column names is used during queries
	//where sqlx's StructScan(), Get(), or Select() is used.
	//
	//By default, column names are not modified in any manner. This is in contrast to
	//the default for sqlx where column names are returned as all lower case which
	//requires your structs to use struct tags for each exported field. By not
	//modifying column names you will not need to use struct tags since column names
	//can exactly match exportable struct field names. This is just a small helper
	//done to reduce the amount of struct tags that are necessary.
	//
	//http://jmoiron.github.io/sqlx/#:~:text=You%20can%20use%20the%20db%20struct%20tag%20to%20specify%20which%20column%20name%20maps%20to%20each%20struct%20field%2C%20or%20set%20a%20new%20default%20mapping%20with%20db.MapperFunc().%20The%20default%20behavior%20is%20to%20use%20strings.Lower%20on%20the%20field%20name%20to%20match%20against%20the%20column%20names.
	MapperFunc func(string) string

	//DeployQueries is a list of SQL queries used to deploy a database schema. These
	//are typically used for CREATE TABLE or CREATE INDEX queries. These queries will
	//be run when DeploySchema() is called.
	//
	//Order matters! Order your queries so that foreign tables for relationships are
	//created before the relationships!
	//
	//Each query should be safe to be rerun multiple times!
	DeployQueries []string

	//DeployFuncs is a list of functions, each containing at least one SQL query,
	//that is used to deploy a database schema. Use these for more complicated schema
	//deployment or initialization steps, such as INSERTing initial data. DeployFuncs
	//should be used more sparingly than DeployQueries. These functions will be run
	//when DeploySchema() is called.
	//
	//These functions are executed after DeployQueries.
	//
	//Each function should be safe to be rerun multiple times!
	DeployFuncs []QueryFunc

	//DeployQueryTranslators is a list of functions that translate a DeployQuery from
	//one database dialect to another. This functionality is provided so that you do
	//not have to rewrite your deployment queries for each database type you want to
	//deploy for.
	//
	//A DeployQueryTranslator function takes a DeployQuery as an input and returns a
	//rewritten query.
	//
	//See predefined translator functions starting with TF.
	DeployQueryTranslators []Translator

	//DeployQueryErrorHandlers is a list of functions that are run when an error
	//results from running a DeployQuery and is used to determine if the error can be
	//ignored. Use this for ignoring expected errors from SQL queries, typically when
	//you rerun DeploySchema() and something already exists but the IS NOT EXISTS
	//term is unavailable.
	//
	//A DeployQueryErrorHandler function takes a DeployQuery and the error resulting
	//from [database/sql.Exec] as an input and returns true if the error should be ignored.
	DeployQueryErrorHandlers []ErrorHandler

	//UpdateQueries is a list of SQL queries used to update a database schema. These
	//are typically used ot add new columns, ALTER a column, or DROP a column. These
	//queries will be run when UpdateSchema() is called.
	//
	//Order matters! Order your queries if they depend on each other (ex.: renamed
	//tables or columns).
	//
	//Each query should be safe to be rerun multiple times!
	UpdateQueries []string

	//UpdateFuncs is a list of functions, each containing at least one SQL query,
	//that is used to update a database schema. Use these for more complicated schema
	//updates, such as reading values before updating. UpdateFuncs should be used
	//more sparingly than UpdateQueries. These functions will be run when
	//UpdateSchema() is called.
	//
	//These functions are executed after UpdateQueries.
	//
	//Each function should be safe to be rerun multiple times!
	UpdateFuncs []QueryFunc

	//UpdateQueryTranslators is a list of functions that translate an UpdateQuery
	//from one database dialect to another.
	//
	//An UpdateQueryTranslator function takes an UpdateQuery as an input and returns
	//a rewritten query.
	UpdateQueryTranslators []Translator

	//UpdateQueryErrorHandlers is a list of functions that are run when an error
	//results from running an UpdateQuery and is used to determine if the error can
	//be ignored.
	//An UpdateQueryErrorHandler function takes an UpdateQuery and the error resulting
	//from Exec as an input and returns true if the error should be ignored.
	UpdateQueryErrorHandlers []ErrorHandler

	//LoggingLevel enables logging at ERROR, INFO, or DEBUG levels.
	LoggingLevel logLevel

	//connection is the established connection to a database for performing queries.
	//This is the underlying sql connection pool. Access this via the Connection()
	//func to run queries against the database.
	connection *sqlx.DB

	//connectionString is the connection string used to establish the connection to
	//the database. This is set upon Connect() being called and is used for debugging.
	connectionString string
}

// QueryFunc is a function used to perform a deployment or Update task that is more
// complex than just a SQL query that could be provided in a DeployQuery or UpdateQuery.
type QueryFunc func(*sqlx.DB) error

// Supported databases.
type dbType string

const (
	DBTypeMySQL   = dbType("mysql")
	DBTypeMariaDB = dbType("mariadb")
	DBTypeSQLite  = dbType("sqlite")
	DBTypeMSSQL   = dbType("mssql")
)

var validDBTypes = []dbType{
	DBTypeMySQL,
	DBTypeMariaDB,
	DBTypeSQLite,
	DBTypeMSSQL,
}

// DBType returns a dbType. This is used when parsing a user-provided database type
// (such as from ann external configuration file) to convert to a dbType defined in
// this package.
func DBType(s string) dbType {
	return dbType(s)
}

var (
	//ErrConnected is returned when a trying to establish a connection to an already
	//connected-to database.
	ErrConnected = errors.New("sqldb: connection already established")

	//ErrSQLitePathNotProvided is returned SQLitePath is empty.
	ErrSQLitePathNotProvided = errors.New("sqldb: SQLite path not provided")

	//ErrHostNotProvided is returned when no Host IP or FQDN was provided.
	ErrHostNotProvided = errors.New("sqldb: database server host not provided")

	//ErrInvalidPort is returned when no Port was provided or the port was invalid.
	ErrInvalidPort = errors.New("sqldb: database server port invalid")

	//ErrNameNotProvided is returned when no database Name was provided.
	ErrNameNotProvided = errors.New("sqldb: database name not provided")

	//ErrUserNotProvided is returned when no database User was provided.
	ErrUserNotProvided = errors.New("sqldb: database user not provided")

	//ErrPasswordNotProvided is returned when no database Password was provided.
	//Blank passwords are not supported since it is terrible for security.
	ErrPasswordNotProvided = errors.New("sqldb: password for database user not provided")
)

var (
	//ErrNoColumnsGiven is returned when trying to build a column list for a query
	//but no columns were provided.
	ErrNoColumnsGiven = errors.New("sqldb: no columns provided")

	//ErrExtraCommaInColumnString is returned when building a column string for a
	//query but an extra comma exists which would cause the query to run incorrectly.
	//Extra commas are usually due to an empty column name being provided or a comma
	//being added to the column name by mistake.
	ErrExtraCommaInColumnString = errors.New("sqldb: extra comma in column name")
)

// cfg is the package-level stored configuration for a database. This is used when
// you are using this package in a singleton manner. This is populated when Use() is
// called.
var cfg *Config

// New returns a Config instance with some defaults set. You would typically call
// Use() and/or Connect() after New().
func New() *Config {
	c := new(Config)

	c.SQLitePragmas = sqliteDefaultPragmas
	c.MapperFunc = defaultMapperFunc
	c.LoggingLevel = LogLevelDefault
	c.ConnectionOptions = make(map[string]string)

	return c
}

// Use stores a config in the package-level variable when you are using this package
// in a singleton manner.
//
// This does not check if Use() has previously been called; Use() should only ever be
// called once unless you are certain you closed an existing database connection.
func Use(c *Config) {
	cfg = c
}

// Connect connects to the database. This establishes the database connection, and
// saves the connection pool for use in running queries. For SQLite, this also runs
// any PRAGMA commands when establishing the connection.
func (c *Config) Connect() (err error) {
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

	//Get the connection string used to connect to the database.
	connString := c.buildConnectionString(false)
	c.connectionString = connString

	//Get the correct driver based on the database type.
	//
	//If using SQLite, the correct driver is chosen based on build tags.
	driver := getDriver(c.Type)

	//Connect to the database.
	//
	//For SQLite, check if the database file exists. This func will not create the
	//database file. The database file needs to be created first with DeploySchema().
	//If the database is in-memory, we can ignore this error though, since, the
	//database will never exist yet an is in fact created when Open() and Ping() are
	//called below.
	if c.IsSQLite() && c.SQLitePath != InMemoryFilePathRacy && c.SQLitePath != InMemoryFilePathRaceSafe {
		_, err = os.Stat(c.SQLitePath)
		if os.IsNotExist(err) {
			return err
		}
	}

	//Connect to the database.
	//
	//Note no "defer conn.Close()" since we want to keep the connection alive for
	//future use in running queries. It is the job of whatever func called Connect()
	//to call Close().
	conn, err := sqlx.Open(driver, connString)
	if err != nil {
		return
	}

	err = conn.Ping()
	if err != nil {
		return
	}

	//Set the mapper func for mapping column names to struct fields.
	if c.MapperFunc != nil {
		conn.MapperFunc(c.MapperFunc)
	}

	//Save the connection for running future queries.
	c.connection = conn

	//Diagnostic logging, useful for logging out which database you are connected to.
	switch c.Type {
	case DBTypeMySQL, DBTypeMariaDB, DBTypeMSSQL:
		c.infoLn("sqldb.Connect", "Connecting to database "+c.Name+" on "+c.Host+" with user "+c.User+".")
	case DBTypeSQLite:
		c.infoLn("sqldb.Connect", "Connecting to database: "+c.SQLitePath+".")
		c.debugLn("sqldb.Connect", "SQLite Library: "+GetSQLiteLibrary()+".")
		c.debugLn("sqldb.Connect", "SQLite PRAGMAs: "+pragmsQueriesToString(c.SQLitePragmas)+".")
	default:
		//This can never occur because we called validate() above to verify that a
		//valid database type was provided.
	}

	return
}

// Connect connects to the database using the config stored at the package level. Use
// this after calling Use().
func Connect() (err error) {
	return cfg.Connect()
}

// defaultMapperFunc is the default function used for handling column name formatting
// when retrieving data from the database and matching up to struct field names. No
// reformatting is done; the column names are returned exactly as they are noted in
// the database schema. This is unlike [sqlx] that lowercases all column names and
// thus requires struct tags to match up against exported struct fields.
//
// See: https://jmoiron.github.io/sqlx/#mapping
func defaultMapperFunc(s string) string {
	return s
}

// validate handles validation of a provided config before establishing a connection
// to the database. This is called in Connect().
func (c *Config) validate() (err error) {
	//Sanitize.
	c.SQLitePath = strings.TrimSpace(c.SQLitePath)
	c.Host = strings.TrimSpace(c.Host)
	c.Name = strings.TrimSpace(c.Name)
	c.User = strings.TrimSpace(c.User)

	//Check config fields based on the database type since each type of database has
	//different requirements. This also checks that a valid (i.e.: supported by this
	//package) database type was provided.
	switch c.Type {
	case DBTypeSQLite:
		if c.SQLitePath == "" {
			return ErrSQLitePathNotProvided
		}

		//We don't check PRAGMAs since they are just strings. We will return any
		//errors when the database is connected to via Open().

	case DBTypeMySQL, DBTypeMariaDB, DBTypeMSSQL:
		if c.Host == "" {
			return ErrHostNotProvided
		}
		if c.Port == 0 || c.Port > 65535 {
			return ErrInvalidPort
		}
		if c.Name == "" {
			return ErrNameNotProvided
		}
		if c.User == "" {
			return ErrUserNotProvided
		}
		if c.Password == "" {
			return ErrPasswordNotProvided
		}

	default:
		return fmt.Errorf("sqldb: invalid database type, should be one of '%s', got '%s'", validDBTypes, c.Type)
	}

	//Use default logging level if an invalid logging level was provided. Not
	//error out here if an invalid value was provided since logging is less important.
	if c.LoggingLevel < LogLevelNone || c.LoggingLevel > LogLevelDebug {
		c.LoggingLevel = LogLevelDefault
		c.errorLn("sqldb.validate", "invalid LoggingLevel, defaulting to LogLevelDefault")
	}

	return
}

// buildConnectionString creates the string used to connect to a database. The
// returned values is built for a specific database type since each type has
// different parameters needed for the connection.
//
// Note that when building the connection string for MySQL or MariaDB, we have to
// omit the database name if we are deploying the database, since, obviously, the
// database does not exist yet! The database name is only appended to the connection
// string when connecting to an already existing database.
//
// When building a connection string for SQLite, we attempt to translate the listed
// SQLitePragmas to the correct format based on the SQLite library in use and
// append these pragmas to the filepath. This is done since you can only reliably set
// PRAGMAs when first connecting to the SQLite database, not anytime afterward, due to
// connection pooling and PRAGMAs being set per-connection.
func (c *Config) buildConnectionString(deployingDB bool) (connString string) {
	switch c.Type {
	case DBTypeMariaDB, DBTypeMySQL:
		//For MySQL or MariaDB, use connection string tooling and formatter instead
		//of building the connection string manually.
		dbConnectionConfig := mysql.NewConfig()
		dbConnectionConfig.User = c.User
		dbConnectionConfig.Passwd = c.Password
		dbConnectionConfig.Net = "tcp"
		dbConnectionConfig.Addr = net.JoinHostPort(c.Host, strconv.Itoa(int(c.Port)))

		if !deployingDB {
			dbConnectionConfig.DBName = c.Name
		}

		connString = dbConnectionConfig.FormatDSN()

	case DBTypeSQLite:
		connString = c.SQLitePath

		//For SQLite, the connection string is simply a path to a file. However, we
		//need to append pragmas as needed.
		if len(c.SQLitePragmas) != 0 {
			pragmasToAdd := pragmsQueriesToString(c.SQLitePragmas)

			if strings.Contains(connString, "?") {
				//handle InMemoryFilePathRaceSafe
				connString += "&" + pragmasToAdd
			} else {
				connString += "&" + pragmasToAdd
			}

			c.debugLn("sqldb.buildConnectionString", "PRAGMAs provided: ", c.SQLitePragmas)
			c.debugLn("sqldb.buildConnectionString", "PRAGMA String:    ", pragmasToAdd)
			c.debugLn("sqldb.buildConnectionString", "Path With PRAGMAS:", connString)
		}

	case DBTypeMSSQL:
		u := &url.URL{
			Scheme: "sqlserver",
			User:   url.UserPassword(c.User, c.Password),
			Host:   net.JoinHostPort(c.Host, strconv.FormatUint(uint64(c.Port), 10)),
		}

		q := url.Values{}
		q.Add("database", c.Name)

		//Handle other connection options.
		if len(c.ConnectionOptions) > 0 {
			for key, value := range c.ConnectionOptions {
				q.Add(key, value)
			}
		}

		u.RawQuery = q.Encode()
		connString = u.String()

	default:
		//we should never hit this since we already validated the database type in in
		//validate().
	}

	return
}

// getDriver returns the Go sql driver used for the chosen database type. This is
// used in Connect() to get the name of the driver as needed by [database/sql.Open].
func getDriver(t dbType) (driver string) {
	switch t {
	case DBTypeSQLite:
		//See sqlite- subfiles based on library used. Correct driver is chosen based
		//on build tags.
		driver = sqliteDriverName

	case DBTypeMySQL, DBTypeMariaDB:
		driver = "mysql"

	case DBTypeMSSQL:
		driver = "mssql" //maybe sqlserver works too?

	default:
		//This can never occur because this func is only called in Connect() after
		//validate() has already been called and verified a valid database type was
		//provided.
	}

	return
}

// Close handles closing the underlying database connection stored in the config.
func (c *Config) Close() (err error) {
	return c.connection.Close()
}

// Close handles closing the underlying database connection stored in the package
// level config.
func Close() (err error) {
	return cfg.Close()
}

// Connected returns if the config represents an established connection to a database.
func (c *Config) Connected() bool {
	//A connection has never been established.
	if c.connection == nil {
		return false
	}

	//A connection was been established but was closed. c.connection won't be nil in
	//this case, it still stores the previous connection's info for some reason. We
	//don't set it to nil in Close() since that isn't how the sql package handles
	//closing.
	err := c.connection.Ping()
	//lint:ignore S1008 - I like the "if err == nil {return...}" format better than "return err == nil".
	if err != nil {
		return false
	}

	//A connection was been established and is open, ping succeeded.
	return true
}

// Connected returns if the config represents an established connection to a database.
func Connected() bool {
	return cfg.Connected()
}

// Connection returns the underlying database connection stored in a config for use
// in running queries.
func (c *Config) Connection() *sqlx.DB {
	return c.connection
}

// Connection returns the underlying database connection stored in the package level
// config for use in running queries.
func Connection() *sqlx.DB {
	return cfg.Connection()
}

// AddConnectionOption adds a key-value pair to a config's ConnnectionOptions field.
// Using this func is just easier then calling map[string]string{"key", "value"}.
// This does not check if the key already exist, it will simply add a duplicate
// key-value pair.
func (c *Config) AddConnectionOption(key, value string) {
	//Initialize map if needed.
	if c.ConnectionOptions == nil {
		c.ConnectionOptions = make(map[string]string)
	}

	c.ConnectionOptions[key] = value
}

// AddConnectionOption adds a key-value pair to a config's ConnnectionOptions field.
// Using this func is just easier then calling map[string]string{"key", "value"}.
// This does not check if the key already exist, it will simply add a duplicate
// key-value pair.
func AddConnectionOption(key, value string) {
	cfg.AddConnectionOption(key, value)
}

// Type return the dbType from a Config.
//
// This func is geared toward usage in a switch statement, specifically for when you
// store your Config in this package's global variable (singleton style). This removes
// the need to have a bunch of if/elseif blocks calling sqldb.IsMariaDB(), sqldb.IsSQLite(),
// and so forth. If you store your Config elsewhere, outside of this package, you can
// just build a switch statement from the Config's Type field.
func Type() dbType {
	return cfg.Type
}
