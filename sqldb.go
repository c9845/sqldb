/*
Package sqldb implements tooling for interfacing with a SQL database.
This implements:
	- Connecting and disconnecting to the database.
	- Creating of database file if database is SQLite.
	- Handling some basic schema deployment functionality.

This package uses sqlx instead of the go standard library sql because sqlx provides some
additional tooling which is nice and makes using the database a bit easier.

You can use this package in two manners: store the database config and connection to the
package level variable for global use, or return the config and store it elsewhere in your
app. Storing the database connection within this package prevents you from connecting to
multiple databases. If you need to connect to multiple databases you will need to store
the configs and connections separately outside this package.

TODO:
- deploy scshema tooling
	- define deploy func type
	- how to handle order of deploy funcs? user provides a slice input which maintains order?
	- create database
	- creating of tables, inserting of data is left up to user via deploy funcs.
- update schema tooling
	- similar to deploy funcs but separate update funcs just for organizing and making sure update funcs aren't provided in deploy func list (or vice versa).
	- handle Update schema erros funcs?
		- or should this be separate/user defined?
		- lots to catch here...could be messy.
- tests!

*/
package sqldb

import (
	"errors"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql" //mysql and mariadb
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" //sqlite
)

//dbType is used to make sure a user provides a supported database type and
//cannot just provide an arbitrary string.
type dbType string

//types of databases this package supports
const (
	DBTypeMySQL   = dbType("mysql")
	DBTypeMariaDB = dbType("mariadb")
	DBTypeSQLite  = dbType("sqlite")
)

//list of valid db types, used during validation
var validDBTypes = []dbType{
	DBTypeMySQL,
	DBTypeMariaDB,
	DBTypeSQLite,
}

//journalMod is used to make sure user provies a supported journaling mode and
//cannot just provide an arbitrary string.
type journalMode string

//supported SQLite journal modes.
const (
	SQLiteJournalModeRollback = journalMode("DELETE")
	SQLiteJournalModeWAL      = journalMode("WAL")
)

//list of valid journal modes, used during validation
var validJournalModes = []journalMode{
	SQLiteJournalModeRollback,
	SQLiteJournalModeWAL,
}

type Config struct {
	//Type represents the type of database to use. This must match a given option noted
	//with this package since this package does not support every database type.
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
	//between the default rollback journal ("DELETE") and the write ahead log ("WAL"). WAL is useful
	//for when you have long-running reads on the database that are blocking access for writes.
	SQLitePragmaJournalMode journalMode

	//MapperFunc is used to override the mapping of database column names to struct field names or
	//struct tags. Mapping of column names is used during queries where StructScan(), Get(), or
	//Select() is used. By default, column names are not modified in any manner, unlike the default
	//for sqlx where column names are returned as all lower case. The default of not modifying column
	//names is more useful in our option since you will not need to use struct tags as much since
	//column names will more likely match exportable struct fields (with upper case first letter).
	//http://jmoiron.github.io/sqlx/#:~:text=You%20can%20use%20the%20db%20struct%20tag%20to%20specify%20which%20column%20name%20maps%20to%20each%20struct%20field%2C%20or%20set%20a%20new%20default%20mapping%20with%20db.MapperFunc().%20The%20default%20behavior%20is%20to%20use%20strings.Lower%20on%20the%20field%20name%20to%20match%20against%20the%20column%20names.
	MapperFunc func(string) string

	//TranslateCreateFuncs is the list of functions run against a CREATE query to translate
	//it from one database format to another. This list is populated initially with funcs
	//defined in this package when New...Config() or Default...Config() funcs are called.
	//However, this list can be modified by providing your own list of funcs. This functionality
	//is provided to support multiple database types for a single app; as in you define the CREATE
	//query for MariaDB but you give your users an option to deploy MariaDB or SQLite but you don't
	//want to have to rewrite your CREATE queries each time for each database type.
	TranslateCreateFuncs []TranslateFunc

	//driver is the database driver type chosen based on the Type provided. This will match one of
	//the values per the golang sql drivers. This is set once Connect() is called.
	driver string

	//connection is the established connection to a database for performing queries. This is
	//a "pooled" connection. Use this via the GetConnection() func to run queries against the db.
	connection *sqlx.DB
}

//defaults
const (
	defaultMySQLPort         = 3306
	defaultMariaDBPort       = 3306
	defaultSQLiteJournalMode = SQLiteJournalModeRollback
)

var (
	defaultMapperFunc = func(s string) string { return s }
)

