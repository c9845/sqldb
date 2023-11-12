/*
Package sqldb provides tooling to make connecting to, deploying, updating, and using
a SQL database easier. This provides some wrapping around the [database/sql] package.

The initial reasoning behind this package was to encapsulate commonly used sql
database connection, schema deploying and updating, and other boilerplate tasks so
that I did not have to update near-identical code in a large amount of projects. For
example, if I found a bug in the sql related code, I did not want to have to update
the same code in 30 different projects.

NOTE! Microsoft SQL Server is not fully tested!

# Global or Local DB Connection

You can use this package in two methods: as a singleton a the database configuration
and connection stored within this package in a globally accessible variable, or store
the configuration and connection somewhere else in your application. Storing the data
yourself allows for connecting to multiple databases at once.

# Deploying a Database

Deploying of a schema is done via DeployQueries and DeployFuncs, along with the
associated DeployQueryTranslators and DeployQueryErrorHandlers. DeployQueries are
just SQL query strings while DeployFuncs are used for more complicated deployment
scenarios, such as INSERTs that rely on some sort of non-SQL data (ex: data
calculated by golang code in your application). Use this for CREATE TABLE, CREATE
INDEX, or INSERTing initial data.

DeployQueryTranslators translate DeployQueries from one database type to another
(ex: MariaDB to SQLite) since different databases support slightly different
SQL dialects. This allows you to write your CREATE TABLE or other deployment
related queries in one database type's format, but then modify the query
programatically to the format required for another database type. This is extremely
useful if your application supports using multiple database tables. Note that
DeployQueryTranslators do not apply to DeployFuncs since DeployFuncs are more than
just a SQL query.

DeployQueryErrorHandlers is a list of functions that are run when any DeployQuery
results in an error (as returned by [sql.Exec]). These funcs are used to evaluate,
and if appropriate, ignore the error. Use this for handling situations in the same
manner as CREATE IF NOT EXISTS.

DeployQueries and DeployFuncs should be safe to be rerun multiple times, particularly
without INSERTing duplicate data. Use IF NOT EXISTS or check if something exists
before INSERTing in DeployFuncs.

# Updating a Schema

Updating an existing database schema is done via UpdateQueries and UpdateFuncs, along
with the associated UpdateQueryTranslators and UpdateQueryErrorHandlers. These all
function the same as the related deploy schema tooling.

All updated related queries are run inside of the same transaction so if an error
occurs the database does not end up in an unknown state. Either the entire schema
update succeeds and is applied or none of the changes are applied.

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
*/
package sqldb

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
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

	//SQLitePragmas is a list of PRAGMA queries to run when connecting to a SQLite
	//database. Typically this is used to set the journal mode or busy timeout.
	//PRAGMAs provided here are in SQLite format with an equals sign
	//(ex.: PRAGMA busy_timeout=5000).
	//
	//Both the mattn and modernc packages allow setting of PRAGMAs in the database
	//filename path. See the below links. PRAGMA statements here will be appended to
	//the SQLitePath, after translating to the correct library's format, so that the
	//PRAGMAs are set properly for the database upon initially opening it.
	//
	//It is important to note that while PRAGMAs are given in SQLite's PRAGMA query
	//format, the PRAGMAs are actually applied to the connection string, not via
	//queries. Setting PRAGMAs via queries cannot be trusted since PRAGMA queries are
	//applied to the specific connection that ran it, however, the [database/sql]
	//package maintains a list of connections, not a single one. Therefore, PRAGMAs
	//will not be applied to all connections and results will be inconsistent; it
	//cannot be guaranteed that you will get the same connection that applied a
	//PRAGMA query.
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
	DeployFuncs []DeployFunc

	//DeployQueryTranslators is a list of functions that translate a DeployQuery from
	//one database dialect to another. This functionality is provided so that you do
	//not have to rewrite your deployment queries for each database type you want to
	//deploy for.
	//
	//A DeployQueryTranslator function takes a DeployQuery as an input and returns a
	//rewritten query.
	//
	//See predefined translator functions starting with TF.
	DeployQueryTranslators []func(string) string

	//DeployQueryErrorHandlers is a list of functions that are run when an error
	//results from running a DeployQuery and is used to determine if the error can be
	//ignored. Use this for ignoring expected errors from SQL queries, typically when
	//you rerun DeploySchema() and something already exists but the IS NOT EXISTS
	//term is unavailable.
	//
	//A DeployQueryErrorHandler function takes a DeployQuery and the error resulting
	//from Exec as an input and returns true if the error should be ignored.
	DeployQueryErrorHandlers []func(string, error) bool

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
	UpdateFuncs []UpdateFunc

	//UpdateQueryTranslators is a list of functions that translate an UpdateQuery
	//from one database dialect to another.
	//
	//An UpdateQueryTranslator function takes an UpdateQuery as an input and returns
	//a rewritten query.
	UpdateQueryTranslators []func(string) string

	//UpdateQueryErrorHandlers is a list of functions that are run when an error
	//results from running an UpdateQuery and is used to determine if the error can
	//be ignored.
	//An UpdateQueryErrorHandler function takes an UpdateQuery and the error resulting
	//from Exec as an input and returns true if the error should be ignored.
	UpdateQueryErrorHandler []func(string, error) bool

	//LoggingLevel enables logging at ERROR, INFO, or DEBUG levels.
	LoggingLevel logLevel

	//connection is the established connection to a database for performing queries.
	//This is the underlying sql connection pool. Access this via the Connection()
	//func to run queries against the database.
	connection *sqlx.DB
}

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

