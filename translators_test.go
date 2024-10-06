package sqldb

import (
	"strings"
	"testing"
)

func TestRunTranslators(t *testing.T) {
	//Define config.
	c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)
	c.DeployQueryTranslators = []Translator{
		TranslateMariaDBToSQLite,
	}
	c.UpdateQueryTranslators = []Translator{
		TranslateMariaDBToSQLite,
	}

	//MariaDB/MySQL query.
	mariadb := `
		CREATE TABLE IF NOT EXISTS users (
			ID INT NOT NULL AUTO_INCREMENT,
			Username VARCHAR(255) NOT NULL,
			Password TEXT NOT NULL,
			DatetimeCreated DATETIME DEFAULT UTC_TIMESTAMP,
			FileBlob MEDIUMBLOB NOT NULL DEFAULT "",
			IntColumn INT NOT NULL,
			VarcharToText VARCHAR(255) NOT NULL,
			DecimalToReal DECIMAL(10,4) NOT NULL DEFAULT 1.1234,
			BoolToInt BOOL NOT NULL DEFAULT 0,
			DateToText DATE NOT NULL,
			TimeToText TIME NOT NULL,
			
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
			Username TEXT NOT NULL,
			Password TEXT NOT NULL,
			DatetimeCreated TEXT DEFAULT CURRENT_TIMESTAMP,
			FileBlob BLOB NOT NULL DEFAULT "",
			IntColumn INTEGER NOT NULL,
			VarcharToText TEXT NOT NULL,
			DecimalToReal REAL NOT NULL DEFAULT 1.1234,
			BoolToInt INTEGER NOT NULL DEFAULT 0,
			DateToText TEXT NOT NULL,
			TimeToText TEXT NOT NULL
			
			
		)
	`

	//Translate.
	sqliteTranslated := c.RunDeployQueryTranslators(mariadb)
	if sqliteExpected != sqliteTranslated {
		expectedLines := strings.Split(sqliteExpected, "\n")
		translatedLines := strings.Split(sqliteTranslated, "\n")

		for index, line := range expectedLines {
			if index > len(expectedLines) {
				t.Log("Mismatched line count")
				break
			}

			if line != translatedLines[index] {
				t.Log("Mismatch at line", index)
				t.Log(line)
				t.Log(translatedLines[index])
			}
		}

		t.Fatal("Bad translation.", sqliteExpected, sqliteTranslated)
	}

	//Update schema query...
	mariadb = "ALTER TABLE users ADD COLUMN BoolToInteger BOOL NOT NULL DEFAULT 0"
	sqliteExpected = "ALTER TABLE users ADD COLUMN BoolToInteger INTEGER NOT NULL DEFAULT 0"

	sqliteTranslated = c.RunUpdateQueryTranslators(mariadb)
	if sqliteExpected != sqliteTranslated {
		t.Fatal("Bad translation.")
	}
}
