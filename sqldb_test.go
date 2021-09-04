package sqldb

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

//Get diretory to test file aka the package root directory so
//we can build paths to test files.
/* dir, err := os.Getwd()
if err != nil {
	t.Fatal(err)
	return
} */

func TestDefaultMapperFunc(t *testing.T) {
	in := "string"
	out := defaultMapperFunc(in)
	if in != out {
		t.Fatal("defaultMapperFunc modified string but should not have.")
		return
	}
}

func TestNewSQLiteConfig(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Provide a path and make sure it was saved properly.
	p := "/fake/path/to/sqlite.db"
	c, err := NewSQLiteConfig(p)
	if err != nil {
		t.Fatal("Error occured but should not have.", err)
		return
	}

	if c.SQLitePath != p {
		t.Fatal("Path not saved correctly.")
		return
	}
	if c.Type != DBTypeSQLite {
		t.Fatal("DB type not set properly.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a path, which is invalid.
	_, err = NewSQLiteConfig("")
	if err == nil {
		t.Fatal("Error did not occur but should have because of invalid path.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestDefaultSQLiteConfig(t *testing.T) {
	config.connection = nil

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Provide a path.
	p := "/fake/path/to/sqlite.db"
	err := DefaultSQLiteConfig(p)
	if err != nil {
		t.Fatal("Error occured but should not have.", err)
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a path, which is invalid.
	err = DefaultSQLiteConfig("")
	if err == nil {
		t.Fatal("Error did not occur but should have because of invalid path.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Make sure this doesn't work if db is connected.
	config.connection = &sqlx.DB{}
	err = DefaultSQLiteConfig(p)
	if err != ErrConnected {
		t.Fatal("Error about already connected db should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestNewMySQLConfig(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Provide connection data and make sure it was saved properly.
	host := "10.0.0.1"
	port := uint(3306)
	dbName := "n"
	dbUser := "u"
	dbPass := "p"
	c, err := NewMySQLConfig(host, port, dbName, dbUser, dbPass)
	if err != nil {
		t.Fatal("Error occured but should not have.", err)
		return
	}

	if c.Host != host {
		t.Fatal("Host not saved correctly.")
		return
	}
	if c.Port != port {
		t.Fatal("Port not saved correctly.")
		return
	}
	if c.Name != dbName {
		t.Fatal("Name not saved correctly.")
		return
	}
	if c.User != dbUser {
		t.Fatal("User not saved correctly.")
		return
	}
	if c.Password != dbPass {
		t.Fatal("Password not saved correctly.")
		return
	}
	if c.Type != DBTypeMySQL {
		t.Fatal("DB type not set properly.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a host, which is invalid.
	_, err = NewMySQLConfig("", port, dbName, dbUser, dbPass)
	if err != ErrHostNotProvided {
		t.Fatal("Error did not occur but should have because a host wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a port, which is invalid.
	_, err = NewMySQLConfig(host, 0, dbName, dbUser, dbPass)
	if err != ErrInvalidPort {
		t.Fatal("Error did not occur but should have because a port wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a port that is too high, which is invalid.
	_, err = NewMySQLConfig(host, 100000, dbName, dbUser, dbPass)
	if err != ErrInvalidPort {
		t.Fatal("Error did not occur but should have because of invalid port (too high).")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a db name, which is invalid.
	_, err = NewMySQLConfig(host, port, "", dbUser, dbPass)
	if err != ErrNameNotProvided {
		t.Fatal("Error did not occur but should have because a db name wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a db user, which is invalid.
	_, err = NewMySQLConfig(host, port, dbName, "", dbPass)
	if err != ErrUserNotProvided {
		t.Fatal("Error did not occur but should have because of a db user wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a db user, which is invalid.
	_, err = NewMySQLConfig(host, port, dbName, dbUser, "")
	if err != ErrPasswordNotProvided {
		t.Fatal("Error did not occur but should have because a db user's password wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestDefaultMySQLConfig(t *testing.T) {
	config.connection = nil

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Provide connection data and make sure it was saved properly.
	host := "10.0.0.1"
	port := uint(3306)
	dbName := "n"
	dbUser := "u"
	dbPass := "p"
	err := DefaultMySQLConfig(host, port, dbName, dbUser, dbPass)
	if err != nil {
		t.Fatal("Error occured but should not have.", err)
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a host, which is invalid.
	err = DefaultMySQLConfig("", port, dbName, dbUser, dbPass)
	if err != ErrHostNotProvided {
		t.Fatal("Error did not occur but should have because a host wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Make sure this doesn't work if db is connected.
	config.connection = &sqlx.DB{}
	err = DefaultMySQLConfig(host, port, dbName, dbUser, dbPass)
	if err != ErrConnected {
		t.Fatal("Error about already connected db should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestNewMariaDBConfig(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Provide connection data and make sure it was saved properly.
	host := "10.0.0.1"
	port := uint(3306)
	dbName := "n"
	dbUser := "u"
	dbPass := "p"
	c, err := NewMariaDBConfig(host, port, dbName, dbUser, dbPass)
	if err != nil {
		t.Fatal("Error occured but should not have.", err)
		return
	}

	if c.Host != host {
		t.Fatal("Host not saved correctly.")
		return
	}
	if c.Port != port {
		t.Fatal("Port not saved correctly.")
		return
	}
	if c.Name != dbName {
		t.Fatal("Name not saved correctly.")
		return
	}
	if c.User != dbUser {
		t.Fatal("User not saved correctly.")
		return
	}
	if c.Password != dbPass {
		t.Fatal("Password not saved correctly.")
		return
	}
	if c.Type != DBTypeMariaDB {
		t.Fatal("DB type not set properly.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a host, which is invalid.
	_, err = NewMariaDBConfig("", port, dbName, dbUser, dbPass)
	if err != ErrHostNotProvided {
		t.Fatal("Error did not occur but should have because a host wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a port, which is invalid.
	_, err = NewMariaDBConfig(host, 0, dbName, dbUser, dbPass)
	if err != ErrInvalidPort {
		t.Fatal("Error did not occur but should have because a port wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a port that is too high, which is invalid.
	_, err = NewMariaDBConfig(host, 100000, dbName, dbUser, dbPass)
	if err != ErrInvalidPort {
		t.Fatal("Error did not occur but should have because of invalid port (too high).")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a db name, which is invalid.
	_, err = NewMariaDBConfig(host, port, "", dbUser, dbPass)
	if err != ErrNameNotProvided {
		t.Fatal("Error did not occur but should have because a db name wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a db user, which is invalid.
	_, err = NewMariaDBConfig(host, port, dbName, "", dbPass)
	if err != ErrUserNotProvided {
		t.Fatal("Error did not occur but should have because of a db user wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a db user, which is invalid.
	_, err = NewMariaDBConfig(host, port, dbName, dbUser, "")
	if err != ErrPasswordNotProvided {
		t.Fatal("Error did not occur but should have because a db user's password wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestDefaultMariaDBConfig(t *testing.T) {
	config.connection = nil

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Provide connection data and make sure it was saved properly.
	host := "10.0.0.1"
	port := uint(3306)
	dbName := "n"
	dbUser := "u"
	dbPass := "p"
	err := DefaultMariaDBConfig(host, port, dbName, dbUser, dbPass)
	if err != nil {
		t.Fatal("Error occured but should not have.", err)
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a host, which is invalid.
	err = DefaultMariaDBConfig("", port, dbName, dbUser, dbPass)
	if err != ErrHostNotProvided {
		t.Fatal("Error did not occur but should have because a host wasn't provided.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Make sure this doesn't work if db is connected.
	config.connection = &sqlx.DB{}
	err = DefaultMariaDBConfig(host, port, dbName, dbUser, dbPass)
	if err != ErrConnected {
		t.Fatal("Error about already connected db should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestValidate(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Use invalid db type.
	c := Config{
		Type: "test",
	}
	err := c.validate()
	if err != ErrInvalidDBType {
		t.Fatal("Error about invalid db type should have been kicked out but wasn't.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Test if default sqlite journal mode is set if it is not provided.
	c = Config{
		Type:       DBTypeSQLite,
		SQLitePath: "/path/to/sqlite.db",
	}
	err = c.validate()
	if err != nil {
		t.Fatal("error occured but should not have", err)
		return
	}
	if c.SQLitePragmaJournalMode != defaultSQLiteJournalMode {
		t.Fatal("default sqlite journal mode wasn't set")
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Set an invalid journal mode.
	c.SQLitePragmaJournalMode = "invalid"
	err = c.validate()
	if err != ErrInvalidJournalMode {
		t.Fatal("error about invalid journal mode was not kicked out as expected")
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

}

func TestBuildConnectionString(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//sqlite with default journal mode
	p := "/path/to/file.db"
	c, err := NewSQLiteConfig(p)
	if err != nil {
		t.Fatal("error occured but should not have")
		return
	}

	connString := c.buildConnectionString(false)
	if !strings.Contains(connString, p) {
		t.Fatal("connection string didn't include path to sqlite file as expected")
		return
	}

	if !strings.Contains(connString, "_journal_mode=DELETE") {
		t.Fatal("journal mode query param not set as expected")
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//sqlite with write ahead log
	c, err = NewSQLiteConfig(p)
	if err != nil {
		t.Fatal("error occured but should not have")
		return
	}
	c.SQLitePragmaJournalMode = SQLiteJournalModeWAL

	connString = c.buildConnectionString(false)
	if !strings.Contains(connString, p) {
		t.Fatal("connection string didn't include path to sqlite file as expected")
		return
	}

	if !strings.Contains(connString, "_journal_mode=WAL") {
		t.Fatal("journal mode query param not set as expected")
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//mysql or mariadb
	host := "10.0.0.1"
	port := uint(3306)
	name := "name"
	user := "user"
	pass := "pass"
	c, err = NewMySQLConfig(host, port, name, user, pass)
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}
