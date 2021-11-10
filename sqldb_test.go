package sqldb

import (
	"strconv"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig(DBTypeMariaDB)
	if c == nil {
		t.Fatal("No config returned")
		return
	}
	if c.Type != DBTypeMariaDB {
		t.Fatal("Config doens't match")
		return
	}
}

func TestNewSQLiteConfig(t *testing.T) {
	p := "/fake/path/to/sqlite.db"
	c := NewSQLiteConfig(p)
	if c.SQLitePath != p {
		t.Fatal("Path not saved correctly.")
		return
	}
	if c.Type != DBTypeSQLite {
		t.Fatal("DB type not set properly.")
		return
	}
}

func TestNewMySQLConfig(t *testing.T) {
	host := "10.0.0.1"
	port := uint(3306)
	dbName := "n"
	dbUser := "u"
	dbPass := "p"
	c := NewMySQLConfig(host, port, dbName, dbUser, dbPass)
	if c.Type != DBTypeMySQL {
		t.Fatal("type not set correctly")
		return
	}
	if c.Host != host {
		t.Fatal("host not saved")
		return
	}
	if c.Port != port {
		t.Fatal("port not saved")
		return
	}
	if c.Name != dbName {
		t.Fatal("name not saved")
		return
	}
	if c.User != dbUser {
		t.Fatal("user not saved")
		return
	}
	if c.Password != dbPass {
		t.Fatal("password not saved")
		return
	}
}

func TestNewMariaDBConfig(t *testing.T) {
	host := "10.0.0.1"
	port := uint(3306)
	dbName := "n"
	dbUser := "u"
	dbPass := "p"
	c := NewMariaDBConfig(host, port, dbName, dbUser, dbPass)
	if c.Type != DBTypeMariaDB {
		t.Fatal("type not set correctly")
		return
	}
	if c.Host != host {
		t.Fatal("host not saved")
		return
	}
	if c.Port != port {
		t.Fatal("port not saved")
		return
	}
	if c.Name != dbName {
		t.Fatal("name not saved")
		return
	}
	if c.User != dbUser {
		t.Fatal("user not saved")
		return
	}
	if c.Password != dbPass {
		t.Fatal("password not saved")
		return
	}
}