// valid checks if a provided dbType is one of our supported databases. This is used
// when validating.
func (t dbType) valid() error {
	contains := slices.Contains(validDBTypes, t)
	if contains {
		return nil
	}

	return fmt.Errorf("invalid db type, should be one of '%s', got '%s'", validDBTypes, t)
}

// Logging levels, each higher level is inclusive of lower levels; i.e.: if you choose
// to use LogLevelDebug, all Error and Info logging will also be output.
type logLevel int

const (
	LogLevelNone  logLevel = iota //no logging.
	LogLevelError                 //general errors, most typical use.
	LogLevelInfo                  //some info on db connections, deployment, updates.
	LogLevelDebug                 //primarily used during development.
)

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

var (
	//ErrInvalidLoggingLevel is returned when an invalid logging level is provided.
	ErrInvalidLoggingLevel = errors.New("sqldb: invalid logging level")
)

// config is the configuration to connect to and use a SQL database. This stores your
// configuration when you are using this package as a singleton.
//
// This is used when you call one of the NewDefaultConfig() functions or when Save()
// is called.
var config Config

// NewConfig returns a base configuration that will need to be modified for use to
// connect to and interact with a database. Typically you would use New...Config()
// instead.
func NewConfig(t dbType) (cfg *Config, err error) {
	err = t.valid()
	if err != nil {
		return
	}

	cfg = &Config{
		Type:              t,
		MapperFunc:        DefaultMapperFunc,
		LoggingLevel:      LogLevelInfo,
		ConnectionOptions: make(map[string]string),
	}
	return
}

// DefaultConfig initializes the globally accessible package level config with some
// defaults set. Typically you would use Default...Config() instead.
func DefaultConfig(t dbType) (err error) {
	cfg, err := NewConfig(t)
	config = *cfg
	return
}

// Save saves a configuration to the package level config. Use this in conjunction with
// New...Config(), or just sqldb.Config{}, when you want to heavily customize the
// config. This is not a method on Config so that any modifications done to the original
// config after Save() is called aren't propagated to the package level config without
// calling Save() again.
func Save(cfg Config) {
	config = cfg
}

// GetDefaultConfig returns the package level saved config.
func GetDefaultConfig() *Config {
	return &config
}

// validate handles validation of a provided config. This is called in Connect().
func (cfg *Config) validate() (err error) {
	//Sanitize.
	cfg.SQLitePath = strings.TrimSpace(cfg.SQLitePath)
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.User = strings.TrimSpace(cfg.User)

	//Make sure db type is valid. A user can modify this to a string of any value via
	//"cfg.Type = 'asdf'" even though Type has a specific dbType type. This catches
	//this slight possibility.
	err = cfg.Type.valid()
	if err != nil {
		return
	}

	//Check config based on db type since each type of db has different requirements.
	switch cfg.Type {
	case DBTypeSQLite:
		if cfg.SQLitePath == "" {
			return ErrSQLitePathNotProvided
		}

		//We don't check PRAGMAs since they are just strings. We will return any
		//errors when the database is connected to via Open().

	case DBTypeMySQL, DBTypeMariaDB, DBTypeMSSQL:
		if cfg.Host == "" {
			return ErrHostNotProvided
		}
		if cfg.Port == 0 || cfg.Port > 65535 {
			return ErrInvalidPort
		}
		if cfg.Name == "" {
			return ErrNameNotProvided
		}
		if cfg.User == "" {
			return ErrUserNotProvided
		}
		if cfg.Password == "" {
			return ErrPasswordNotProvided
		}
	}

	//Make sure logging level is a valid type if something was provided.
	if cfg.LoggingLevel < LogLevelNone || cfg.LoggingLevel > LogLevelDebug {
		cfg.LoggingLevel = LogLevelError
		cfg.debugPrintln("sqldb.validate", "invalid LoggingLevel, defaulting to LogLevelError")
	}

	return
}

