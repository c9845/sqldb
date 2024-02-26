package sqldb

import "testing"

func TestNewMariaDB(t *testing.T) {
	c := NewMariaDB("10.0.0.1", "db_name", "user1", "password1")
	if c.Type != DBTypeMariaDB {
		t.FailNow()
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
