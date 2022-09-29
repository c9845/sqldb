package sqldb

import (
	"net/url"
	"strings"

	"github.com/jmoiron/sqlx"
)

// defaults
const (
	//Possible libraries. Used in comparisons, such as when building connection string
	//pragmas.
	sqliteLibraryMattn   = "github.com/mattn/go-sqlite3"
	sqliteLibraryModernc = "modernc.org/sqlite"

	//InMemoryFilePathRacy is the "path" to provide for the SQLite file when you want
	//to use an in-memory database instead of a filesystem file database. This is racy
	//because each "Connect" call to :memory: will open a brand new database.
	//
	//This is good for running tests since then each test runs with a separate
	//in-memory db.
	InMemoryFilePathRacy = ":memory:"

	//InMemoryFilePathRaceSafe is the "path" to provide for the SQLite file when you
	//want to use an in-memory database between multiple "Connect" calls. This is race
	//safe since multiple calls of "Connect" will connect to the same in-memory db,
	//although connecting more than once to the same db would be very odd.
	InMemoryFilePathRaceSafe = "file::memory:?cache=shared"
)

// NewSQLiteConfig returns a config for connecting to a SQLite database.
func NewSQLiteConfig(pathToFile string) (cfg *Config) {
	//The returned error can be ignored since it only returns if a bad db type is
	//provided but we are providing a known-good db type.
	cfg, _ = NewConfig(DBTypeSQLite)

	cfg.SQLitePath = pathToFile
	cfg.SQLitePragmas = sqliteDefaultPragmas
	cfg.TranslateCreateTableFuncs = []func(string) string{
		TFMySQLToSQLiteReformatID,
		TFMySQLToSQLiteRemovePrimaryKeyDefinition,
		TFMySQLToSQLiteReformatDefaultTimestamp,
		TFMySQLToSQLiteReformatDatetime,
	}

	return
}

// DefaultSQLiteConfig initializes the globally accessible package level config with
// some defaults set.
func DefaultSQLiteConfig(pathToFile string) {
	cfg := NewSQLiteConfig(pathToFile)
	config = *cfg
}

// IsSQLite returns true if the database is a SQLite database. This is easier
// than checking for equality against the Type field in the config.
func (cfg *Config) IsSQLite() bool {
	return cfg.Type == DBTypeSQLite
}

// IsSQLite returns true if the database is a SQLite database. This is easier
// than checking for equality against the Type field in the config.
func IsSQLite() bool {
	return config.IsSQLite()
}

// GetSQLiteVersion returns the version of SQLite that is embedded into the app. This
// works by creating a temporary in-memory SQLite database to run a query against. We
// don't use the config or an already established connection because we may want to
// get the SQLiter version before a database is connected to!
func GetSQLiteVersion() (version string, err error) {
	//Get driver name based on SQLite library in use.
	driver, err := getDriver(DBTypeSQLite)
	if err != nil {
		return
	}

	//Connect.
	conn, err := sqlx.Open(driver, ":memory:")
	if err != nil {
		return
	}
	defer conn.Close()

	//Query for version.
	q := "SELECT sqlite_version()"
	err = conn.Get(&version, q)

	//Close and return.
	err = conn.Close()
	return
}

// GetSQLiteLibrary returns the SQLite library that was used to build the binary. The
// library is set at build/run with -tags {mattn || modernc}.
func GetSQLiteLibrary() string {
	return sqliteLibrary
}

// buildPragmaString builds the string of pragmas that should be appended to the filename
// when connecting to a SQLite database. This is needed to set pragmas reliably since
// pragmas must be set upon initially connecting to the database. The difficulty in
// setting pragmas is that each SQLite library (mattn vs modernc) has a slighly different
// format for setting pragmas. This takes the list of pragmas in SQLite query format (
// PRAGMA busy_timeout = 5000) and translates them to the correct format for the SQLite
// library in use.
func buildPragmaString(pragmas []string) (filenamePragmaString string) {
	v := url.Values{}

	for _, p := range pragmas {
		//Sanitize, make replace/stripping of "PRAGMA" keyword easier.
		p = strings.ToLower(p)

		//Strip out the PRAGMA keyword.
		p = strings.Replace(p, "pragma", "", 1)

		//Build filename pragma as expected by SQLite library is use.
		switch GetSQLiteLibrary() {
		case sqliteLibraryMattn:
			//ex: _busy_timeout=5000
			key, value, found := strings.Cut(p, "=")
			if !found {
				continue
			}

			//trim
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			//add
			key = "_" + key
			v.Add(key, value)
		case sqliteLibraryModernc:
			//ex: _pragma=busy_timeout=5000
			key := "_pragma"
			value := p

			//trim
			value = strings.TrimSpace(value)
			value = strings.Replace(value, " ", "", -1)

			//add
			v.Add(key, value)
		default:
			//this can never happen since we hardcode libraries.
		}
	}

	return "?" + v.Encode()
}

// GetDefaultSQLitePragmas returns the default PRAGMAs this package defines for use with
// either SQLite library. This can be helpful for debugging. We don't just export the
// sqliteDefaultPragmas slice so that it cannot be modified.
func GetDefaultSQLitePragmas() []string {
	return sqliteDefaultPragmas
}