//errors
var (
	//ErrInvalidDBType is returned when a user provided an database type that we don't support.
	ErrInvalidDBType = errors.New("sqldb: invalid db type provided")

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

	//ErrInvalidJournalMode is returned when user provides an invalid journalling mode for
	//a SQLite database. This shouldn't really ever occur since user has to use a defined
	//journalling mode.
	ErrInvalidJournalMode = errors.New("sqldb: Invalid SQLite journal mode")

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

//NewSQLiteConfig returns a config for connecting to a SQLite database. You will need to
//call Connect() to establish the connection to the database and store the config for use
//elsewhere in your app. The config will not be saved to the package level var for global
//use.
func NewSQLiteConfig(pathToFile string) (c Config, err error) {
	//build base config
	c = Config{
		Type:                    DBTypeSQLite,
		SQLitePath:              pathToFile,
		SQLitePragmaJournalMode: defaultSQLiteJournalMode,
		MapperFunc:              defaultMapperFunc,
		TranslateCreateFuncs:    defaultTranslateCreateFuncs(),
	}

	//validate the config so we can make sure user provided valid value(s). validate will be
	//called again when user tries to connect to the database too since the config could have
	//been modified.
	err = c.validate()
	return
}

//DefaultSQLiteConfig returns a reference to a config saved within this package for
//connecting to a SQLite database. Connect() will be called automatically to establish
//a connection to the database. This config is stored within this package for global use.
func DefaultSQLiteConfig(pathToFile string) (err error) {
	//make sure the package level config isn't already set to prevent accidentally
	//overwriting it.
	if config.connection != nil {
		return ErrConnected
	}

	//call NewSQLiteConfig to get the config, however, we will store it to the
	//package level variable and return it instead.
	cfg, err := NewSQLiteConfig(pathToFile)
	if err != nil {
		return
	}

	config = cfg

	//TODO: call connect??

	return
}

//NewMySQLConfig returns a config for connecting to a MySQL database. This is just a helper func
//around setting the proper fields required to connect to a MySQL database and assumes some default
//values. You will need to call Connect() on the config and save the config for use elsewhere in your
//app. The config will not be saved to the package level var for global use.
func NewMySQLConfig(host string, port uint, name, user, password string) (c Config, err error) {
	c = Config{
		Type:                 DBTypeMariaDB,
		Host:                 host,
		Port:                 port,
		Name:                 name,
		User:                 user,
		Password:             password,
		MapperFunc:           defaultMapperFunc,
		TranslateCreateFuncs: defaultTranslateCreateFuncs(),
	}

	//validate the config so we can make sure user provided valid value(s). validate will be
	//called again when user tries to connect to the database too since the config could have
	//been modified.
	err = c.validate()
	return
}

//DefaultMySQLConfig returns a reference to a config saved within this package for
//connecting to a MySQL database. Connect() will be called automatically to establish
//a connection to the database. This config is stored within this package for global use.
func DefaultMySQLConfig(host string, port uint, name, user, password string) (err error) {
	//make sure the package level config isn't already set to prevent accidentally
	//overwriting it.
	if config.connection != nil {
		return ErrConnected
	}

	//call NewSQLiteConfig to get the config, however, we will store it to the
	//package level variable and return it instead.
	cfg, err := NewMySQLConfig(host, port, name, user, password)
	if err != nil {
		return
	}

	config = cfg

	//TODO: call connect??

	return
}

//NewMariaDBConfig returns a config for connecting to a MariaDB database. This is just a wrapper
//around GetMySQLConfig but with setting the database type properly.
func NewMariaDBConfig(host string, port uint, name, user, password string) (c Config, err error) {
	c, err = NewMySQLConfig(host, port, name, user, password)
	c.Type = DBTypeMariaDB

	//we don't need to call validate() here since it was called in NewMySQLConfig() and we know
	//the change made to the Type field is a valid option.
	return
}

//DefaultMariaDBConfig returns a reference to a config saved within this package for
//connecting to a MariaDB database. Connect() will be called automatically to establish
//a connection to the database. This config is stored within this package for global use.
func DefaultMariaDBConfig(host string, port uint, name, user, password string) (err error) {
	//make sure the package level config isn't already set to prevent accidentally
	//overwriting it.
	if config.connection != nil {
		return ErrConnected
	}

	//call NewSQLiteConfig to get the config, however, we will store it to the
	//package level variable and return it instead.
	cfg, err := NewMariaDBConfig(host, port, name, user, password)
	if err != nil {
		return
	}

	config = cfg

	//TODO: call connect??

	return
}

//validate handles validation of a provided config.
func (c *Config) validate() (err error) {
	//handle some sanitizing
	c.SQLitePath = strings.TrimSpace(c.SQLitePath)
	c.Host = strings.TrimSpace(c.Host)
	c.Name = strings.TrimSpace(c.Name)
	c.User = strings.TrimSpace(c.User)

	//check if a valid db type was provided.
	//This should never result in "false" since the user has to provide one of our defined
	//database types due to the "type" declaration (db type isn't just a string value).
	if !isTypeValid(c.Type, validDBTypes) {
		return ErrInvalidDBType
	}

	//check other details based on db type
	if c.Type == DBTypeSQLite {
		if c.SQLitePath == "" {
			return ErrSQLitePathNotProvided
		}
		if c.SQLitePragmaJournalMode == "" {
			c.SQLitePragmaJournalMode = defaultSQLiteJournalMode
		}
		if !isJournalModeValid(c.SQLitePragmaJournalMode, validJournalModes) {
			return ErrInvalidJournalMode
		}
	}
	if c.Type == DBTypeMySQL || c.Type == DBTypeMariaDB {
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

//isTypeValid checks if a provided database type is a valid supported database type.
func isTypeValid(needle dbType, haystack []dbType) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}

	return false
}

//isJournalModeValid checks if a provided journal mode is a valid supported database type.
func isJournalModeValid(needle journalMode, haystack []journalMode) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}

	return false
}

