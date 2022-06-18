package sqldb

//defaults
const defaultMySQLPort = 3306

//NewMySQLConfig returns a config for connecting to a MySQL database.
func NewMySQLConfig(host string, port uint, name, user, password string) (c *Config) {
	//Returned error is ignored since it only returns if a bad db type is provided
	//and we are providing a known good db type here.
	c, _ = NewConfig(DBTypeMySQL)

	c.Host = host
	c.Port = port
	c.Name = name
	c.User = user
	c.Password = password

	return
}

//DefaultMySQLConfig initializes the package level config with some defaults set. This
//wraps around NewSQLiteConfig and saves the config to the package.
func DefaultMySQLConfig(host string, port uint, name, user, password string) {
	cfg := NewMySQLConfig(host, port, name, user, password)
	config = *cfg
}

//IsMySQL returns true if the database is a MySQL database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func (c *Config) IsMySQL() bool {
	return c.Type == DBTypeMySQL
}

//IsMySQL returns true if the database is a MySQL database. This is easier
//than checking for equality against the Type field in the config (c.Type == sqldb.DBTypeSQLite).
func IsMySQL() bool {
	return config.IsMySQL()
}
