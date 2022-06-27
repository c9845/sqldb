package sqldb

import (
	"testing"
)

func TestTFMySQLToSQLiteRemovePrimaryKeyDefinition(t *testing.T) {
	//mysql format query
	mysql := `
		CREATE TABLE IF NOT EXISTS users (
			ID INT NOT NULL AUTO_INCREMENT,
			Username VARCHAR(255) NOT NULL,
			Password TEXT NOT NULL,
			DatetimeCreated DATETIME DEFAULT UTC_TIMESTAMP,
			
			PRIMARY KEY(ID)
		)
	`

	//expected sqlite query
	//note blank lines because PRIMARY KEY(ID) is replaced with "", newlines aren't removed.
	sqliteExpected := `
		CREATE TABLE IF NOT EXISTS users (
			ID INT NOT NULL AUTO_INCREMENT,
			Username VARCHAR(255) NOT NULL,
			Password TEXT NOT NULL,
			DatetimeCreated DATETIME DEFAULT UTC_TIMESTAMP
			
			
		)
	`

	//try the func defined to do this.
	sqliteActual := TFMySQLToSQLiteRemovePrimaryKeyDefinition(mysql)
	if sqliteActual != sqliteExpected {
		t.Fatal("mismatch", sqliteActual, sqliteExpected)
		return
	}
}
