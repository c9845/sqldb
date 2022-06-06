/*
Package sqldb handles a establishing and managing a database connection, deploying and
updating a database schema, translating queries from one database format to another,
and provides various other tooling for interacting with SQL databases.

This package uses "sqlx" instead of the go standard library "sql" package because "sqlx"
provides some additional tooling which makes using the database a bit easier (i.e.: Get(),
Select(), and StructScan() that can thus be in queries).

You can use this package in two manners: store the database configuration globally in the
package level variable, or store the configuration elsewhere in your app. Storing the
configuration yourself allows for connecting to multiple databases at one time.


Deploying a Database

Deploying of a database schema is done via queries stored as strings and then provided
in the DeployQueries field of your config. When you call the DeploySchema() function, a
connection to the database (server or file) is established and then each query is run. You
should take care to provide the list of queries in DeployQueries in an order where foreign
key tables are created first so you don't get unnecessary errors. DeployQueries can also be
used to insert initial data into a database. After each DeployQuery is run, DeployFuncs are
run. These functions are used to handle more complex operations to deploy your database then
simple queries allow. Use TranslateCreateTableFuncs to automatically translate queries from
one database format to another (i.e.: MySQL to SQLite) so that you do not need to maintain
and list queries for each database type separately.


Updating a Schema

Updating a database schema happens in a similar manner to deploying, a list of queries is
run against the database. These queries run encapsulated in a transaction so that either the
entire database is updated, or none of queries are applied. This is done to eliminate the
possibility of a partially updated database schema. Each update query is run through a list
of error analyzer functions, UpdateIgnoreErrorFuncs, to determine if an error can be ignored.
This is typically used to ignore errors for when you are adding a column that already exists,
removing a column that is already removed, etc. Take note that SQLite does not allow for columns
to be modified!

Extremely important: You should design your queries that deploy or update the schema to be
safe to rerun multiple times. You don't want issues to occur if a user interacting with your
app somehow tries to deploy the database over and over or update it after it has already been
updated. For example, use "IF NOT EXISTS".


SQLite Library

This package support two SQLite libraries, mattn/sqlite3 and modernc/sqlite. The mattn
library encapsulates the SQLite C source code and requires CGO for compilation which
can be troublesome for cross-compiling. The modernc library is an automatic translation
of the C source code to golang, however, it isn't the "source" SQLite C code and therefore
doesn't have the same extent of testing.

As of now, mattn is the default if no build tags are provided. This is simply due to the
longer history of this library being available and the fact that this uses the SQLite
C source code.

Use either library with build tags:
  go build -tags mattn ...
  go build -tags modernc ...
  go run -tags mattn ...
  go run -tags modernc ...
*/
package sqldb

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql" //mysql and mariadb, not an empty import b/c we use it to generate the connection string
	//sqlite is imported in other -sqlite- files due to build tags.

	"github.com/jmoiron/sqlx"
	"golang.org/x/exp/slices"
)

