package sqldb

// defaults
const defaultMSSQLPort uint = 1433

// NewMSSQL is a shorthand for calling New() and then manually setting the applicable
// MS SQL Server fields.
func NewMSSQL(host, dbName, user, password string) *Config {
	c := New()
	c.Type = DBTypeMSSQL
	c.Host = host
	c.Port = defaultMSSQLPort
	c.User = user
	c.Password = password

	return c
}

// IsMSSQL returns true if a config represents a MS SQL connection.
func (c *Config) IsMSSQL() bool {
	return c.Type == DBTypeMSSQL
}

// IsMSSQL returns true if a config represents a MS SQL connection.
func IsMSSQL() bool {
	return cfg.IsMSSQL()
}
