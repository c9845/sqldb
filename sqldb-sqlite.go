package sqldb

import "github.com/jmoiron/sqlx"

const (
	//InMemoryFilePathRacy is the "path" to provide for the SQLite file when you want
	//to use an in-memory database instead of a filesystem file database. This is racy
	//because each "Connect" call to :memory: will open a brand new database.
	InMemoryFilePathRacy = ":memory:"

	//InMemoryFilePathRaceSafe is the "path" to provide for the SQLite file when you
	//want to use an in-memory database between multiple "Connect" calls. This is race
	//safe since multiple calls of "Connect" will connect to the same in-memory db,
	//although connecting more than once to the same db would be very odd.
	InMemoryFilePathRaceSafe = "file::memory:?cache=shared"
)

//NewSQLiteConfig returns a config for connecting to a SQLite database.
func NewSQLiteConfig(pathToFile string) (c *Config) {
	//Returned error is ignored since it only returns if a bad db type is provided
	//and we are providing a known good db type here.
	c, _ = NewConfig(DBTypeSQLite)

	c.SQLitePath = pathToFile
	c.SQLitePragmaJournalMode = defaultSQLiteJournalMode
	c.TranslateCreateTableFuncs = []TranslateFunc{
		TFMySQLToSQLiteReformatID,
		TFMySQLToSQLiteRemovePrimaryKeyDefinition,
		TFMySQLToSQLiteReformatDefaultTimestamp,
		TFMySQLToSQLiteReformatDatetime,
	}

	return
}

//DefaultSQLiteConfig initializes the package level config with some defaults set. This
//wraps around NewSQLiteConfig and saves the config to the package.
func DefaultSQLiteConfig(pathToFile string) {
	cfg := NewSQLiteConfig(pathToFile)
	config = *cfg
}

//IsSQLite returns true if the database is a SQLite database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func (c *Config) IsSQLite() bool {
	return c.Type == DBTypeSQLite
}

//IsSQLite returns true if the database is a SQLite database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func IsSQLite() bool {
	return config.IsSQLite()
}

//GetSQLiteVersion returns the version of SQLite that is embedded into the app. This is
//used for diagnostics. This works by creating a temporary in-memory SQLite database to
//run query against.
func GetSQLiteVersion() (version string, err error) {
	driver, err := getDriver(DBTypeSQLite)
	if err != nil {
		return
	}

	//connect
	conn, err := sqlx.Open(driver, ":memory:")
	if err != nil {
		return
	}
	defer conn.Close()

	//query for version
	q := "SELECT sqlite_version()"
	err = conn.Get(&version, q)

	//close
	err = conn.Close()
	return
}

//SQLitePragmaJournalMode set the journal mode for the package level config. Use
//this before calling Connect() to change the journal mode.
func SQLitePragmaJournalMode(j journalMode) {
	config.SQLitePragmaJournalMode = j
}

//GetSQLiteLibrary returns the sqlite library that was used to build the binary. The
//library is set at build/run with -tags {mattn || modernc}. This returns the import
//path of the library in use.
func GetSQLiteLibrary() string {
	return sqliteLibrary
}
