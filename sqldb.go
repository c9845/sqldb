/*
Package sqldb provides some tooling to make connecting to, deploying, updating, and
using a SQL database easier. This provides some wrapping around the sql package. This
package was written to reduce the amount of boilerplate code to connect, deploy, or
update a database.

NOTE! Microsoft SQL Server support is very untested!

# Global or Local DB Connection

You can use this package in two methods: as a singleton with the database configuration
and connection stored in a package-level globally-accessible variable, or store the
configuration and connection somewhere else in your application. Storing the data
yourself allows for connecting to multiple databases at once.

# Deploying a Database

Deploying of a schema is done via queries stored in your configuration's DeployQueries
and DeployFuncs fields. These queries and funcs are run when DeploySchema() is called.
DeploySchema() will create the database, if needed, before deploying tables, creating
indexes, inserting initial data, etc. Each DeployQuery is run before the DeployFuncs.

DeployQueries are typically used for creating tables or indexes. DeployFuncs are
typically used for more complex operations such as inserting initial data only if no
data already exists or handling a complicated schema change based on existing values.
When writing DeployQueries or DeployFuncs you should ensure the result of each is
indempotent; rerunning DeployQueries or DeployFuncs over and over should be successful
without returning errors about "already exists" or creating or inserting duplicate
records.

An advanced tool of deployment is TranslateDeployCreateTableFuncs. Funcs listed here
are used to translate a CREATE TABLE query from one database format to another (i.e.:
MySQL to SQLite). This allows you to write your CREATE TABLE queries in one database
format but allow running your application with a database deployed in multiple formats;
it just makes writing CREATE TABLE queries a bit easier. This translation is necessary
since different databases have different CREATE TABLE query formats or column types (
for example, SQLite doesn't really have VARCHAR). Each DeployQuery is run through each
TranslateDeployCreateTableFunc with translation of the query performed as needed.

# Updating a Schema

Updating a database schema happens in a similar manner to deploying, a list of queries
in UpdateQueries and UpdateFuncs is run against the database. All queries are run
encapsulated in a single transaction so that either the updated is successful, or none
of updates are applied. This is done to eliminate the possibility of a partial update.

Each UpdateQuery is run through a list of error analyzer functions, UpdateIgnoreErrorFuncs,
to determine if an error can be ignored. This is typically used to ignore errors for
when you are adding a column that already exists, removing a column that is already
removed, etc. These funcs just help ignore errors that aren't really errors.

# SQLite Library

This package support two SQLite libraries, mattn/go-sqlite3 and modernc/sqlite. The mattn
library encapsulates the SQLite C source code and requires CGO for compilation which
can be troublesome for cross-compiling. The modernc library is an automatic translation
of the C source code to golang, however, it isn't the "source" SQLite C code and
therefore doesn't have the same level of trustworthiness or extent of testing, however,
it can be cross-compiled much more easily.

As of now, mattn is the default if no build tags are provided. This is simply due to
the longer history of this library being available and the fact that this uses the
SQLite C source code.

Use either library with build tags:

	go build -tags mattn ...
	go build -tags modernc ...
	go run -tags mattn ...
	go run -tags modernc ...

The mattn/go-sqlite3 library sets some default PRAGMA values, as noted in the source code
at https://github.com/mattn/go-sqlite3/blob/ae2a61f847e10e6dd771ecd4e1c55e0421cdc7f9/sqlite3.go#L1086.
Some of these are just safe defaults, for example, busy_timeout. In order to treat the
mattn/go-sqlite3 and modernc/sqlite more similarly, some of these mattn/go-sqlite3 default
PRAGMAs are set when using the modernc/sqlite library. This is simply done to make
using either SQLite library more comparibly interchangable.

# Notes

This package uses "sqlx" instead of the go standard library "sql" package because "sqlx"
provides some additional tooling which makes using the database a bit easier (i.e.: Get(),
Select(), and StructScan() that can thus be in queries).

You should design your queries (DeployQueries, DeployFuncs, UpdateQueries, UpdateFuncs)
so that deploying or updating the schema is safe to rerun multiple times. You do not
want issues to occur if a user interacting with yourapp somehow tries to deploy the
database over and over or update it after it has already been updated. For example,
use "IF NOT EXISTS" with creating tables or indexes.
*/
package sqldb

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	//MySQL and MariaDB driver import is not an empty import b/c we use it the driver
	//library to generate the connection string.
	"github.com/go-sql-driver/mysql"

	//SQLite driver is imported in other sqldb-sqlite-*.go files due to different
	//libraries & build tags.

	//MS SQL Server.
	_ "github.com/denisenkom/go-mssqldb"

	"github.com/jmoiron/sqlx"
	"golang.org/x/exp/slices"
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
	//database. Typically this is used to set the journal mode or busy timeout. PRAGMAs
	//provided here are in SQLite format with an equals sign (ex.: PRAGMA busy_timeout=5000).
	//
	//Both the mattn/go-sqlite3 and modernc/sqlite packages allow setting of PRAGMAs
	//in the database filename path. See the below links. PRAGMA statements here will
	//be appended (after translating to the correct package's format) to the SQLitePath
	//so that the PRAGMAs are set properly for the database upon initially connecting.
	//
	//Setting PRAGMAs via a query cannot be trusted; the sql package provides a
	//connection pool and PRAGMAs are set per connection and you cannot be guaranteed
	//to get the same connection to resuse again.
	//
	//https://github.com/mattn/go-sqlite3#connection-string)
	//https://pkg.go.dev/modernc.org/sqlite#Driver.Open
	SQLitePragmas []string

	//MapperFunc is used to override the mapping of database column names to struct
	//field names or struct tags. Mapping of column names is used during queries where
	//sqlx's StructScan(), Get(), or Select() is used.
	//
	//By default, column names are not modified in any manner. This is in contrast to
	//the default for sqlx where column names are returned as all lower case which
	//requires your structs to use struct tags for each field. By not modifying column
	//names you will not need to use struct tags since column names can exactly match
	//exportable struct field names.
	//
	//http://jmoiron.github.io/sqlx/#:~:text=You%20can%20use%20the%20db%20struct%20tag%20to%20specify%20which%20column%20name%20maps%20to%20each%20struct%20field%2C%20or%20set%20a%20new%20default%20mapping%20with%20db.MapperFunc().%20The%20default%20behavior%20is%20to%20use%20strings.Lower%20on%20the%20field%20name%20to%20match%20against%20the%20column%20names.
	MapperFunc func(string) string

	//DeployQueries is a list of queries used to deploy the database schema. These
	//queries typically create tables or indexes or insert initial data into the
	//database. The queries listed here will be executed in order when DeploySchema()
	//is called. Make sure the order of the queries listed here makes sense for your
	//foreign key relationships! Each query should be safe to rerun multiple times!
	DeployQueries []string

	//DeployFuncs is a list of functions used to deploy the database. Use this for more
	//complicated deployment queries than the queries provided in DeployQueries. These
	//funcs get executed after all DeployQueries and should be used much more sparsely
	//compared to DeployQueries. Each func should be safe to rerun multiple times!
	DeployFuncs []DeployFunc

	//UpdateQueries is a list of queries used to update the database schema. These
	//queries typically add new columns, alter a column's type, or drop a column.
	//The queries listed here will be executed in order when UpdateSchema() is called.
	//Each query should be safe to rerun multiple times!
	UpdateQueries []string

	//UpdateFuncs is a list of functions used to deploy the database. Use this for more
	//complicated updates to the database schema or values stored within the database.
	//These funcs get executed after all UpdateQueries and should be used much more
	//sparsely compared to UpdateQueries. Each func should be safe to rerun multiple
	//times!
	UpdateFuncs []UpdateFunc

	//UpdateIgnoreErrorFuncs is a list of functions run when an UpdateQuery results in
	//an error and determins if the error can be ignored. This is used to ignore errors
	//for queries that aren't actual errors (ex.: adding a column that already exists).
	//Each func in this list should be very narrowly focused, checking both the query
	//and error, so that real errors aren't ignored by mistake.
	//
	//Some default funcs are predefined. See funcs in this package starting with UF...
	UpdateIgnoreErrorFuncs []UpdateIgnoreErrorFunc

	//TranslateCreateTableFuncs is a list of functions run against each DeployQuery
	//or UpdateQuery that contains a CREATE TABLE clause that modifies the query to
	//translate it from one database format to another. This tooling is used so that
	//you can write your CREATE TABLE queries in one database format (ex.: MySQL) but
	//deploy your database in multiple formats (ex.: MySQL & SQLite).
	//
	//A list of default funcs are predefined. See funcs in this package starting with
	//TF...
	TranslateCreateTableFuncs []func(string) string

	//TranslateUpdateFuncs is a list of functions run against each UpdateQuery that
	//modifies the query to translate it from one database format to another. See
	//TranslateCreateTableFuncs for more info.
	TranslateUpdateFuncs []func(string) string

	//Debug turns on diagnostic logging.
	Debug bool

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

