package sqldb

// defaults
const defaultMSSQLPort uint = 1433

// NewMSSQLConfig returns a config for connecting to a Microsoft SQL Server database.
func NewMSSQLConfig(host string, port uint, name, user, password string) (cfg *Config) {
	cfg = NewMySQLConfig(host, port, name, user, password)
	cfg.Type = DBTypeMSSQL
	return
}

// DefaultMSSQLConfig initializes the globally accessible package level config with
// some defaults set.
func DefaultMSSQLConfig(host string, port uint, name, user, password string) {
	cfg := NewMSSQLConfig(host, port, name, user, password)
	config = *cfg
}

// IsMSSQL returns true if the database is a Microsoft SQL Server database. This is
// easier than checking for equality against the Type field in the config.
func (cfg *Config) IsMSSQL() bool {
	return cfg.Type == DBTypeMSSQL
}

// IsMSSQL returns true if the database is a Microsoft SQL Server database. This is
// easier than checking for equality against the Type field in the config.
func IsMSSQL() bool {
	return config.IsMariaDB()
}
