package sqldb

import "testing"

func TestNewMSSQL(t *testing.T) {
	c := NewMSSQL("10.0.0.1", "db_name", "user1", "password1")
	if c.Type != DBTypeMSSQL {
		t.FailNow()
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
