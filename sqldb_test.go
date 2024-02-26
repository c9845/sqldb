package sqldb

import (
	"strconv"
	"testing"
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

func TestConnect(t *testing.T) {
	//Only test with SQLite since that is the only database type we can be assured
	//exists/is available on any device that attempts to run this func.
	c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)

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
	c.SQLitePath = SQLiteInMemoryFilepathRaceSafe
	err = c.Connect()
	if err == nil {
		t.Fatal("Error about bad db type should have occured.")
		return
	}

	//Test with a good config and PRAGMA set, check PRAGMA is set.
	c.Close()
	c = NewSQLite(SQLiteInMemoryFilepathRaceSafe)
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

func TestDefaultMapperFunc(t *testing.T) {
	in := "asdfasdfasdf"
	out := DefaultMapperFunc(in)
	if in != out {
		t.Fatal("defaultMapperFunc modified provided string but should not have.")
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
	t.Run("mariadb-deploy", func(t *testing.T) {
		c := NewMariaDB("10.0.0.1", "", "user", "password")
		got := c.buildConnectionString(true)
		expected := c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/"
		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string not built correctly.")
			return
		}
	})

	//For connecting to an already existing MariaDB/MySQL database.
	t.Run("mariadb-existing", func(t *testing.T) {
		c := NewMariaDB("10.0.0.1", "db_name", "user", "password")
		got := c.buildConnectionString(false)
		expected := c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + ")/" + c.Name
		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string not built correctly.")
			return
		}
	})

	//For deploying SQLite.
	t.Run("sqlite-deploy", func(t *testing.T) {
		c := NewSQLite("/path/to/sqlite.db")
		got := c.buildConnectionString(true)
		expected := c.SQLitePath
		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string for SQLite should just be the path but wasn't.")
			return
		}
	})

	//For connecting to an already existing SQLite database.
	t.Run("sqlite-existing", func(t *testing.T) {
		c := NewSQLite("/path/to/sqlite.db")
		got := c.buildConnectionString(false)
		expected := c.SQLitePath
		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string for SQLite should just be the path but wasn't.")
			return
		}
	})

	//For connecting to an already existing SQLite using PRAGMAs.
	t.Run("sqlite-with-pragmas", func(t *testing.T) {
		c := NewSQLite("/path/to/sqlite.db")
		c.SQLitePragmas = []string{
			"PRAGMA busy_timeout = 5000",
		}

		got := c.buildConnectionString(false)

		expected := ""
		switch GetSQLiteLibrary() {
		case sqliteLibraryMattn:
			expected = c.SQLitePath + "?_busy_timeout=5000"
		case sqliteLibraryModernc:
			expected = c.SQLitePath + "?_pragma=busy_timeout=5000"
		}

		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string for SQLite with PRAGMAs is wrong.")
			return
		}
	})

	//In-memory SQLite database, that has a query parameter in it already.
	t.Run("sqlite-in-memory", func(t *testing.T) {
		c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)
		c.SQLitePragmas = []string{
			"PRAGMA busy_timeout = 5000",
		}

		got := c.buildConnectionString(false)

		expected := ""
		switch GetSQLiteLibrary() {
		case sqliteLibraryMattn:
			expected = c.SQLitePath + "&_busy_timeout=5000"
		case sqliteLibraryModernc:
			expected = c.SQLitePath + "&_pragma=busy_timeout=5000"
		}

		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string for SQLite with PRAGMAs is wrong.")
			return
		}
	})

	//For deploying MS SQL.
	t.Run("mssql-deploy", func(t *testing.T) {
		c := NewMSSQL("10.0.0.1", "", "user", "password")
		got := c.buildConnectionString(true)
		expected := "sqlserver://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10)
		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string not built correctly.")
			return
		}
	})

	//For connecting to an already existing MS SQL database.
	t.Run("mssql-existing", func(t *testing.T) {
		c := NewMSSQL("10.0.0.1", "", "user", "password")
		got := c.buildConnectionString(false)
		expected := "sqlserver://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + "?database=" + c.Name
		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string not built correctly.")
			return
		}
	})

	//Test MS SQL with additional connection parameters.
	t.Run("mssql-additional", func(t *testing.T) {
		c := NewMSSQL("10.0.0.1", "", "user", "password")
		c.AddConnectionOption("encrypt", "false")
		got := c.buildConnectionString(false)
		expected := "sqlserver://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.FormatUint(uint64(c.Port), 10) + "?database=" + c.Name + "&encrypt=false"
		if got != expected {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("Connection string not built correctly.")
			return
		}
	})
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

func TestClose(t *testing.T) {
	c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)
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
