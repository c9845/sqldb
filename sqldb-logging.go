package sqldb

import "log"

//This file specifically deals with logging. See the config LoggingLevel field.

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
