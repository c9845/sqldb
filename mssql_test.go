package sqldb

import "testing"

func TestNewMSSQL(t *testing.T) {
	host := "10.0.0.1"
	dbName := "db_name"
	user := "user"
	password := "password"

	c := NewMSSQL(host, dbName, user, password)
	if c.Type != DBTypeMSSQL {
		t.Fatal("wrong db type", c.Type)
		return
	}

	if c.Host != host {
		t.Fatal("host does not match", c.Host, host)
		return
	}
	if c.Port != defaultMSSQLPort {
		t.Fatal("default port not set")
		return
	}
	if c.Name != dbName {
		t.Fatal("db name does not match", c.Name, dbName)
		return
	}
	if c.User != user {
		t.Fatal("user does not match", c.User, user)
		return
	}
	if c.Password != password {
		t.Fatal("host does not match", c.Password, password)
		return
	}
}

func TestIsMSSQL(t *testing.T) {
	c := NewMSSQL("10.0.0.1", "db_name", "user1", "password!")
	if !c.IsMSSQL() {
		t.Fatal("DB type isn't detected as MSSQL", c.Type)
		return
	}
}