//Config is the details used for establishing and using a database connection.
type Config struct {
	//Type represents the type of database to use.
	Type dbType

	//Host is the IP or FQDN of the database server. This is not required for SQLite
	//databases.
	Host string

	//Port is the port the database listens on. This is not required for SQLite databases.
	Port uint

	//Name is the name of the database to connect to. This is not required for SQLite databases.
	Name string

	//User is the user who has access to the database. This is not required for SQLite databases.
	User string

	//Password is the matching user's password. This is not required for SQLite databases.
	Password string

	//SQLitePath is the path where the SQLite database file is located. This is only required for
	//SQLite databases.
	SQLitePath string

	//SQLitePragmaJournalMode sets the SQLite database journalling mode. This is used to switch
	//between the default rollback journal ("DELETE") and the write ahead log ("WAL"). WAL is
	//useful for when you have long-running reads on the database that are blocking access for
	//writes.
	SQLitePragmaJournalMode journalMode

	//MapperFunc is used to override the mapping of database column names to struct field names
	//or struct tags. Mapping of column names is used during queries where StructScan(), Get(),
	//or Select() is used. By default, column names are not modified in any manner, unlike the
	//default for "sqlx" where column names are returned as all lower case. The default of not
	//modifying column names is more useful in our option since you will not need to use struct
	//tags as much since column names can exactly match exportable struct field names (typically
	//struct fields used for storing column data are exported).
	//http://jmoiron.github.io/sqlx/#:~:text=You%20can%20use%20the%20db%20struct%20tag%20to%20specify%20which%20column%20name%20maps%20to%20each%20struct%20field%2C%20or%20set%20a%20new%20default%20mapping%20with%20db.MapperFunc().%20The%20default%20behavior%20is%20to%20use%20strings.Lower%20on%20the%20field%20name%20to%20match%20against%20the%20column%20names.
	MapperFunc func(string) string

	//DeployQueries is a list of queries used to deploy the database schema. These queries
	//typically create tables or indexes or insert initial data into the database. The queries
	//listed here will be executed in order when DeploySchema() is called. The list of queries
	//provided must be in the correct order to create any tables referred to by a foreign key
	//first!
	DeployQueries []string

	//DeployFuncs is a list of functions used to deploy the database. Use this for more
	//complicated deployment queries than the queries provided in DeployQueries. This list
	//gets executed after the DeployQueries list. You will need to establish and close the
	//db connection in each funtion. This is typically used when you want to reuse other funcs
	//you have defined to handle inserting initial users or creating of default values. This
	//should rarely be used!
	DeployFuncs []func() error

	//TranslateCreateTableFuncs is a list of functions run against each DeployQueries that
	//contains a "CREATE TABLE" clause that modifies the query to translate it from one
	//database format to another. This functionality is provided so that you can write your
	//CREATE TABLE queries in one database's format (ex.: MySQL) but deploy your database
	//in multiple formats (ex.: MySQL & SQLite).
	//Some default funcs are predefined, names as TF...
	TranslateCreateTableFuncs []TranslateFunc

	//UpdateQueries is a list of queries used to update the database schema. These queries
	//typically add new columns, alter a column's type, or alter values stored in a column.
	//The queries listed here will be executed in order when UpdateSchema() is called. The
	//queries should be safe to be rerun multiple times (i.e.: if UpdateSchema() is called
	//automatically each time your app starts).
	UpdateQueries []string

	//UpdateIgnoreErrorFuncs is a list of functions run when an UpdateQuery results in an
	//error and determins if the error can be ignored. This is used to ignore errors for
	//queries that aren't actual errors (ex.: adding a column that already exists). Each
	//func in this list should be very narrowly focused, checking both the query and error,
	//so that real errors aren't ignored by mistake.
	//Some default funcs are predefined, named as UF...
	UpdateIgnoreErrorFuncs []UpdateIgnoreErrorFunc

	//Debug turns on diagnostic logging.
	Debug bool

	//connection is the established connection to a database for performing queries. This is
	//a "pooled" connection. Use this via the GetConnection() func to run queries against the db.
	connection *sqlx.DB
}

//Supported databases.
type dbType string

const (
	DBTypeMySQL   = dbType("mysql")
	DBTypeMariaDB = dbType("mariadb")
	DBTypeSQLite  = dbType("sqlite")
)

var validDBTypes = []dbType{
	DBTypeMySQL,
	DBTypeMariaDB,
	DBTypeSQLite,
}

//DBType returns a dbType. This is used when parsing a user-provided db type to match
//the db types defined in this package.
func DBType(s string) dbType {
	return dbType(s)
}

//valid checks if a provided dbType is one of our supported databases. This is used
//when validating db config/connection information.
func (t dbType) valid() error {
	contains := slices.Contains(validDBTypes, t)
	if contains {
		return nil
	}

	return fmt.Errorf("invalid db type, should be one of '%s', got '%s'", validDBTypes, t)
}

