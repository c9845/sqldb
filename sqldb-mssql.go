package sqldb

// defaults
const defaultMSSQLPort uint = 1433

// IsMSSQL returns true if a config represents a MS SQL connection.
func (c *Config) IsMSSQL() bool {
	return c.Type == DBTypeMSSQL
}

// IsMSSQL returns true if a config represents a MS SQL connection.
func IsMSSQL() bool {
	return cfg.IsMSSQL()
}
