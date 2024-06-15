package sqldb

import "testing"

func TestNewMariaDB(t *testing.T) {
	host := "10.0.0.1"
	dbName := "db_name"
	user := "user"
	password := "password"

	c := NewMariaDB(host, dbName, user, password)
	if c.Type != DBTypeMariaDB {
		t.FailNow()
		return
	}

	if c.Host != host {
		t.Fatal("host does not match", c.Host, host)
		return
	}
	if c.Port != defaultMariaDBPort {
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

func TestIsMariaDB(t *testing.T) {
	c := NewMariaDB("10.0.0.1", "db_name", "user1", "password!")
	if !c.IsMariaDB() {
		t.Fatal("DB type isn't detected as MariaDB", c.Type)
		return
	}
}