//Supported SQLite journal modes.
type journalMode string

const (
	SQLiteJournalModeRollback = journalMode("DELETE")
	SQLiteJournalModeWAL      = journalMode("WAL")
)

//JournalMode returns a journalMode. This is provided so that other journal modes
//besides the const defined Rollback/DELETE and WAL can be used (ex.: TRUNCATE).
//Providing this tooling allows for a more "I meant that" appearance in code when
//using a non-const defined journal mode. This is also why a "Valid()" func is not
//defined for journalMode since other modes can be provided.
func JournalMode(s string) journalMode {
	return journalMode(s)
}

//defaults
const (
	defaultMySQLPort         = 3306
	defaultMariaDBPort       = 3306
	defaultSQLiteJournalMode = SQLiteJournalModeRollback
)

//errors
var (
	//ErrConnected is returned when a database connection is already established and a user is
	//trying to connect or trying to modify a config that is already in use.
	ErrConnected = errors.New("sqldb: connection already established")

	//ErrSQLitePathNotProvided is returned when user doesn't provided a path to the SQLite database
	//file, or the path provided is all whitespace.
	ErrSQLitePathNotProvided = errors.New("sqldb: SQLite path not provided")

	//ErrHostNotProvided is returned when user doesn't provide the host (IP or FQDN) where a MySQL
	//or MariaDB server is running.
	ErrHostNotProvided = errors.New("sqldb: Database server host not provided")

	//ErrInvalidPort is returned when user doesn't provide, or provided an invalid port, for where
	//the MySQL or MariaDB server is running.
	ErrInvalidPort = errors.New("sqldb: Database server port invalid")

	//ErrNameNotProvided is returned when user doesn't provide a name for a database.
	ErrNameNotProvided = errors.New("sqldb: Database name not provided")

	//ErrUserNotProvided is returned when user doesn't provide a user to connect to the database
	//server with.
	ErrUserNotProvided = errors.New("sqldb: Database user not provided")

	//ErrPasswordNotProvided is returned when user doesn't provide the password to connect to the
	//database with. We do not support blank passwords for security.
	ErrPasswordNotProvided = errors.New("sqldb: Password for database user not provided")

	//ErrNoColumnsGiven is returned when user is trying to build a column list for a query but
	//not columns were provided.
	ErrNoColumnsGiven = errors.New("sqldb: No columns provided")

	//ErrDoubleCommaInColumnString is returned when building a column string for a query but
	//a double comma exists which would cause the query to not run correctly. Double commas
	//are usually due to an empty column name being provided or a comma being added to the
	//column name by mistake.
	ErrDoubleCommaInColumnString = errors.New("sqldb: Extra comma in column name")
)

//config is the package level saved config. This stores your config when you want to store it for
//global use. This is used when you call one of the NewDefaultConfig() funcs which return a pointer
//to this config.
var config Config

//NewConfig returns a base configuration that will need to be modified for use with a db.
//Typically you would use New...Config() instead.
func NewConfig(t dbType) (c *Config, err error) {
	err = t.valid()
	if err != nil {
		return
	}

	c = &Config{
		Type:       t,
		MapperFunc: DefaultMapperFunc,
	}
	return
}

//DefaultConfig initializes the package level config. This wraps around NewConfig(). Typically
//you would use Default...Config() instead.
func DefaultConfig(t dbType) (err error) {
	cfg, err := NewConfig(t)
	config = *cfg

	return
}

//Save saves a configuration to the package level config. Use this in conjunction with New...Config
//when you want to heavily customize the config. This does not use a method so that any modifications
//done to the original config aren't propagated to the package level config without calling Save()
//again.
func Save(c Config) {
	config = c
}