func TestValidate(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Test MariaDB/MySQL with missing stuff.
	c := NewConfig(DBTypeMariaDB)
	err := c.validate()
	if err != ErrHostNotProvided {
		t.Fatal("ErrHostNotProvided should have occured but didnt")
		return
	}

	c.Host = "10.0.0.1"
	err = c.validate()
	if err != ErrInvalidPort {
		t.Fatal("ErrInvalidPort should have occured but didnt")
		return
	}

	c.Port = 3306
	err = c.validate()
	if err != ErrNameNotProvided {
		t.Fatal("ErrNameNotProvided should have occured but didnt")
		return
	}

	c.Name = "dbname"
	err = c.validate()
	if err != ErrUserNotProvided {
		t.Fatal("ErrUserNotProvided should have occured but didnt")
		return
	}

	c.User = "user"
	err = c.validate()
	if err != ErrPasswordNotProvided {
		t.Fatal("ErrPasswordNotProvided should have occured but didnt")
		return
	}

	c.Password = "password"
	err = c.validate()
	if err != nil {
		t.Fatal("Unexpected error", err)
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Test for SQLite.
	c = NewConfig(DBTypeSQLite)
	err = c.validate()
	if err != ErrSQLitePathNotProvided {
		t.Fatal("ErrSQLitePathNotProvided should have occured but didnt")
		return
	}

	c.SQLitePath = "/path/to/sqlite.db"
	err = c.validate()
	if err != nil {
		t.Fatal("unexpected error", err)
		return
	}

	c.SQLitePragmaJournalMode = "bad"
	err = c.validate()
	if err != ErrInvalidJournalMode {
		t.Fatal("ErrInvalidJournalMode should have occured but didnt")
		return
	}

	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Bad db type, which should never occur.
	c = NewConfig("bad-db-type")
	err = c.validate()
	if err != ErrInvalidDBType {
		t.Fatal("ErrInvalidDBType should have occured but didnt")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestIsJournalModeValid(t *testing.T) {
	m := defaultSQLiteJournalMode
	yes := isJournalModeValid(m, validJournalModes)
	if !yes {
		t.Fatal("Journal mode should be valid")
		return
	}

	m = "bad"
	yes = isJournalModeValid(m, validJournalModes)
	if yes {
		t.Fatal("Journal mode should NOT be valid")
		return
	}
}

func TestBuildConnectionString(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//For deploying mysql/mariadb.
	c := NewConfig(DBTypeMySQL)
	c.Host = "10.0.0.1"
	c.Port = 3306
	c.User = "user"
	c.Password = "password"

	connString := c.buildConnectionString(true)
	manuallyBuilt := c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/"
	if connString != manuallyBuilt {
		t.Fatal("Conn string not built correctly", connString, manuallyBuilt)
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//For existing mysql/mariadb.
	c = NewConfig(DBTypeMySQL)
	c.Host = "10.0.0.1"
	c.Port = 3306
	c.User = "user"
	c.Password = "password"
	c.Name = "database"

	connString = c.buildConnectionString(false)
	manuallyBuilt = c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/" + c.Name
	if connString != manuallyBuilt {
		t.Fatal("Conn string not built correctly", connString, manuallyBuilt)
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//For SQLite.
	c = NewConfig(DBTypeSQLite)
	c.SQLitePath = "/path/to/sqlite.db"
	connString = c.buildConnectionString(false)
	if connString != c.SQLitePath {
		t.Fatal("Conn string for SQLite should just be the path but wasn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestGetDriver(t *testing.T) {
	driver, err := getDriver(DBTypeMariaDB)
	if err != nil {
		t.Fatal("error occured but should not have", err)
		return
	}
	if driver != "mysql" {
		t.Fatal("Invalid driver returned for mysql")
		return
	}

	driver, err = getDriver(DBTypeSQLite)
	if err != nil {
		t.Fatal("error occured but should not have", err)
		return
	}
	if driver != "sqlite3" {
		t.Fatal("Invalid driver returned for sqlite")
		return
	}

	_, err = getDriver("bad")
	if err != ErrInvalidDBType {
		t.Fatal("ErrInvalidDBType should have occured")
		return
	}
}

func TestIsMySQLOrMariaDB(t *testing.T) {
	c := NewConfig(DBTypeMySQL)
	if !c.IsMySQLOrMariaDB() {
		t.Fatal("Is mysql")
		return
	}

	c = NewConfig(DBTypeMariaDB)
	if !c.IsMySQLOrMariaDB() {
		t.Fatal("Is mariadb")
		return
	}

	c = NewConfig(DBTypeSQLite)
	if c.IsMySQLOrMariaDB() {
		t.Fatal("Is sqlite")
		return
	}
}

func TestIsMySQL(t *testing.T) {
	c := NewConfig(DBTypeMySQL)
	if !c.IsMySQL() {
		t.Fatal("is mysql")
		return
	}
}

func TestIsMariaDB(t *testing.T) {
	c := NewConfig(DBTypeMariaDB)
	if !c.IsMariaDB() {
		t.Fatal("is mariadb")
		return
	}
}

func TestIsSQLite(t *testing.T) {
	c := NewConfig(DBTypeSQLite)
	if !c.IsSQLite() {
		t.Fatal("is sqlite")
		return
	}
}

func TestDefaultMapperFunc(t *testing.T) {
	in := "string"
	out := DefaultMapperFunc(in)
	if in != out {
		t.Fatal("defaultMapperFunc modified string but should not have.")
		return
	}
}

func TestDefaults(t *testing.T) {
	//set config
	path := "/path/to/sqlite.db"
	DefaultSQLiteConfig(path)

	//get config
	c := GetDefaultConfig()
	if c.SQLitePath != path {
		t.Fatal("default path not saved correctly")
		return
	}
	if c.Type != DBTypeSQLite {
		t.Fatal("default type not saved correctly")
		return
	}

	//modify
	f := func(in string) (out string) {
		out = strings.ToUpper(in)
		return
	}
	MapperFunc(f)
	if config.MapperFunc("in") != "IN" {
		t.Fatal("MapperFunc not set correctly")
		return
	}

	f2 := func(in string) (out string) {
		out = in
		return
	}
	TranslateCreateTableFuncs([]TranslateFunc{f2})
	if len(config.TranslateCreateTableFuncs) != 1 {
		t.Fatal("translate create table func not added")
		return
	}

	q := `
		CREATE TABLE IF NOT EXISTS users (
			ID INT NOT NULL AUTO_INCREMENT,
			Username VARCHAR(255) NOT NULL,
			PRIMARY KEY(ID)
		)
	`
	SetDeployQueries([]string{q})
	if len(config.DeployQueries) != 1 {
		t.Fatal("deploy queries not added")
		return
	}

	u := "ALTER TABLE users ADD COLUMN Fname VARCHAR(255) NOT NULL DEFAULT ''"
	SetUpdateQueries([]string{u})
	if len(config.UpdateQueries) != 1 {
		t.Fatal("update queries not added")
		return
	}

}
