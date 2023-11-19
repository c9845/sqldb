package sqldb

import (
	"net/url"
	"strings"

	"github.com/jmoiron/sqlx"
)

const (
	//Possible SQLite libraries. These are used in comparisons, such as when building
	//the connection string PRAGMAs.
	sqliteLibraryMattn   = "github.com/mattn/go-sqlite3"
	sqliteLibraryModernc = "modernc.org/sqlite"

	//InMemoryFilePathRacy is the path to provide for SQLitePath when you want to use
	//an in-memory database instead of a file on disk. This is racy because each call
	//to Connect() will open a brand new database. If you only call Connect() once
	//then this is safe to use.
	//
	//This is good for running tests since then each test runs with a separate
	//in-memory db.
	InMemoryFilePathRacy = ":memory:"

	//InMemoryFilePathRaceSafe is the path to provide for SQLitePath when you want to
	//use an in-memory database instead of a file on disk. This is race safe since
	//multiple calls of Connect() will connect to the same in-memory database,
	//although connecting more than once to the same database would be very odd.
	InMemoryFilePathRaceSafe = "file::memory:?cache=shared"
)

// IsSQLite returns true if a config represents a SQLite connection.
func (c *Config) IsSQLite() bool {
	return c.Type == DBTypeSQLite
}

// IsSQLite returns true if a config represents a SQLite connection.
func IsSQLite() bool {
	return cfg.IsSQLite()
}

// GetSQLiteVersion returns the version of SQLite that is embedded into your app.
// This works by creating a temporary in-memory SQLite database to run a query
// against.
//
// A separate database connection is established because you might want to get the
// SQLite version before you call Connect(). This also just keeps things separate
// from your own connection.
func GetSQLiteVersion() (version string, err error) {
	//Get driver name based on SQLite library in use.
	driver := getDriver(DBTypeSQLite)

	//Connect.
	conn, err := sqlx.Open(driver, InMemoryFilePathRacy)
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
// library is set at build/run with go build tags.
func GetSQLiteLibrary() string {
	return sqliteLibrary
}

// pragmsQueriesToString takes SQLite PRAGMAs in query format and retuns them in the
// format needed to be appended to a SQLite database filepath per the in-use SQLite
// driver.
//
// SQLite PRAGMAs need to be set upon initially connecting to the database. The
// PRAGMAs are added to the database file's path as query parameters (?...&...).
// However, the format of these appended query parameters differs between SQLite
// libraries. This translates PRAGMA statements into the format required by the
// library the binary is built with.
//
// "PRAGMA busy_timeout = 5000" becomes "_pragma=busy_timeout=5000" when using the
// modernc library.
func pragmsQueriesToString(pragmas []string) (filenamePragmaString string) {
	v := url.Values{}

	for _, p := range pragmas {
		//Sanitize, to make replace/stripping of "PRAGMA" keyword easier.
		p = strings.ToLower(p)

		//Strip out the PRAGMA keyword.
		p = strings.TrimPrefix(p, "pragma")

		//Build pragma as expected by library in use.
		switch GetSQLiteLibrary() {
		case sqliteLibraryMattn:
			//Library's format:  _busy_timeout=5000
			key, value, found := strings.Cut(p, "=")
			if !found {
				continue
			}

			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			key = "_" + key
			v.Add(key, value)

		case sqliteLibraryModernc:
			//Library's format: _pragma=busy_timeout=5000
			key := "_pragma"
			value := p

			value = strings.TrimSpace(value)
			value = strings.Replace(value, " ", "", -1)

			v.Add(key, value)

		default:
			//This can never happen since we hardcode the supported SQLite libraries.
		}
	}

	return "?" + v.Encode()
}