//validate handles validation of a provided config. This is called in Connect().
func (c *Config) validate() (err error) {
	//sanitize
	c.SQLitePath = strings.TrimSpace(c.SQLitePath)
	c.Host = strings.TrimSpace(c.Host)
	c.Name = strings.TrimSpace(c.Name)
	c.User = strings.TrimSpace(c.User)

	//Make sure db type is valid. A user can modify this to a string via
	//"c.Type = 'asdf'" even though Type has a specific dbType type. This catches
	//this slight possibility.
	err = c.Type.valid()
	if err != nil {
		return
	}

	//check config based on db type since each type of db has different requirements
	switch c.Type {
	case DBTypeSQLite:
		if c.SQLitePath == "" {
			return ErrSQLitePathNotProvided
		}
		if c.SQLitePragmaJournalMode == "" {
			c.SQLitePragmaJournalMode = defaultSQLiteJournalMode
		}

		//We don't check if journal mode is valid since non-const defined journal
		//modes can also be provided. This allows for using other SQLite journal
		//modes that aren't defined in this package. An error will most likely be
		//kicked out by SQLite when setting the journal mode if it is invalid.

	case DBTypeMySQL, DBTypeMariaDB:
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
	}

	return
}

//buildConnectionString creates the string used to connect to a database. The connection string
//returned is built for a specific database type since each type has different parameters needed
//for the connection. Note that when building the connection string for mysql or mariadb, we
//have to omit the databasename if we are deploying the database, since, obviously, the database
//does not exist yet. The database name is only appended to the connection string when the
//database exists.
func (c *Config) buildConnectionString(deployingDB bool) (connString string) {
	switch c.Type {
	case DBTypeMariaDB, DBTypeMySQL:
		//for mysql or mariadb, use connection string tooling and formatter instead of
		//us building the connection string manually.
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
		//for sqlite, the connection string is simply a path to a file. We do not set
		//pragmas in the connection string since it is messy, instead we set them with
		//PRAGMA queries once the database connection is established.
		connString = c.SQLitePath

	default:
		//we should never hit this since we already validated the config in validate().
	}

	return
}

//getDriver returns the golang sql driver used for the chosen database type.
func getDriver(t dbType) (driver string, err error) {
	if err := t.valid(); err != nil {
		return "", err
	}

	switch t {
	case DBTypeMySQL, DBTypeMariaDB:
		driver = "mysql"
	case DBTypeSQLite:
		driver = sqliteDriverName //see sqlite subfiles based on library used.
	}

	return
}

//Connect connects to the database. This sets the database driver in the config, establishes
//the database connection, and saves the connection pool for use in making queries. For SQLite
//this also runs any PRAGMA commands.
func (c *Config) Connect() (err error) {
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

	//Get the connection string inclusive of the db name.
	connString := c.buildConnectionString(false)

	//Get the correct driver based on the database type.
	//This is set based on the empty (_) imported package.
	//Error should never occur this since we already validated the config in validate().
	driver, err := getDriver(c.Type)
	if err != nil {
		return
	}

	//Connect to the database.
	//For SQLite, check if the database file exists. This func will not create the database
	//file. The database file needs to be created first with Deploy().
	if c.Type == DBTypeSQLite {
		_, err = os.Stat(c.SQLitePath)
		if os.IsNotExist(err) && c.SQLitePath != InMemoryFilePathRacy && c.SQLitePath != InMemoryFilePathRaceSafe {
			return err
		}
	}

	//This doesn't really establish a connection to the database, it just "builds" the
	//connection. The connection is established with Ping() below.
	//Note no defer conn.Close() since we want to keep the db connection alive for future
	//use in running queries.
	conn, err := sqlx.Open(driver, connString)
	if err != nil {
		return
	}

	//Test the connection to the database to make sure it works. This opens the connection
	//for future use.
	err = conn.Ping()
	if err != nil {
		return
	}

	//Run any PRAGMA queries for SQLite as needed
	//We already validated the value stored in SQLitePragma... so we can just pass it as a
	//value to the query.
	//Could not use bindvar parameter here for some reason.
	//Cannot use prepare, then exec, as this causes a "cannot commit transaction - SQL statements in process" error.
	if c.Type == DBTypeSQLite {
		q := "PRAGMA journal_mode = " + string(c.SQLitePragmaJournalMode)
		_, innerErr := conn.Exec(q)
		if innerErr != nil {
			innerErr = fmt.Errorf("could not set journal_mode, %w", innerErr)
			return innerErr
		}
	}

	//Set the mapper for mapping column names to struct fields.
	if c.MapperFunc != nil {
		conn.MapperFunc(c.MapperFunc)
	}

	//Save the connection for running queries.
	c.connection = conn

	//diagnostic logging
	if c.Debug {
		switch c.Type {
		case DBTypeMySQL, DBTypeMariaDB:
			log.Println("sqldb.Connect", "Connecting to database "+c.Name+" on "+c.Host+" with user "+c.User)
		case DBTypeSQLite:
			log.Println("sqldb.Connect", "Connecting to database "+c.SQLitePath+" (Journal Mode: "+string(c.SQLitePragmaJournalMode)+")")
		}
	}

	return
}

