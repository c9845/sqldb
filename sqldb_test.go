package sqldb

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestNew(t *testing.T) {
	c := New()
	if len(c.SQLitePragmas) != len(SQLiteDefaultPragmas) {
		t.FailNow()
		return
	}
	if c.MapperFunc == nil {
		t.FailNow()
		return
	}
	if c.LoggingLevel != LogLevelDefault {
		t.FailNow()
		return
	}
	if c.ConnectionOptions == nil {
		t.FailNow()
		return
	}
}

func TestNewMariaDB(t *testing.T) {
	c := NewMariaDB("10.0.0.1", "db_name", "user1", "password1")
	if c.Type != DBTypeMariaDB {
		t.FailNow()
		return
	}
}

func TestNewMySQL(t *testing.T) {
	c := NewMySQL("10.0.0.1", "db_name", "user1", "password1")
	if c.Type != DBTypeMySQL {
		t.FailNow()
		return
	}
}

func TestNewSQLite(t *testing.T) {
	c := NewSQLite("/path/to/sqlite.db")
	if c.Type != DBTypeSQLite {
		t.FailNow()
		return
	}
}

func TestNewMSSQL(t *testing.T) {
	c := NewMSSQL("10.0.0.1", "db_name", "user1", "password1")
	if c.Type != DBTypeMSSQL {
		t.FailNow()
		return
	}
}

func TestUse(t *testing.T) {
	c := New()
	c.Host = "10.0.0.1"
	c.Name = "test"

	Use(c)

	if cfg.Host != c.Host {
		t.FailNow()
		return
	}
	if cfg.Name != c.Name {
		t.FailNow()
		return
	}
}

func TestValidate(t *testing.T) {
	//Test MariaDB/MySQL with missing stuff.
	c := New()
	c.Type = DBTypeMariaDB

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

	c.Port = defaultMariaDBPort
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
	c = New()
	c.Type = DBTypeSQLite

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
	c = New()
	c.Type = "bad" //setting to a string which gets autocorrected to the dbType type even though the value is invalid.

	err = c.validate()
	if err == nil {
		t.Fatal("Error about bad db type should have occured in validate.")
		return
	}
}

func TestBuildConnectionString(t *testing.T) {
	//For deploying a MariaDB/MySQL database (note missing database name).
	c := NewMariaDB("10.0.0.1", "", "user", "password")
	connString := c.buildConnectionString(true)
	manuallyBuilt := c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/"
	if connString != manuallyBuilt {
		t.Fatal("Connection string not built correctly.", connString, manuallyBuilt)
		return
	}

	//For connecting to already deployed MariaDB/MySQL database.
	c = NewMariaDB("10.0.0.1", "db_name", "user", "password")
	connString = c.buildConnectionString(false)
	manuallyBuilt = c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/" + c.Name
	if connString != manuallyBuilt {
		t.Fatal("Connection string not built correctly.", connString, manuallyBuilt)
		return
	}

	//For deploying SQLite.
	c = NewSQLite("/path/to/sqlite.db")
	connString = c.buildConnectionString(true)
	if connString != c.SQLitePath {
		t.Fatal("Connection string for SQLite should just be the path but wasn't.", connString)
		return
	}

	//For connecting to already deployed SQLite.
	c = NewSQLite("/path/to/sqlite.db")
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

	//For deploying mssql.
	c = NewMSSQL("10.0.0.1", "", "user", "password")
	connString = c.buildConnectionString(true)
	manuallyBuilt = "sqlserver://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + "?database=" + c.Name
	if connString != manuallyBuilt {
		t.Fatal("Connection string not built correctly.", connString, manuallyBuilt)
		return
	}

	//Test MSSQL additional connection parameters.
	c.AddConnectionOption("encrypt", "false")
	connString = c.buildConnectionString(true)
	if !strings.Contains(connString, "encrypt=false") {
		t.Fatal("Connection option not added to connection string as expected.", connString)
		return
	}
}

