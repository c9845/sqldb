package sqldb

// defaults
const defaultMySQLPort uint = 3306

// NewMySQLConfig returns a config for connecting to a MySQL database.
func NewMySQLConfig(host string, port uint, name, user, password string) (cfg *Config) {
	//The returned error can be ignored since it only returns if a bad db type is
	//provided but we are providing a known-good db type.
	cfg, _ = NewConfig(DBTypeMySQL)

	cfg.Host = host
	cfg.Port = port
	cfg.Name = name
	cfg.User = user
	cfg.Password = password

	cfg.ConnectionOptions = make(map[string]string)

	return
}

// DefaultMySQLConfig initializes the globally accessible package level config with
// some defaults set.
func DefaultMySQLConfig(host string, port uint, name, user, password string) {
	cfg := NewMySQLConfig(host, port, name, user, password)
	config = *cfg
}

// IsMySQL returns true if the database is a MySQL database. This is easier
// than checking for equality against the Type field in the config.
func (cfg *Config) IsMySQL() bool {
	return cfg.Type == DBTypeMySQL
}

// IsMySQL returns true if the database is a MySQL database. This is easier
// than checking for equality against the Type field in the config.
func IsMySQL() bool {
	return config.IsMySQL()
}
