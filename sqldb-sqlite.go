package sqldb

import "github.com/jmoiron/sqlx"

//NewSQLiteConfig returns a config for connecting to a SQLite database.
func NewSQLiteConfig(pathToFile string) (c *Config) {
	c = NewConfig(DBTypeSQLite)

	c.SQLitePath = pathToFile
	c.SQLitePragmaJournalMode = defaultSQLiteJournalMode
	c.TranslateCreateTableFuncs = []TranslateFunc{
		TFToSQLiteReformatID,
		TFToSQLiteRemovePrimaryKeyDefinition,
		TFToSQLiteReformatDefaultTimestamp,
		TFToSQLiteReformatDatetime,
	}

	return
}

//DefaultSQLiteConfig initializes the package level config with some defaults set. This
//wraps around NewSQLiteConfig and saves the config to the package.
func DefaultSQLiteConfig(pathToFile string) {
	cfg := NewSQLiteConfig(pathToFile)
	config = *cfg
	return
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
	//connect
	conn, err := sqlx.Open("sqlite3", ":memory:")
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
