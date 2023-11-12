package sqldb

// defaults
const defaultMariaDBPort uint = 3306

// NewMariaDBConfig returns a config for connecting to a MariaDB database.
func NewMariaDBConfig(host string, port uint, name, user, password string) (cfg *Config) {
	cfg = NewMySQLConfig(host, port, name, user, password)
	cfg.Type = DBTypeMariaDB
	return
}

// DefaultMariaDBConfig initializes the globally accessible package level config with
// some defaults set.
func DefaultMariaDBConfig(host string, port uint, name, user, password string) {
	cfg := NewMariaDBConfig(host, port, name, user, password)
	config = *cfg
}

// IsMariaDB returns true if the database is a MariaDB database. This is easier
// than checking for equality against the Type field in the config.
func (cfg *Config) IsMariaDB() bool {
	return cfg.Type == DBTypeMariaDB
}

// IsMariaDB returns true if the database is a MariaDB database. This is easier
// than checking for equality against the Type field in the config.
func IsMariaDB() bool {
	return config.IsMariaDB()
}

// IsMySQLOrMariaDB returns if the database is a MySQL or MariaDB. This is useful
// since MariaDB is a fork of MySQL and most things are compatible; this way you
// don't need to check IsMySQL() and IsMariaDB().
func (cfg *Config) IsMySQLOrMariaDB() bool {
	return cfg.Type == DBTypeMySQL || cfg.Type == DBTypeMariaDB
}

// IsMySQLOrMariaDB returns if the database is a MySQL or MariaDB for the package
// level config.
func IsMySQLOrMariaDB() bool {
	return config.IsMySQLOrMariaDB()
}