func TestConnect(t *testing.T) {
	//Only test with SQLite since that is the only database type we can be assured
	//exists/is available on any device that attempts to run this func.
	c := NewSQLite(InMemoryFilePathRaceSafe)

	err := c.Connect()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer c.Close()

	//Try connecting again which will fail.
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
	c.SQLitePath = InMemoryFilePathRaceSafe
	err = c.Connect()
	if err == nil {
		t.Fatal("Error about bad db type should have occured.")
		return
	}

	//Test with a good config and PRAGMA set, check PRAGMA is set.
	c.Close()
	c = NewSQLite(InMemoryFilePathRaceSafe)
	c.SQLitePragmas = []string{
		"PRAGMA busy_timeout = 5000",
		//cannot set journal mode to WAL with in-memory db. journal mode is memory.
	}

	err = c.Connect()
	if err != nil {
		t.Log("con ", c.connectionString)
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
	expectedBusyTimeout := "5000"
	if busyTimeout != expectedBusyTimeout {
		t.Fatal("PRAGMA busy_timeout not set correctly.", busyTimeout, expectedBusyTimeout)
		return
	}

	//Check if we are connected.
	if !c.Connected() {
		t.Fatal("Connection not showing connected as it should!")
		return
	}
}

func TestGetDriver(t *testing.T) {
	d := getDriver(DBTypeMariaDB)
	if d != "mysql" {
		t.FailNow()
		return
	}

	d = getDriver(DBTypeSQLite)
	if d != sqliteDriverName {
		t.FailNow()
		return
	}

	d = getDriver(DBTypeMSSQL)
	if d != "mssql" {
		t.FailNow()
		return
	}
}

func TestIsMySQL(t *testing.T) {
	c := NewMySQL("10.0.0.1", "db_name", "user1", "password!")
	if !c.IsMySQL() {
		t.Fatal("DB type isn't detected as MySQL", c.Type)
		return
	}
}

func TestIsMariaDB(t *testing.T) {
	c := NewMariaDB("10.0.0.1", "db_name", "user1", "password!")
	if !c.IsMariaDB() {
		t.Fatal("DB type isn't detected as MariaDB", c.Type)
		return
	}
}

func TestIsSQLite(t *testing.T) {
	c := NewSQLite("/path/to/sqlite.db")
	if !c.IsSQLite() {
		t.Fatal("DB type isn't detected as SQLite", c.Type)
		return
	}
}

func TestDefaultMapperFunc(t *testing.T) {
	in := "asdfasdfasdf"
	out := DefaultMapperFunc(in)
	if in != out {
		t.Fatal("defaultMapperFunc modified provided string but should not have.")
		return
	}
}

func TestClose(t *testing.T) {
	c := NewSQLite(InMemoryFilePathRaceSafe)
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

func TestDeploySchema(t *testing.T) {
	c := NewSQLite(InMemoryFilePathRaceSafe)
	c.LoggingLevel = LogLevelDebug

	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	insertInitial := func(c *sqlx.DB) error {
		insert := "INSERT INTO users (Username) VALUES (?)"
		_, err := c.Exec(insert, "initialuser@example.com")
		return err
	}
	c.DeployFuncs = []QueryFunc{insertInitial}

	opts := &DeploySchemaOptions{
		CloseConnection: false,
	}
	err := c.DeploySchema(opts)
	if err != nil {
		t.Fatal(err)
		return
	}
	if !c.Connected() {
		t.Fatal("Connection should still be connected.")
		return
	}

	//Try inserting.
	insert2 := `INSERT INTO users (Username) VALUES (?)`
	_, err = c.connection.Exec(insert2, "username@example.com")
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

	//Try deploying to an already connected to db. This must fail.
	err = c.DeploySchema(opts)
	if err != ErrConnected {
		t.Fatal("Error about db already connected should have occured.")
		return
	}

	//Close connection
	c.Close()

	//Try deploying with an invalid config.
	c.SQLitePath = ""
	err = c.DeploySchema(opts)
	if err == nil {
		t.Fatal("Error about invalid config should have occured.")
		return
	}

	//Try deploying with a bad deploy func.
	c.SQLitePath = InMemoryFilePathRaceSafe
	insertInitial = func(c *sqlx.DB) error {
		q := "SELECT INTO users VALUES (?)"
		_, err := c.Exec(q, "initialuser@example.com")
		return err
	}
	c.DeployFuncs = []QueryFunc{insertInitial}

	err = c.DeploySchema(opts)
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
	c := NewSQLite(InMemoryFilePathRaceSafe)
	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err := c.DeploySchema(nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Make sure connection is connected.
	if c.Connected() {
		t.Fatal("Connection should be connected")
	}

	//Close connection
	c.Close()
}

func TestUpdateSchema(t *testing.T) {
	c := NewSQLite(InMemoryFilePathRaceSafe)

	//Need to deploy schema first.
	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	deployOpts := &DeploySchemaOptions{
		CloseConnection: false,
	}
	err := c.DeploySchema(deployOpts)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Update schema.
	updateTable := `ALTER TABLE users ADD COLUMN FirstName TEXT`
	c.UpdateQueries = []string{updateTable}

	uf := func(c *sqlx.DB) error {
		q := "ALTER TABLE users ADD COLUMN LastName TEXT"
		_, err := c.Exec(q)
		return err
	}
	c.UpdateFuncs = []QueryFunc{uf}

	updateOpts := &UpdateSchemaOptions{
		CloseConnection: false,
	}
	err = c.UpdateSchema(updateOpts)
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
	err = c.UpdateSchema(updateOpts)
	if err == nil {
		t.Fatal("Error about invalid config should have occured.")
		return
	}

	//Try updating a db that is not already connected.
	//Note error checking b/c db has not been deployed. If we deploy and close the
	//db connection, the in-memory db is gone.
	c.Close()
	c.SQLitePath = InMemoryFilePathRaceSafe
	err = c.UpdateSchema(updateOpts)
	if err != nil && !strings.Contains(err.Error(), "no such table") {
		t.Fatal(err)
		return
	}

	//Test with a bad update func.
	c.Close()
	c = NewSQLite(InMemoryFilePathRaceSafe)

	createTable = `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err = c.DeploySchema(deployOpts)
	if err != nil {
		t.Fatal(err)
		return
	}

	uf = func(c *sqlx.DB) error {
		q := "ALTER ELBAT dynamite ADD COLUMN LastName TEXT"
		_, err := c.Exec(q)
		return err
	}
	c.UpdateFuncs = []QueryFunc{uf}

	err = c.UpdateSchema(updateOpts)
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

func TestSQLitePragmasAsString(t *testing.T) {
	p := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}
	c := NewSQLite("/path/to/sqlite.db")
	c.SQLitePragmas = p

	pragmaString := pragmsQueriesToString(c.SQLitePragmas)

	u, err := url.ParseQuery(pragmaString)
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

		enc := parsed.Encode()
		if pragmaString != enc {
			t.Fatal("Incorrect PRAGMAs for connection string.", pragmaString, enc)
			return
		}

	case sqliteLibraryModernc:
		shouldBe := "_pragma=journal_mode=wal&_pragma=busy_timeout=5000"
		parsed, err := url.ParseQuery(shouldBe)
		if err != nil {
			t.Fatal(err)
			return
		}

		enc := parsed.Encode()
		if pragmaString != enc {
			t.Fatal("Incorrect PRAGMAs for connection string.", pragmaString, enc)
			return
		}
	}
}

func TestAddConnectionOption(t *testing.T) {
	//Get config to work off of.
	c := NewMSSQL("10.0.0.1", "db_name", "user1", "password1")

	//Add option.
	k := "key"
	v := "value"
	c.AddConnectionOption(k, v)

	//Make sure connection option was saved.
	vFound, ok := c.ConnectionOptions[k]
	if !ok {
		t.Fatal("Could not find key in connection options.")
		return
	}
	if vFound != v {
		t.Fatal("Connection option value mismatch.", vFound, v)
		return
	}
}

func TestType(t *testing.T) {
	c := NewMariaDB("10.0.0.1", "test", "user", "password")
	Use(c)

	if Type() != DBTypeMariaDB {
		t.Fatal("Type() returned incorrect value.", Type())
		return
	}
}