// buildConnectionString creates the string used to connect to a database. The
// returned values is built for a specific database type since each type has different
// parameters needed for the connection.
//
// Note that when building the connection string for MySQL or MariaDB, we have to omit
// the databasename if we are deploying the database, since, obviously, the database
// does not exist yet. The database name is only appended to the connection string when
// the database exists.
//
// When building a connection string for SQLite, we attempt to translate and listed
// SQLitePragmas to the correct format based on the SQLite library in use and appending
// these pragmas to the filepath. This is done since you can only reliably set pragmas
// when first connecting to the SQLite database, not anytime afterward, due to connection
// pooling and pragmas being set per-connection.
func (cfg *Config) buildConnectionString(deployingDB bool) (connString string) {
	switch cfg.Type {
	case DBTypeMariaDB, DBTypeMySQL:
		//For MySQL or MariaDB, use connection string tooling and formatter instead
		//of building the connection string manually.
		dbConnectionConfig := mysql.NewConfig()
		dbConnectionConfig.User = cfg.User
		dbConnectionConfig.Passwd = cfg.Password
		dbConnectionConfig.Net = "tcp"
		dbConnectionConfig.Addr = net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port)))

		if !deployingDB {
			dbConnectionConfig.DBName = cfg.Name
		}

		connString = dbConnectionConfig.FormatDSN()

	case DBTypeSQLite:
		connString = cfg.SQLitePath

		//For SQLite, the connection string is simply a path to a file. However, we
		//need to append pragmas as needed.
		if len(cfg.SQLitePragmas) != 0 {
			pragmasToAdd := cfg.SQLitePragmasAsString()
			connString += pragmasToAdd

			cfg.debugPrintln("sqldb.buildConnectionString", "PRAGMA String:", pragmasToAdd)
			cfg.debugPrintln("sqldb.buildConnectionString", "Path With PRAGMAS:", connString)
		}

	case DBTypeMSSQL:
		u := &url.URL{
			Scheme: "sqlserver",
			User:   url.UserPassword(cfg.User, cfg.Password),
			Host:   net.JoinHostPort(cfg.Host, strconv.FormatUint(uint64(cfg.Port), 10)),
		}

		q := url.Values{}
		q.Add("database", cfg.Name)

		//Handle other connection options.
		if len(cfg.ConnectionOptions) > 0 {
			for key, value := range cfg.ConnectionOptions {
				q.Add(key, value)
			}
		}

		u.RawQuery = q.Encode()
		connString = u.String()

	default:
		//we should never hit this since we already validated the db type in in
		//validate().
	}

	return
}

// getDriver returns the Go sql driver used for the chosen database type.
func getDriver(t dbType) (driver string, err error) {
	if err := t.valid(); err != nil {
		return "", err
	}

	switch t {
	case DBTypeMySQL, DBTypeMariaDB:
		driver = "mysql"
	case DBTypeSQLite:
		//See sqlite subfiles based on library used. Correct driver is chosen based
		//on build tags.
		driver = sqliteDriverName

	case DBTypeMSSQL:
		driver = "mssql" //maybe sqlserver works too?
	}

	return
}

// Connect connects to the database. This establishes the database connection, and
// saves the connection pool for use in running queries. For SQLite this also runs any
// PRAGMA commands.
func (cfg *Config) Connect() (err error) {
	//Make sure the connection isn't already established to prevent overwriting it.
	//This forces users to call Close() first to prevent any incorrect db usage.
	if cfg.Connected() {
		return ErrConnected
	}

	//Make sure the config is valid.
	err = cfg.validate()
	if err != nil {
		return
	}

	//Get the connection string used to connect to the database.
	connString := cfg.buildConnectionString(false)

	//Get the correct driver based on the database type. If using SQLite, the correct
	//driver is chosen based on build tags. Error should never occur this since we
	//already validated the config in validate().
	//
	//We can ignore the error here since an invalid Type would have already been caught
	//in .validate().
	driver, _ := getDriver(cfg.Type)

	//Connect to the database.
	//
	//For SQLite, check if the database file exists. This func will not create the
	//database file. The database file needs to be created first with Deploy(). If
	//the database is in-memory, we can ignore this error.
	if cfg.IsSQLite() && cfg.SQLitePath != InMemoryFilePathRacy && cfg.SQLitePath != InMemoryFilePathRaceSafe {
		_, err = os.Stat(cfg.SQLitePath)
		if os.IsNotExist(err) {
			return err
		}
	}

	//This doesn't really establish a connection to the database, it just "builds" the
	//connection. The connection is established with Ping() below.
	//
	//Note no `defer conn.Close()` since we want to keep the db connection alive for
	//future use in running queries. It is the job of whatever func called Connect to
	//call Close (or defer cfg.Close()).
	conn, err := sqlx.Open(driver, connString)
	if err != nil {
		return
	}

	//Test the connection to the database to make sure it works. This opens the
	//connection for future use.
	err = conn.Ping()
	if err != nil {
		return
	}

	//Set the mapper func for mapping column names to struct fields.
	if cfg.MapperFunc != nil {
		conn.MapperFunc(cfg.MapperFunc)
	}

	//Save the connection for running future queries.
	cfg.connection = conn

	//Diagnostic logging.
	switch cfg.Type {
	case DBTypeMySQL, DBTypeMariaDB, DBTypeMSSQL:
		cfg.infoPrintln("sqldb.Connect", "Connecting to database "+cfg.Name+" on "+cfg.Host+" with user "+cfg.User+".")
	case DBTypeSQLite:
		cfg.infoPrintln("sqldb.Connect", "Connecting to database: "+cfg.SQLitePath+".")
		cfg.infoPrintln("sqldb.Connect", "SQLite Library: "+GetSQLiteLibrary()+".")
		cfg.infoPrintln("sqldb.Connect", "SQLite PRAGMAs: "+cfg.SQLitePragmasAsString()+".")
	default:
		//this can never happen since we hardcode the supported sqlite libraries.
	}

	return
}

