package sqldb

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestUpdateSchema(t *testing.T) {
	c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)

	//Need to deploy schema first.
	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	deployOpts := &DeploySchemaOptions{
		CloseConnection: false,
	}
	err := c.DeploySchema(deployOpts)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Update schema.
	updateTable := `ALTER TABLE users ADD COLUMN FirstName TEXT`
	c.UpdateQueries = []string{updateTable}

	uf := func(c *sqlx.DB) error {
		q := "ALTER TABLE users ADD COLUMN LastName TEXT"
		_, err := c.Exec(q)
		return err
	}
	c.UpdateFuncs = []QueryFunc{uf}

	updateOpts := &UpdateSchemaOptions{
		CloseConnection: false,
	}
	err = c.UpdateSchema(updateOpts)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Insert into new columns to make sure they were created.
	insert := `INSERT INTO users (Username, FirstName) VALUES (?, ?)`
	_, err = c.connection.Exec(insert, "username@example.com", "john")
	if err != nil {
		t.Fatal(err)
		return
	}

	insert = `INSERT INTO users (Username, FirstName, LastName) VALUES (?, ?, ?)`
	_, err = c.connection.Exec(insert, "username@example.com", "john", "doe")
	if err != nil {
		t.Fatal(err)
		return
	}

	//Try updating with an invalid config.
	c.SQLitePath = ""
	err = c.UpdateSchema(updateOpts)
	if err == nil {
		t.Fatal("Error about invalid config should have occured.")
		return
	}

	//Try updating a db that is not already connected.
	//Note error checking b/c db has not been deployed. If we deploy and close the
	//db connection, the in-memory db is gone.
	c.Close()
	c.SQLitePath = SQLiteInMemoryFilepathRaceSafe
	err = c.UpdateSchema(updateOpts)
	if err != nil && !strings.Contains(err.Error(), "no such table") {
		t.Fatal(err)
		return
	}

	//Test with a bad update func.
	c.Close()
	c = NewSQLite(SQLiteInMemoryFilepathRaceSafe)

	createTable = `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err = c.DeploySchema(deployOpts)
	if err != nil {
		t.Fatal(err)
		return
	}

	uf = func(c *sqlx.DB) error {
		q := "ALTER ELBAT dynamite ADD COLUMN LastName TEXT"
		_, err := c.Exec(q)
		return err
	}
	c.UpdateFuncs = []QueryFunc{uf}

	err = c.UpdateSchema(updateOpts)
	if err == nil {
		t.Fatal("Error about bad update func should have occured.")
		return
	}
	if c.Connected() {
		t.Fatal("Connection should be closed after bad update func.")
		return
	}
}
