package sqldb

// defaults
const defaultMariaDBPort uint = 3306

// IsMariaDB returns true if a config represents a MariaDB connection.
func (c *Config) IsMariaDB() bool {
	return c.Type == DBTypeMariaDB
}

// IsMariaDB returns true if a config represents a MariaDB connection.
func IsMariaDB() bool {
	return cfg.IsMariaDB()
}
