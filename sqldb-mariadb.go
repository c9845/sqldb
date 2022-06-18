package sqldb

//defaults
const defaultMariaDBPort = 3306

//NewMariaDBConfig returns a config for connecting to a MySQL database.
func NewMariaDBConfig(host string, port uint, name, user, password string) (c *Config) {
	c = NewMySQLConfig(host, port, name, user, password)
	c.Type = DBTypeMariaDB
	return
}

//DefaultMariaDBConfig initializes the package level config with some defaults set. This
//wraps around NewSQLiteConfig and saves the config to the package.
func DefaultMariaDBConfig(host string, port uint, name, user, password string) {
	cfg := NewMariaDBConfig(host, port, name, user, password)
	config = *cfg
}

//IsMariaDB returns true if the database is a MariaDb database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func (c *Config) IsMariaDB() bool {
	return c.Type == DBTypeMariaDB
}

//IsMariaDB returns true if the database is a MariaDb database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func IsMariaDB() bool {
	return config.IsMariaDB()
}