// DBType returns a dbType. This is used when parsing a user-provided database type (such
// as from a configuration file) to convert to a db type defined in this package.
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

// errors
var (
	//ErrConnected is returned when a trying to establish a connection to an already
	//connected-to database.
	ErrConnected = errors.New("sqldb: connection already established")

	//ErrSQLitePathNotProvided is returned when user doesn't provided a path to the
	//SQLite database file, or the path provided is all whitespace.
	ErrSQLitePathNotProvided = errors.New("sqldb: SQLite path not provided")

	//ErrHostNotProvided is returned when user doesn't provide the host IP or FQDN
	//of a MySQL or MariaDB server.
	ErrHostNotProvided = errors.New("sqldb: database server host not provided")

	//ErrInvalidPort is returned when user doesn't provide, or provided an invalid
	//port, of a MySQL or MariaDB server.
	ErrInvalidPort = errors.New("sqldb: database server port invalid")

	//ErrNameNotProvided is returned when user doesn't provide a name of a database.
	ErrNameNotProvided = errors.New("sqldb: database name not provided")

	//ErrUserNotProvided is returned when user doesn't provide a user to connect to
	//the database server with.
	ErrUserNotProvided = errors.New("sqldb: database user not provided")

	//ErrPasswordNotProvided is returned when user doesn't provide the password to
	//connect to the database with. Blank passwords are not supported for security.
	ErrPasswordNotProvided = errors.New("sqldb: password for database user not provided")

	//ErrNoColumnsGiven is returned when user is trying to build a column list for a
	//query but no columns were provided.
	ErrNoColumnsGiven = errors.New("sqldb: no columns provided")

	//ErrExtraCommaInColumnString is returned when building a column string for a
	//query but an extra comma exists which would cause the query to not run correctly.
	//Extra commas are usually due to an empty column name being provided or a comma
	//being added to the column name by mistake.
	ErrExtraCommaInColumnString = errors.New("sqldb: extra comma in column name")
)

