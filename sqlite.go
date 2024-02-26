package sqldb

import (
	"net/url"
	"strings"

	"github.com/jmoiron/sqlx"
)

// library is used for handling the SQLite libraries/drivers that can be used.
type library string

const (
	//Possible SQLite libraries. These are used in comparisons, such as when building
	//the connection string PRAGMAs.
	sqliteLibraryMattn   library = "github.com/mattn/go-sqlite3"
	sqliteLibraryModernc library = "modernc.org/sqlite"
)

const (
	//SQLiteInMemoryFilepathRacy is the path to provide for SQLitePath when you want
	//to use an in-memory database instead of a file on disk. This is racy because
	//each call to Connect() will open a brand new database. If you only call
	//Connect() once then this is safe to use.
	//
	//This is good for running tests since then each test runs with a separate
	//in-memory db.
	SQLiteInMemoryFilepathRacy = ":memory:"

	//SQLiteInMemoryFilepathRaceSafe is the path to provide for SQLitePath when you
	//want to use an in-memory database instead of a file on disk. This is race safe
	//since multiple calls of Connect() will connect to the same in-memory database,
	//although connecting more than once to the same database would be very odd.
	SQLiteInMemoryFilepathRaceSafe = "file::memory:?cache=shared"
)

// NewSQLite is a shorthand for calling New() and then manually setting the applicable
// SQLite fields.
func NewSQLite(path string) *Config {
	c := New()
	c.Type = DBTypeSQLite
	c.SQLitePath = path

	return c
}

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
	conn, err := sqlx.Open(driver, SQLiteInMemoryFilepathRacy)
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
func GetSQLiteLibrary() library {
	return sqliteLibrary
}

// pragmasToURLValues takes SQLite PRAGMAs in SQLite query format and retuns them in
// a url.Values for appending to a SQLite filepath URL.
//
// SQLite PRAGMAs need to be set upon initially connecting to the database. The
// PRAGMAs are added to the database's filepath as query parameters (?...&...).
// However, the format of these appended query parameters differs between SQLite
// libraries (mattn vs modernc). This func translates PRAGMA statements, written in
// the SQLite query format, into the filepath format required by the SQLite driver
// the binary is built with.
//
// Example:
// - SQLite Query Format: "PRAGMA busy_timeout = 5000".
// - Mattn Format:        "_busy_timeout=5000".
// - Modernc: Format:     "_pragma=busy_timeout=5000".
func pragmasToURLValues(pragmas []string, lib library) (v url.Values) {
	v = url.Values{}

	for _, p := range pragmas {
		//Sanitize, to make replace/stripping of "PRAGMA" keyword easier.
		p = strings.ToLower(p)

		//Strip out the PRAGMA keyword.
		p = strings.TrimPrefix(p, "pragma")

		//Build pragma key-value pairs as expected by driver/library in use.
		switch lib {
		case sqliteLibraryMattn:
			key, value, found := strings.Cut(p, "=")
			if !found {
				continue
			}

			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			key = "_" + key
			v.Add(key, value)

		case sqliteLibraryModernc:
			key := "_pragma"
			value := p

			value = strings.TrimSpace(value)
			value = strings.Replace(value, " ", "", -1)

			v.Add(key, value)

		default:
			//This can never happen since we hardcode the supported SQLite libraries.
		}
	}

	return
}
