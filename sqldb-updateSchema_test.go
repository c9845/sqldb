package sqldb

import (
	"errors"
	"testing"
)

func TestUFAddDuplicateColumn(t *testing.T) {
	//base config
	c := NewSQLiteConfig("/path/to/db.db")

	//test
	q := "ALTER TABLE users ADD COLUMN Active BOOL NOT NULL DEFAULT 1"
	err := errors.New("Result: duplicate column name: Active")
	if !UFAddDuplicateColumn(*c, q, err) {
		t.Fatal("Duplicate column being added should have been ignored")
		return
	}
}

func TestUFDropUnknownColumn(t *testing.T) {
	//sqlite testing
	c := NewSQLiteConfig("/path/to/db.db")

	q := "ALTER TABLE users DROP COLUMN MiddleName"
	err := errors.New(`Result: no such column: "potatos"`)
	if !UFDropUnknownColumn(*c, q, err) {
		t.Fatal("Already dropped column should have been ignored")
		return
	}

	//mariadb testing
	c = NewMariaDBConfig("host.example.com", defaultMariaDBPort, "dbname", "user", "pwd")

	q = "ALTER TABLE users DROP COLUMN MiddleName"
	err = errors.New("Error Code: 1091. Can't DROP COLUMN `potatos`; check that it exists")
	if !UFDropUnknownColumn(*c, q, err) {
		t.Fatal("Already dropped column should have been ignored")
		return
	}
}

func TestUFModifySQLiteColumn(t *testing.T) {
	//sqlite testing
	c := NewSQLiteConfig("/path/to/db.db")
	q := "ALTER TABLE users MODIFY COLUMN MiddleName VARCHAR(128) NOT NULL"
	err := errors.New(`Result: near "MODIFY": syntax error`)
	if !UFModifySQLiteColumn(*c, q, err) {
		t.Fatal("Cannot modify SQLite columns.")
		return
	}
}

func TestUFAlreadyExists(t *testing.T) {
	//sqlite testing
	c := NewSQLiteConfig("/path/to/db.db")

	q := "CREATE INDEX users__Username_idx ON users(Username);"
	err := errors.New(`Result: index users__Username_idx already exists"`)
	if !UFIndexAlreadyExists(*c, q, err) {
		t.Fatal("Duplicate index should have been ignored")
		return
	}

	//mariadb testing
	c = NewMariaDBConfig("host.example.com", defaultMariaDBPort, "dbname", "user", "pwd")

	q = "CREATE INDEX users__Username_idx ON users(Username);"
	err = errors.New(`Error Code: 1061. Duplicate key name 'users__Username_idx'`)
	if !UFIndexAlreadyExists(*c, q, err) {
		t.Fatal("Duplicate index should have been ignored")
		return
	}
}
