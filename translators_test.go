package sqldb

import (
	"testing"
)

func TestRunTranslators(t *testing.T) {
	//Define config.
	c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)
	c.DeployQueryTranslators = []Translator{
		TranslateMariaDBToSQLiteCreateTable,
	}

	//MariaDB/MySQL query.
	mariadb := `
		CREATE TABLE IF NOT EXISTS users (
			ID INT NOT NULL AUTO_INCREMENT,
			Username VARCHAR(255) NOT NULL,
			Password TEXT NOT NULL,
			DatetimeCreated DATETIME DEFAULT UTC_TIMESTAMP,
			FileBlob MEDIUMBLOB NOT NULL DEFAULT "",
			
			PRIMARY KEY(ID)
		)
	`

	//Expected SQLite query.
	//
	//Note blank lines because PRIMARY KEY(ID) is replaced with ""; newlines aren't
	//removed. Note tab indentation whitespace too!
	sqliteExpected := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username VARCHAR(255) NOT NULL,
			Password TEXT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			FileBlob BLOB NOT NULL DEFAULT ""
			
			
		)
	`

	//Translate.
	sqliteTranslated := c.RunDeployQueryTranslators(mariadb)
	if sqliteExpected != sqliteTranslated {
		t.Fatal("Bad translation.", sqliteExpected, sqliteTranslated)
	}
}
