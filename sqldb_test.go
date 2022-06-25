package sqldb

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestValid(t *testing.T) {
	if err := DBTypeMySQL.valid(); err != nil {
		t.Fatal("DBTypeMySQL is valid, error should not have occured.", err)
		return
	}

	bad := DBType("bad")
	if err := bad.valid(); err == nil {
		t.Fatal("bad DBType is not valid but no error was returned")
		return
	}
}

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
}

func TestBuildConnectionString(t *testing.T) {
	//For deploying mysql/mariadb.
	c := NewMariaDBConfig("10.0.01", uint(3306), "", "user", "password")
	connString := c.buildConnectionString(true)
	manuallyBuilt := c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/"
	if connString != manuallyBuilt {
		t.Fatal("Connection string not built correctly.", connString, manuallyBuilt)
		return
	}

	//For connecting to already deployted mysql/mariadb.
	c = NewMariaDBConfig("10.0.01", uint(3306), "db-name", "user", "password")
	connString = c.buildConnectionString(false)
	manuallyBuilt = c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/" + c.Name
	if connString != manuallyBuilt {
		t.Fatal("Connection string not built correctly.", connString, manuallyBuilt)
		return
	}

	//For deploying SQLite.
	c = NewSQLiteConfig("/path/to/sqlite.db")
	connString = c.buildConnectionString(true)
	if connString != c.SQLitePath {
		t.Fatal("Connection string for SQLite should just be the path but wasn't.", connString)
		return
	}

	//For connecting to already deployed SQLite.
	c = NewSQLiteConfig("/path/to/sqlite.db")
	connString = c.buildConnectionString(false)
	if connString != c.SQLitePath {
		t.Fatal("Connection string for SQLite should just be the path but wasn't.", connString)
		return
	}

	//Test SQLite with additional PRAGMAs
	c.SQLitePragmas = []string{"PRAGMA busy_timeout = 5000"}
	connString = c.buildConnectionString(false)
	if !strings.Contains(connString, c.SQLitePath) {
		t.Fatal("Connection string for SQLite should include the path but didn't.", connString)
		return
	}
	if !strings.Contains(connString, "busy_timeout") {
		t.Fatal("PRAGMAs not added to connection string as expected.", connString)
		return
	}
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

	//Try connecting again which should fail.
	err = c.Connect()
	if err != ErrConnected {
		t.Fatal("Error about connecting to an already connected database should have occured.")
		return
	}

	//Test with a bad config.
	c.Close()
	c.SQLitePath = ""
	err = c.Connect()
	if err == nil {
		t.Fatal("Error about bad config should have occured.")
		return
	}

	//Test with a bad db type.
	c.Type = DBType("bad")
	c.SQLitePath = InMemoryFilePathRacy
	err = c.Connect()
	if err == nil {
		t.Fatal("Error about bad db type should have occured.")
		return
	}

	//Test with a good config and pragma set, check pragmas are set.
	c.Close()
	c = NewSQLiteConfig(InMemoryFilePathRacy)
	c.SQLitePragmas = []string{
		"PRAGMA busy_timeout = 5000",
		//cannot set journal mode to WAL with in-memory db. journal mode is memory.
	}

	err = c.Connect()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()

	var busyTimeout string
	q := "PRAGMA busy_timeout"
	err = c.Connection().Get(&busyTimeout, q)
	if err != nil {
		t.Fatal(err)
		return
	}
	if strings.ToLower(busyTimeout) != strings.ToLower("5000") {
		t.Fatal("PRAGMA busy_timeout not set correctly.", busyTimeout)
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

	insertInitial := func(c *sqlx.DB) error {
		q := "INSERT INTO users (Username) VALUES (?)"
		_, err := c.Exec(q, "initialuser@example.com")
		return err
	}
	c.DeployFuncs = []DeployFunc{insertInitial}

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

	//Make sure results were inserted (DeployFunc and extra query).
	q := "SELECT Count(ID) FROM Users"
	var count int64
	err = c.Connection().Get(&count, q)
	if err != nil {
		t.Fatal("Could not query.")
		return
	} else if count != 2 {
		t.Fatal("Data not inserted correctly.", count)
		return
	}

	//Try deploying to an already connected to db. This should fail.
	err = c.DeploySchemaWithOps(DeploySchemaOptions{false, false})
	if err != ErrConnected {
		t.Fatal("Error about db already connected should have occured.")
		return
	}

	//Close connection
	c.Close()

	//Try deploying with an invalid config.
	c.SQLitePath = ""
	err = c.DeploySchemaWithOps(DeploySchemaOptions{false, false})
	if err == nil {
		t.Fatal("Error about invalid config should have occured.")
		return
	}

	//Try deploying with a bad deploy func.
	c.SQLitePath = InMemoryFilePathRacy
	insertInitial = func(c *sqlx.DB) error {
		q := "SELECT INTO users VALUES (?)"
		_, err := c.Exec(q, "initialuser@example.com")
		return err
	}
	c.DeployFuncs = []DeployFunc{insertInitial}

	err = c.DeploySchemaWithOps(DeploySchemaOptions{false, false})
	if err == nil {
		t.Fatal("Error should have occured because of bad deploy func.")
		return
	}
	if c.Connected() {
		t.Fatal("Connection should be closed after deploy func error.")
		return
	}
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

	uf := func(tx *sqlx.Tx) error {
		q := "ALTER TABLE users ADD COLUMN LastName TEXT"
		_, err := tx.Exec(q)
		return err
	}
	c.UpdateFuncs = []UpdateFunc{uf}

	err = c.UpdateSchemaWithOps(UpdateSchemaOptions{false})
	if err != nil {
		t.Fatal(err)
		return
	}

	//Insert into new columns to make sure they were created.
	insert := `INSERT INTO users (Username, FirstName) VALUES (?, ?)`
	_, err = c.connection.Exec(insert, "username@example.com", "john")
	if err != nil {
		t.Fatal(err)
		return
	}

	insert = `INSERT INTO users (Username, FirstName, LastName) VALUES (?, ?, ?)`
	_, err = c.connection.Exec(insert, "username@example.com", "john", "doe")
	if err != nil {
		t.Fatal(err)
		return
	}

	//Try updating with an invalid config.
	c.SQLitePath = ""
	err = c.UpdateSchemaWithOps(UpdateSchemaOptions{false})
	if err == nil {
		t.Fatal("Error about invalid config should have occured.")
		return
	}

	//Try updating a db that is not already connected.
	//Note error checking b/c db has not been deployed. If we deploy and close the
	//db connection, the in-memory db is gone, therefore we just with a blank db and
	//the table does not exist yet.
	c.Close()
	c.SQLitePath = InMemoryFilePathRacy
	err = c.UpdateSchemaWithOps(UpdateSchemaOptions{false})
	if err != nil && !strings.Contains(err.Error(), "no such table") {
		t.Fatal(err)
		return
	}

	//Test with a bad update func.
	c.Close()
	c = NewSQLiteConfig(InMemoryFilePathRacy)
	if c == nil {
		t.Fatal("No config returned")
		return
	}

	createTable = `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err = c.DeploySchemaWithOps(DeploySchemaOptions{false, false})
	if err != nil {
		t.Fatal(err)
		return
	}

	uf = func(tx *sqlx.Tx) error {
		q := "ALTER ELBAT dynamite ADD COLUMN LastName TEXT"
		_, err := tx.Exec(q)
		return err
	}
	c.UpdateFuncs = []UpdateFunc{uf}

	err = c.UpdateSchemaWithOps(UpdateSchemaOptions{false})
	if err == nil {
		t.Fatal("Error about bad update func should have occured.")
		return
	}
	if c.Connected() {
		t.Fatal("Connection should be closed after bad update func.")
		return
	}
}

func TestGetSQLiteVersion(t *testing.T) {
	v, err := GetSQLiteVersion()
	if err != nil {
		t.Fatal(err)
		return
	}
	if v == "" {
		t.Fatal("SQLite version not returned.", v)
		return
	}
}

func TestBuildColumnString(t *testing.T) {
	//Good state for inserting.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, valString, err := c.buildColumnString(false)
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID,Fname,Bday" {
		t.Fatal("colString not build right", colString)
		return
	}
	if valString != "?,?,?" {
		t.Fatal("valString not build right", valString)
		return
	}

	//Good state for updating.
	colString, valString, err = c.buildColumnString(true)
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID=?,Fname=?,Bday=?" {
		t.Fatal("colString not build right", colString)
		return
	}
	if valString != "" {
		t.Fatal("valString should be blank", valString)
		return
	}

	//Provide extra comma in column.
	c = append(c, "Lname,")
	colString, _, err = c.buildColumnString(false)
	if err != ErrExtraCommaInColumnString {
		t.Fatal("Error about double comma should have occured.", colString)
		return
	}

	//Test with no columns
	c = Columns{}
	_, _, err = c.buildColumnString(false)
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestForSelect(t *testing.T) {
	//Good test.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, err := c.ForSelect()
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID,Fname,Bday" {
		t.Fatal("colString not build right", colString)
		return
	}

	//No columns bad test.
	c = Columns{}
	_, err = c.ForSelect()
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestForInsert(t *testing.T) {
	//Good test.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, valString, err := c.ForInsert()
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID,Fname,Bday" {
		t.Fatal("colString not build right", colString)
		return
	}
	if valString != "?,?,?" {
		t.Fatal("valString not build right", valString)
		return
	}

	//No columns bad test.
	c = Columns{}
	_, _, err = c.ForInsert()
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestForUpdate(t *testing.T) {
	//Good test.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, err := c.ForUpdate()
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID=?,Fname=?,Bday=?" {
		t.Fatal("colString not build right", colString)
		return
	}

	//No columns bad test.
	c = Columns{}
	_, err = c.ForUpdate()
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestString(t *testing.T) {
	s := "hello"
	w := Where(s)
	if w.String() != s {
		t.Fatal("Mismatch", w.String(), s)
		return
	}
}

func TestBuildPragmaString(t *testing.T) {
	p := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}

	built := buildPragmaString(p)

	u, err := url.ParseQuery(built)
	if err != nil {
		t.Fatal(err)
		return
	}
	if len(u) != 2 {
		t.Fatal("PRAGMAs not built correctly", len(u))
		return
	}

	switch GetSQLiteLibrary() {
	case sqliteLibraryMattn:
		shouldBe := "_journal_mode=wal&_busy_timeout=5000"
		parsed, err := url.ParseQuery(shouldBe)
		if err != nil {
			t.Fatal(err)
			return
		}

		enc := "?" + parsed.Encode()
		if built != enc {
			t.Fatal("Incorrect PRAGMAs for connection string.", built, enc)
			return
		}

	case sqliteLibraryModernc:
		shouldBe := "_pragma=journal_mode=wal&_pragma=busy_timeout=5000"
		parsed, err := url.ParseQuery(shouldBe)
		if err != nil {
			t.Fatal(err)
			return
		}

		enc := "?" + parsed.Encode()
		if built != enc {
			t.Fatal("Incorrect PRAGMAs for connection string.", built, enc)
			return
		}
	}
}
