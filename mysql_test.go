package sqldb

import "testing"

func TestNewMySQL(t *testing.T) {
	c := NewMySQL("10.0.0.1", "db_name", "user1", "password1")
	if c.Type != DBTypeMySQL {
		t.FailNow()
		return
	}
}

func TestIsMySQL(t *testing.T) {
	c := NewMySQL("10.0.0.1", "db_name", "user1", "password!")
	if !c.IsMySQL() {
		t.Fatal("DB type isn't detected as MySQL", c.Type)
		return
	}
}
