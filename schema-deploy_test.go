package sqldb

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestDeploySchema(t *testing.T) {
	c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)
	c.LoggingLevel = LogLevelDebug

	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	insertInitial := func(c *sqlx.DB) error {
		insert := "INSERT INTO users (Username) VALUES (?)"
		_, err := c.Exec(insert, "initialuser@example.com")
		return err
	}
	c.DeployFuncs = []QueryFunc{insertInitial}

	opts := &DeploySchemaOptions{
		CloseConnection: false,
	}
	err := c.DeploySchema(opts)
	if err != nil {
		t.Fatal(err)
		return
	}
	if !c.Connected() {
		t.Fatal("Connection should still be connected.")
		return
	}

	//Try inserting.
	insert2 := `INSERT INTO users (Username) VALUES (?)`
	_, err = c.connection.Exec(insert2, "username@example.com")
	if err != nil {
		t.Fatal(err)
		return
	}

	//Make sure results were inserted (DeployFunc and extra query).
	q := "SELECT Count(ID) FROM Users"
	var count int64
	err = c.Connection().Get(&count, q)
	if err != nil {
		t.Fatal("Could not query.")
		return
	} else if count != 2 {
		t.Fatal("Data not inserted correctly.", count)
		return
	}

	//Try deploying to an already connected to db. This must fail.
	err = c.DeploySchema(opts)
	if err != ErrConnected {
		t.Fatal("Error about db already connected should have occured.")
		return
	}

	//Close connection
	c.Close()

	//Try deploying with an invalid config.
	c.SQLitePath = ""
	err = c.DeploySchema(opts)
	if err == nil {
		t.Fatal("Error about invalid config should have occured.")
		return
	}

	//Try deploying with a bad deploy func.
	c.SQLitePath = SQLiteInMemoryFilepathRaceSafe
	insertInitial = func(c *sqlx.DB) error {
		q := "SELECT INTO users VALUES (?)"
		_, err := c.Exec(q, "initialuser@example.com")
		return err
	}
	c.DeployFuncs = []QueryFunc{insertInitial}

	err = c.DeploySchema(opts)
	if err == nil {
		t.Fatal("Error should have occured because of bad deploy func.")
		return
	}
	if c.Connected() {
		t.Fatal("Connection should be closed after deploy func error.")
		return
	}
}

func TestDeploySchemaAndClose(t *testing.T) {
	c := NewSQLite(SQLiteInMemoryFilepathRaceSafe)
	createTable := `
		CREATE TABLE IF NOT EXISTS users (
			ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			Username TEXT NOT NULL
		)
	`
	c.DeployQueries = []string{createTable}

	err := c.DeploySchema(nil)
	if err != nil {
		t.Fatal(err)
		return
	}

	//Make sure connection is connected.
	if c.Connected() {
		t.Fatal("Connection should be connected")
	}

	//Close connection
	c.Close()
}