// Connect handles the connection to the database using the default package level
// config.
func Connect() (err error) {
	return config.Connect()
}

// Close closes the connection to the database.
func (cfg *Config) Close() (err error) {
	return cfg.connection.Close()
}

// Close closes the connection using the default package level config.
func Close() (err error) {
	return config.Close()
}

// Connected returns if the config represents an established connection to the database.
func (cfg *Config) Connected() bool {
	//A connection has never been established.
	if cfg.connection == nil {
		return false
	}

	//A connection was been established but was closed. c.connection won't be nil in
	//this case, it still stores the previous connection's info for some reason. We
	//don't set it to nil in Close() since that isn't how the sql package handles
	//closing.
	err := cfg.connection.Ping()
	//lint:ignore S1008 - I like the "if {return...}" format better than "return err == nil".
	if err != nil {
		return false
	}

	//A connection was been established and is open, ping succeeded.
	return true
}

// Connected returns if the config represents an established connection to the database.
func Connected() bool {
	return config.Connected()
}

// Connection returns the database connection stored in a config for use in running
// queries.
func (cfg *Config) Connection() *sqlx.DB {
	return cfg.connection
}

// Connection returns the database connection for the package level config.
func Connection() *sqlx.DB {
	return config.Connection()
}

// IsMySQLOrMariaDB returns if the database is a MySQL or MariaDB. This is useful
// since MariaDB is a fork of MySQL and most things are compatible; this way you
// don't need to check IsMySQL() and IsMariaDB().
func (cfg *Config) IsMySQLOrMariaDB() bool {
	return cfg.Type == DBTypeMySQL || cfg.Type == DBTypeMariaDB
}

// IsMySQLOrMariaDB returns if the database is a MySQL or MariaDB for the package
// level config.
func IsMySQLOrMariaDB() bool {
	return config.IsMySQLOrMariaDB()
}

// DefaultMapperFunc is the default MapperFunc set on configs. It returns the column
// names unmodified.
func DefaultMapperFunc(s string) string {
	return s
}

// MapperFunc sets the mapper func for the package level config.
func MapperFunc(m func(string) string) {
	config.MapperFunc = m
}

// AddConnectionOption adds a key-value pair to a config's ConnnectionOptions field.
// Using this func is just easier then calling map[string]string{"key", "value"}. This
// does not check if the key already exist, it will simply add a duplicate key-value
// pair.
func (cfg *Config) AddConnectionOption(key, value string) {
	//Initialize map if needed.
	if cfg.ConnectionOptions == nil {
		cfg.ConnectionOptions = make(map[string]string)
	}

	cfg.ConnectionOptions[key] = value
}

// AddConnectionOption adds a key-value pair to the ConnectionOptions field for the
// package level config.
func AddConnectionOption(key, value string) {
	config.AddConnectionOption(key, value)
}

// UseDefaultTranslateFuncs populates TranslateCreateTableFuncs and TranslateUpdateFuncs
// with the default translation funcs.
func (cfg *Config) UseDefaultTranslateFuncs() {
	if cfg.IsSQLite() {
		cfg.TranslateCreateTableFuncs = []func(string) string{
			TFMySQLToSQLiteReformatID,
			TFMySQLToSQLiteRemovePrimaryKeyDefinition,
			TFMySQLToSQLiteReformatDefaultTimestamp,
			TFMySQLToSQLiteReformatDatetime,
			TFMySQLToSQLiteBLOB,
		}
		cfg.TranslateUpdateFuncs = []func(string) string{
			TFMySQLToSQLiteBLOB,
		}
	}
}

// UseDefaultTranslateFuncs populates TranslateCreateTableFuncs and TranslateUpdateFuncs
// with the default translation funcs for the package level config.
func UseDefaultTranslateFuncs() {
	config.UseDefaultTranslateFuncs()
}
