package sqldb

import (
	"errors"
	"log"
)

/*
This file specifically deals with logging for this package. See the config
LoggingLevel field.
*/

// Logging levels, each higher level is inclusive of lower levels; i.e.: if you choose
// to use LogLevelDebug, all Error and Info logging will also be output.
type logLevel int

const (
	LogLevelNone  logLevel = iota //no logging, really should never be used.
	LogLevelError                 //general errors, most typical use.
	LogLevelInfo                  //some info on db connections, deployment, updates.
	LogLevelDebug                 //primarily used during development.

	LogLevelDefault = LogLevelError
)

var (
	//ErrInvalidLoggingLevel is returned when an invalid logging level is provided.
	ErrInvalidLoggingLevel = errors.New("sqldb: invalid logging level")
)

// errorLn performs log.Println if LoggingLevel is set to LogLevelError or a
// higher logging level.
func (c *Config) errorLn(v ...any) {
	if c.LoggingLevel >= LogLevelError {
		log.Println(v...)
	}
}

// infoLn performs log.Println if LoggingLevel is set to LogLevelInfo or a
// higher logging level.
func (c *Config) infoLn(v ...any) {
	if c.LoggingLevel >= LogLevelInfo {
		log.Println(v...)
	}
}

// debugLn performs log.Println if LoggingLevel is set to LogLevelDebug or a
// higher logging level.
func (c *Config) debugLn(v ...any) {
	if c.LoggingLevel >= LogLevelDebug {
		log.Println(v...)
	}
}