//Connect handles the connection to the database using the default package level config
func Connect() (err error) {
	return config.Connect()
}

//Close closes the connection to the database.
func (c *Config) Close() (err error) {
	return c.connection.Close()
}

//Close closes the connection using the default package level config.
func Close() (err error) {
	return config.Close()
}

//Connected returns if the config represents an established connection to the database.
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
	if err != nil {
		return false
	}

	//a connection was been established and is open, ping succeeded
	return true
}

//Connected returns if the config represents an established connection to the database.
func Connected() bool {
	return config.Connected()
}

//Connection returns the database connection stored in a config for use in running queries
func (c *Config) Connection() *sqlx.DB {
	return c.connection
}

//Connection returns the database connection for the package level config.
func Connection() *sqlx.DB {
	return config.Connection()
}

//IsMySQLOrMariaDB returns if the database is a MySQL or MariaDB. This is useful
//since MariaDB is a fork of MySQL and most things are compatible; this way you
//don't need to check IsMySQL and IsMariaDB.
func (c *Config) IsMySQLOrMariaDB() bool {
	return c.Type == DBTypeMySQL || c.Type == DBTypeMariaDB
}

//DefaultMapperFunc is the default MapperFunc set on configs. It returns the column
//names unmodified.
func DefaultMapperFunc(s string) string {
	return s
}

//GetDefaultConfig returns the package level saved config.
func GetDefaultConfig() (c *Config) {
	return &config
}

//MapperFunc sets the mapper func for the package level config.
func MapperFunc(m func(string) string) {
	config.MapperFunc = m
}

//TranslateCreateTableFuncs sets the translation funcs for creating a table for the package
//level config.
func TranslateCreateTableFuncs(fs []TranslateFunc) {
	config.TranslateCreateTableFuncs = fs
}

//SetDeployQueries sets the list of queries to deploy the database schema for the package
//level config. Beware of the order! Queries must be listed in order where any foreign
//key tables were created prior.
func SetDeployQueries(qs []string) {
	config.DeployQueries = qs
}

//SetDeployFuncs sets the list of funcs to deploy the database schema for the package
//level config.
func SetDeployFuncs(fs []func() error) {
	config.DeployFuncs = fs
}

//SetUpdateQueries sets the list of funcs to update the database schema for the package level
//config.
func SetUpdateQueries(qs []string) {
	config.UpdateQueries = qs
}

//SetUpdateIgnoreErrorFuncs sets the list of funcs to handle update schema errors for the
//package level config.
func SetUpdateIgnoreErrorFuncs(fs []UpdateIgnoreErrorFunc) {
	config.UpdateIgnoreErrorFuncs = fs
}

//debugLog is a helper function to clean up logging out debugging information. This
//removes the need for if c.Debug {} checks whenever we want to log out something.
func (c *Config) debugLog(s string) {
	if c.Debug {
		log.Println(s)
	}
}
