package sqldb

// defaults
const defaultMariaDBPort uint = 3306

// NewMariaDB is a shorthand for calling New() and then manually setting the applicable
// MariaDB fields.
func NewMariaDB(host, dbName, user, password string) *Config {
	c := New()
	c.Type = DBTypeMariaDB
	c.Host = host
	c.Port = defaultMariaDBPort
	c.User = user
	c.Password = password

	return c
}

// IsMariaDB returns true if a config represents a MariaDB connection.
func (c *Config) IsMariaDB() bool {
	return c.Type == DBTypeMariaDB
}

// IsMariaDB returns true if a config represents a MariaDB connection.
func IsMariaDB() bool {
	return cfg.IsMariaDB()
}
