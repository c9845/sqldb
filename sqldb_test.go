package sqldb

import (
	"strconv"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	c, err := NewConfig(DBTypeMariaDB)
	if err != nil {
		t.Fatal(err)
		return
	}
	if c == nil {
		t.Fatal("No config returned")
		return
	}
	if c.Type != DBTypeMariaDB {
		t.Fatal("Config doesn't match")
		return
	}

	//test with bad db type
	c, err = NewConfig("maybe")
	if err == nil {
		t.Fatal("Error about invalid db type should have occured.")
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
	c, err := NewConfig(DBTypeMariaDB)
	if err != nil {
		t.Fatal(err)
		return
	}
	err = c.validate()
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
	c, err = NewConfig(DBTypeSQLite)
	if err != nil {
		t.Fatal(err)
		return
	}
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

	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Bad db type, which should never occur.
	c, err = NewConfig(DBTypeMySQL)
	if err != nil {
		t.Fatal(err)
		return
	}

	c.Type = "ppop" //setting to a string which gets autocorrected to the dbType type even though the value is invalid.
	err = c.validate()
	if err == nil {
		t.Fatal("Error about bad db type should have occured in validate.")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestBuildConnectionString(t *testing.T) {
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//For deploying mysql/mariadb.
	c, err := NewConfig(DBTypeMySQL)
	if err != nil {
		t.Fatal(err)
		return
	}
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
	c, err = NewConfig(DBTypeMySQL)
	if err != nil {
		t.Fatal(err)
		return
	}
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
	c, err = NewConfig(DBTypeSQLite)
	if err != nil {
		t.Fatal(err)
		return
	}
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
	if err == nil {
		t.Fatal("error about bad db type should have been returned")
		return
	}
}

func TestIsMySQLOrMariaDB(t *testing.T) {
	c, err := NewConfig(DBTypeMySQL)
	if err != nil {
		t.Fatal(err)
		return
	}
	if !c.IsMySQLOrMariaDB() {
		t.Fatal("Is mysql")
		return
	}

	c, err = NewConfig(DBTypeMariaDB)
	if err != nil {
		t.Fatal(err)
		return
	}
	if !c.IsMySQLOrMariaDB() {
		t.Fatal("Is mariadb")
		return
	}

	c, err = NewConfig(DBTypeSQLite)
	if err != nil {
		t.Fatal(err)
		return
	}
	if c.IsMySQLOrMariaDB() {
		t.Fatal("Is sqlite")
		return
	}
}

func TestIsMySQL(t *testing.T) {
	c, err := NewConfig(DBTypeMySQL)
	if err != nil {
		t.Fatal(err)
		return
	}
	if !c.IsMySQL() {
		t.Fatal("is mysql")
		return
	}
}

func TestIsMariaDB(t *testing.T) {
	c, err := NewConfig(DBTypeMariaDB)
	if err != nil {
		t.Fatal(err)
		return
	}
	if !c.IsMariaDB() {
		t.Fatal("is mariadb")
		return
	}
}

func TestIsSQLite(t *testing.T) {
	c, err := NewConfig(DBTypeSQLite)
	if err != nil {
		t.Fatal(err)
		return
	}
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

func TestConnect(t *testing.T) {
	//Test with sqlite.
	//No test with mariadb/mysql b/c we probably don't have a db server accessible.
	c := NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}
	if c.Type != DBTypeSQLite {
		t.Fatal("Config doesn't match")
		return
	}

	err := c.Connect()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()
}

func TestClose(t *testing.T) {
	//Test with sqlite.
	//No test with mariadb/mysql b/c we probably don't have a db server accessible.
	c := NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}
	if c.Type != DBTypeSQLite {
		t.Fatal("Config doesn't match")
		return
	}

	err := c.Connect()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = c.Close()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = c.connection.Ping()
	if err == nil {
		t.Fatal("Connection isn't really closed")
		return
	}

	//call close multiple times to make sure there isn't anything odd that happens
	//if a user does this.
	err = c.Close()
	if err != nil {
		t.Fatal(err)
		return
	}

	err = c.Close()
	if err != nil {
		t.Fatal(err)
		return
	}
}

func TestConnected(t *testing.T) {
	//Test with sqlite.
	//No test with mariadb/mysql b/c we probably don't have a db server accessible.
	c := NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}
	if c.Type != DBTypeSQLite {
		t.Fatal("Config doesn't match")
		return
	}

	//test before connecting, should be false
	if c.Connected() {
		t.Fatal("Connected should be false, never connected")
		return
	}

	//connect and test, should be true
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
		return
	}

	if !c.Connected() {
		t.Fatal("Connected should be true")
		return
	}

	//disconnect and test, should be false
	err = c.Close()
	if err != nil {
		t.Fatal(err)
		return
	}

	if c.Connected() {
		t.Fatal("Connected should be false")
		return
	}

	//connect again, should be true
	err = c.Connect()
	if err != nil {
		t.Fatal(err)
		return
	}

	if !c.Connected() {
		t.Fatal("Connected should be true")
		return
	}

	err = c.Close()
	if err != nil {
		t.Fatal(err)
		return
	}
}

