package sqldb

// defaults
const defaultMySQLPort uint = 3306

// NewMySQL is a shorthand for calling New() and then manually setting the applicable
// MySQL fields.
func NewMySQL(host, dbName, user, password string) *Config {
	c := New()
	c.Type = DBTypeMySQL
	c.Host = host
	c.Port = defaultMySQLPort
	c.Name = dbName
	c.User = user
	c.Password = password

	return c
}

// IsMySQL returns true if a config represents a MySQL connection.
func (c *Config) IsMySQL() bool {
	return c.Type == DBTypeMySQL
}

// IsMySQL returns true if a config represents a MySQL connection.
func IsMySQL() bool {
	return cfg.IsMySQL()
}