//buildConnectionString creates the string used to connect to a database. The connection string
//returned is build for a specific database type since each type has different parameters needed
//for the connection.
//Note that when building the connection string for mysql or mariadb, we have to skip the database
//name if we are deploying the database, since, obviously, the database doesn't exist yet.
func (c *Config) buildConnectionString(deployingDB bool) (connString string) {
	switch c.Type {
	case DBTypeMariaDB, DBTypeMySQL:
		//for mysql or mariadb, use connection string tooling and formatter
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
		//For sqlite, the connection string is simply a path to a file, however we do
		//have to add extra pragma stuff based on journaling mode we want sqlite to be
		//in. We set the pragma at the time of connection instead of via queries once
		//the db is connected just for ease, simplicity, and not having to run a bunch
		//of queries.
		//
		//Note that since the connection string will have extra stuff appended to it, it
		//will no longer be a valid path to a file and will cause issues if used as such,
		//especially on linux systems. You should have already confirmed the path was to
		//a valid file (or a place where a file can be created).
		//
		//For more info on sqlite pragmas, journalling mode, and connection strings:
		// - https://www.sqlite.org/wal.html
		// - https://github.com/mattn/go-sqlite3#connection-string
		connString = c.SQLitePath

		v := url.Values{}
		if c.SQLitePragmaJournalMode == SQLiteJournalModeWAL {
			v.Set("_journal_mode", "WAL")
		} else {
			//sqlite default
			v.Set("_journal_mode", "DELETE")
		}

		u, err := url.Parse(connString)
		if err != nil {
			log.Println("Could not parse connection string.", err)
			return
		}

		u.RawQuery = v.Encode()
		connString = u.String()

		//no default since we already validated that the provided db type is a valid value
	}

	return
}

//Connect connects to the database. This sets the database driver in the config and saves the
//connection pool for use in making queries.
func (c *Config) Connect() (err error) {
	//make sure the connection isn't already established to prevent overwriting it.
	if c.connection != nil {
		return ErrConnected
	}

	//make sure the config is valid
	err = c.validate()
	if err != nil {
		return
	}

	//get the connection string
	connString := c.buildConnectionString(false)

	//get the correct driver based on the database type
	switch c.Type {
	case DBTypeMySQL, DBTypeMariaDB:
		c.driver = "mysql"
	case DBTypeSQLite:
		c.driver = "sqlite3"

		//no default since we already validated that the db type provided is valid in validate()
	}

	//Connect to the database.
	//For SQLite, check if the database file exists. This will not create the database file, you
	//should call DeployDB() first.
	if c.Type == DBTypeSQLite {
		_, err = os.Stat(c.SQLitePath)
		if os.IsNotExist(err) {
			return err
		}
	}

	//This doesn't really establish a connection to the database, it just "builds" the connection.
	//The connection is established with Ping() below.
	conn, err := sqlx.Open(c.driver, connString)
	if err != nil {
		return
	}

	//Test the connection to the database to make sure it works. This opens the connection for future
	//use.
	err = conn.Ping()
	if err != nil {
		return
	}

	//Set the mapper for mapping column names to struct fields.
	if c.MapperFunc != nil {
		conn.MapperFunc(c.MapperFunc)
	}

	//Save the connection for running queries.
	c.connection = conn

	return
}

