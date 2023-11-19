package sqldb

// defaults
const defaultMySQLPort uint = 3306

// IsMySQL returns true if a config represents a MySQL connection.
func (c *Config) IsMySQL() bool {
	return c.Type == DBTypeMySQL
}

// IsMySQL returns true if a config represents a MySQL connection.
func IsMySQL() bool {
	return cfg.IsMySQL()
}
