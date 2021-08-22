/*
Package sqldb implements tooling for interfacing with a SQL database.
This implements:
	- Connecting and disconnecting to the database.
	- Creating of database file if database is SQLite.
	- Handling some basic schema deployment functionality.

This package uses sqlx instead of the go standard library sql because sqlx provides some
additional tooling which is nice and makes using the database a bit easier.

TODO:
- mapper func in config. otherwise use default of not changing given column names.
	- https://jmoiron.github.io/sqlx/#:~:text=You%20can%20use%20the%20db%20struct%20tag%20to%20specify%20which%20column%20name%20maps%20to%20each%20struct%20field%2C%20or%20set%20a%20new%20default%20mapping%20with%20db.MapperFunc().%20The%20default%20behavior%20is%20to%20use%20strings.Lower%20on%20the%20field%20name%20to%20match%20against%20the%20column%20names.
	- default is to not touch column names so that you don't need to have tags on each struct field, i.e. we assume column names in db match struct fields exactly (case sensitive).
- validation
- save config to package level var.
- allow saving config externally to allow for multiple sql db connections, use with dependency injection, etc.
- connect to db (using internal or external config).
- build connection string.
	- separate func from connect for use when deploying db and the db itself isn't created yet.
- close connection to db (internal or external config).
- build column string
	- helper funcs for select, insert, update.
- IsDb... funcs (internal or external config).
- translate
	- separate file?
	- lots of docs for this.
- getsqlite version func.
	- maybe same for mariadb & sqlite?
- deploy schema tooling
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

	"github.com/jmoiron/sqlx"
)

//dbType is used to make sure a user provides a supported database type
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
	SQLitePragmaJournalMode string

	//driver is the database driver type chosen based on the Type provided. This will match one of
	//the values per the golang sql drivers.
	driver string

	//connection is the established connection to a database for performing queries. This is
	//a "pooled" connection. Use this via the GetConnection() func.
	connection *sqlx.DB
}

//defaults
const (
	defaultMySQLPort               = 3306
	defaultMariaDBPort             = 3306
	defaultSQLitePragmaJournalMode = "DELETE"
)

//Columns is used to hold columns for a query. This helps in organizing a query you are building.
type Columns []string

//Bindvars holds the parameters you want to use in a query. This helps in organizing a query you are
//building.
type Bindvars []interface{}

//errors
var (
	//ErrInvalidDBType is returned when a user provided an database type that we don't support.
	ErrInvalidDBType = errors.New("sqldb: invalid db type provided")
)

//NewSQLiteConfig returns a config for connecting to a SQLite database. This is just a helper func
//around setting the proper fields required to connect to a SQLite database and assumes some default
//values. You will need to call Connect() on the config and save the config for use elsewhere in your
//app (this allows for multiple database connections).
func NewSQLiteConfig(pathToFile string) (c Config, err error) {
	//build base config
	c = Config{
		Type:                    DBTypeSQLite,
		SQLitePath:              pathToFile,
		SQLitePragmaJournalMode: defaultSQLitePragmaJournalMode,
	}

	//validate the config so user doesn't have to call validate() separately.
	err = c.validate()
	return
}

//NewMySQLConfig returns a config for connecting to a MySQL database. This is just a helper func
//around setting the proper fields required to connect to a MySQL database and assumes some default
//values. You will need to call Connect() on the config and save the config for use elsewhere in your
//app (this allows for multiple database connections).
func NewMySQLConfig(host string, port uint, name, user, password string) (c Config, err error) {
	c = Config{
		Type:     DBTypeMariaDB,
		Host:     host,
		Port:     port,
		Name:     name,
		User:     user,
		Password: password,
	}

	//validate the config so user doesn't have to call validate() separately.
	err = c.validate()
	return
}

//NewMariaDBConfig returns a config for connecting to a MariaDB database. This is just a wrapper
//around GetMySQLConfig but with setting the database type properly.
func NewMariaDBConfig(host string, port uint, name, user, password string) (c Config, err error) {
	c, err = NewMySQLConfig(host, port, name, user, password)
	c.Type = DBTypeMariaDB
	return
}

//config is the package level saved config. This stores your config when you want to store it for global use.
//This is populated by the Connect() function.
var config Config

//NewDefaultConfig returns a reference to a config saved within this package. This config's data is saved
//in this package for global use elsewhere in your app (using sqldb.GetConfig(), sqldb.GetConnection(), etc.).
//Note that since this config is saved to a package level variable, you can only connect to one database at
//a time.
func NewDefaultConfig(t dbType) (c *Config, err error) {
	//set defaults based on db type
	switch t {
	case DBTypeSQLite:
		cfg, innerErr := NewSQLiteConfig("")
		if innerErr != nil {
			err = innerErr
			return
		}
		config = cfg

	default:
		err = ErrInvalidDBType
		return
	}

	c = &config
	return
}

//validate handles validation of a provided config.
func (c *Config) validate() (err error) {
	//check if a valid db type was provided.
	//This should never result in "false" since the user has to provide one of our defined
	//database types due to the "type" declaration (db type isn't just a string value).
	if !isTypeValid(c.Type, validDBTypes) {
		return
	}

	return
}

//isTypeValid checks if a provided database type, from a config, is a valid supported database type.
func isTypeValid(needle dbType, haystack []dbType) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}

	return false
}

//func (c *config).Connect()