func TestDeploySchema(t *testing.T) {
	//Test with sqlite in-memory db.
	//No test with mariadb/mysql b/c we probably don't have a db server accessible.
	c := NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}

	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err := c.DeploySchemaWithOps(DeploySchemaOptions{false, false})
	if err != nil {
		t.Fatal(err)
		return
	}

	//Try inserting
	insert := `INSERT INTO users (Username) VALUES (?)`
	_, err = c.connection.Exec(insert, "username@example.com")
	if err != nil {
		t.Fatal(err)
		return
	}

	//Close connection
	c.Close()
}

func TestDeploySchemaAndClose(t *testing.T) {
	//Test with sqlite in-memory db.
	//No test with mariadb/mysql b/c we probably don't have a db server accessible.
	c := NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}

	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err := c.DeploySchemaWithOps(DeploySchemaOptions{false, true})
	if err != nil {
		t.Fatal(err)
		return
	}

	//Make sure connection is closed.
	if c.Connected() {
		t.Fatal("Connection should be closed")
	}

	//Close connection
	c.Close()
}

func TestUpdateSchema(t *testing.T) {
	//Test with sqlite in-memory db.
	//No test with mariadb/mysql b/c we probably don't have a db server accessible.
	c := NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}

	//Need to deploy schema first.
	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err := c.DeploySchemaWithOps(DeploySchemaOptions{false, false})
	if err != nil {
		t.Fatal(err)
		return
	}

	//Update
	updateTable := `ALTER TABLE users ADD COLUMN FirstName TEXT`
	c.UpdateQueries = []string{updateTable}

	err = c.UpdateSchemaWithOps(UpdateSchemaOptions{false})
	if err != nil {
		t.Fatal(err)
		return
	}

	//Insert into new column to make sure it was created.
	insert := `INSERT INTO users (Username, FirstName) VALUES (?, ?)`
	_, err = c.connection.Exec(insert, "username@example.com", "john")
	if err != nil {
		t.Fatal(err)
		return
	}

	//Close connection
	c.Close()
}

func TestUpdateSchemaAndClose(t *testing.T) {
	//Test with sqlite in-memory db.
	//No test with mariadb/mysql b/c we probably don't have a db server accessible.
	c := NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}

	//Need to deploy schema first.
	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err := c.DeploySchemaWithOps(DeploySchemaOptions{false, false})
	if err != nil {
		t.Fatal(err)
		return
	}

	//Update
	updateTable := `ALTER TABLE users ADD COLUMN FirstName TEXT`
	c.UpdateQueries = []string{updateTable}

	err = c.UpdateSchema()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Make sure connection is closed.
	if c.Connected() {
		t.Fatal("Connection should be closed")
	}
}