// config is the package level saved config. This stores your config when you want to
// use this package as a singleton and store your config for global use. This is used
// when you call one of the NewDefaultConfig() funcs which returns a pointer to this
// config.
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
		Type:       t,
		MapperFunc: DefaultMapperFunc,
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

			//Logging for development debugging.
			// cfg.debugPrintln("sqldb.buildConnectionString", "PRAGMA String:", pragmasToAdd)
			// cfg.debugPrintln("sqldb.buildConnectionString", "Path With PRAGMAS:", connString)
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
		//we should never hit this since we already validated the config in validate().
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

// Connect connects to the database. This sets the database driver in the config,
// establishes the database connection, and saves the connection pool for use in making
// queries. For SQLite this also runs any PRAGMA commands.
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

	//Set the mapper for mapping column names to struct fields.
	if cfg.MapperFunc != nil {
		conn.MapperFunc(cfg.MapperFunc)
	}

	//Save the connection for running future queries.
	cfg.connection = conn

	//Diagnostic logging.
	switch cfg.Type {
	case DBTypeMySQL, DBTypeMariaDB, DBTypeMSSQL:
		cfg.debugPrintln("sqldb.Connect", "Connecting to database "+cfg.Name+" on "+cfg.Host+" with user "+cfg.User)
	case DBTypeSQLite:
		cfg.debugPrintln("sqldb.Connect", "Connecting to database "+cfg.SQLitePath+".")
		cfg.debugPrintln("sqldb.Connect", "SQLite Library: "+GetSQLiteLibrary()+".")
		cfg.debugPrintln("sqldb.Connect", "PRAGMAs: "+cfg.SQLitePragmasAsString()+".")
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

// println performs log.Println if Debug is true for the config. This is just a helper
// func to remove the need for checking if Debug == true every time we want to log out
// debugging information.
func (cfg *Config) debugPrintln(v ...any) {
	if cfg.Debug {
		log.Println(v...)
	}
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
