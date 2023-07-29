package sqldb

import "log"

//This file specifically deals with logging. See the config LoggingLevel field.

// errorPrintLn performs log.Println if LoggingLevel is set to LogLevelError or a
// higher logging level.
func (cfg *Config) errorPrintln(v ...any) {
	if cfg.LoggingLevel >= LogLevelError {
		log.Println(v...)
	}
}

// infoPrintLn performs log.Println if LoggingLevel is set to LogLevelInfo or a
// higher logging level.
func (cfg *Config) infoPrintln(v ...any) {
	if cfg.LoggingLevel >= LogLevelInfo {
		log.Println(v...)
	}
}

// debugPrintLn performs log.Println if LoggingLevel is set to LogLevelDebug or a
// higher logging level.
func (cfg *Config) debugPrintln(v ...any) {
	if cfg.LoggingLevel >= LogLevelDebug {
		log.Println(v...)
	}
}