//Close closes the connection to the database.
func (c *Config) Close() (err error) {
	return c.connection.Close()
}

//GetDefaultConfig returns the package level saved config. The config may or may not be connected
//to the database (use Connected() func). This is used to return the config to a user for use
//elsewhere to (1) inspect the config, (2) get a connection to run a query, or (3) connect or
//disconnect to the database.
func GetDefaultConfig() (c *Config) {
	return &config
}

//Connected returns if the config represents an established connection to the database.
func (c *Config) Connected() bool {
	return c.connection != nil
}

//Columns is used to hold columns for a query. This helps in organizing a query you are building.
type Columns []string

//Bindvars holds the parameters you want to use in a query. This helps in organizing a query
//you are building.
type Bindvars []interface{}

//buildColumnString takes a slice of strings, representing columns, and returns them as
//a string to be used in a sql SELECT, INSERT, or UPDATE. This simply formats the columns
//for the query type correctly (concats them together with a seperator and/or parameter
//placeholder (i.e.: ?)) and returns the parameter placholder string to be used for the
//VALUES clause in an INSERT query as needed. Using this func instead of building column
//list manually ensures column list is formatted correctly and count of parameter
//placeholders match the count of columns.
func (cols Columns) buildColumnString(forUpdate bool) (colString, valString string, err error) {
	//make sure at least one column is provided
	if len(cols) == 0 {
		err = ErrNoColumnsGiven
		return
	}

	//build the strings
	if forUpdate {
		//For an UPDATE query, we just append the parameter placeholder to each column
		//name. The first line here adds the =? to each provided column except the last
		//in the slice, the second line adds the =? to the last column.
		colString = strings.Join(cols, "=?,")
		colString += "=?"

	} else {
		//For a SELECT or INSERT query, we just append a comma to separate each column.
		colString = strings.Join(cols, ",")

		//We also need a list of parameter placeholders, also separated by commas. However,
		//the final comma after the last placeholder needs to be stripped to not cause errors.
		valString = strings.Repeat("?,", len(cols))
		valString = valString[:len(valString)-1]
	}

	//Check for any double commas. This is usually caused by a column name being given with
	//a comma already appended or an empty column was provided.
	if idx := strings.Index(colString, ",,"); idx != -1 {
		err = ErrDoubleCommaInColumnString
		return
	}

	return
}

//ForSelect builds the column string for a SELECT query.
func (c Columns) ForSelect() (colString string, err error) {
	colString, _, err = c.buildColumnString(false)
	return
}

//ForInsert builds the column string for an INSERT query.
func (c Columns) ForInsert() (colString, valString string, err error) {
	colString, valString, err = c.buildColumnString(false)
	return
}

//ForUpdate builds the column string for an UPDATE query.
func (c Columns) ForUpdate() (colString string, err error) {
	colString, _, err = c.buildColumnString(true)
	return
}

//IsSQLite returns true if the database is a SQLite database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func (c *Config) IsSQLite() bool {
	return c.Type == DBTypeSQLite
}

//IsMySQL returns true if the database is a MySQL database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func (c *Config) IsMySQL() bool {
	return c.Type == DBTypeMySQL
}

//IsMariaDB returns true if the database is a MariaDb database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func (c *Config) IsMariaDB() bool {
	return c.Type == DBTypeMariaDB
}

//IsMySQLOrMariaDB returns if the database is a MySQL or MariaDB. This is useful
//since MariaDB is a fork of MySQL and most things are compatible; this way you
//don't need to check IsMySQL and IsMariaDB.
func (c *Config) IsMySQLOrMariaDB() bool {
	return c.Type == DBTypeMySQL || c.Type == DBTypeMariaDB
}

//GetSQLiteVersion returns the version of SQLite that is embedded into the app. This is
//used for diagnostics. This works by creating a temporary in-memory SQLite database to
//run query against.
func GetSQLiteVersion() (version string, err error) {
	conn, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		return
	}
	defer conn.Close()

	//query for version
	q := "SELECT sqlite_version()"
	err = conn.Get(&version, q)
	return
}
