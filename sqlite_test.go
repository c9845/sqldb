package sqldb

import (
	"net/url"
	"strings"
	"testing"
)

func TestNewSQLite(t *testing.T) {
	c := NewSQLite("/path/to/sqlite.db")
	if c.Type != DBTypeSQLite {
		t.FailNow()
		return
	}
}

func TestIsSQLite(t *testing.T) {
	c := NewSQLite("/path/to/sqlite.db")
	if !c.IsSQLite() {
		t.Fatal("DB type isn't detected as SQLite", c.Type)
		return
	}
}

func TestGetSQLiteVersion(t *testing.T) {
	v, err := GetSQLiteVersion()
	if err != nil {
		t.Fatal(err)
		return
	}
	if v == "" {
		t.Fatal("SQLite version not returned.", v)
		return
	}
}

func TestPragmasToURLValues(t *testing.T) {
	//Test with mattn, single PRAGMA.
	t.Run("mattn", func(t *testing.T) {
		lib := sqliteLibraryMattn

		pragmas := []string{
			"PRAGMA busy_timeout = 5000",
		}

		got := pragmasToURLValues(pragmas, lib)

		expected := url.Values{}
		expected.Add("_busy_timeout", "5000")

		if got.Encode() != expected.Encode() {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("mismatch for mattn library")
		}
	})

	//Test with mattn, multiple PRAGMAs.
	t.Run("mattn-multi", func(t *testing.T) {
		lib := sqliteLibraryMattn

		pragmas := []string{
			"PRAGMA busy_timeout = 5000",
			"PRAGMA journal_mode = WAL",
		}

		got := pragmasToURLValues(pragmas, lib)

		expected := url.Values{}
		expected.Add("_busy_timeout", "5000")
		expected.Add("_journal_mode", "wal")

		if got.Encode() != expected.Encode() {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("mismatch for mattn library")
			return
		}
	})

	//Test with modernc, single PRAGMA
	t.Run("modernc", func(t *testing.T) {
		lib := sqliteLibraryModernc

		pragmas := []string{
			"PRAGMA busy_timeout = 5000",
		}

		got := pragmasToURLValues(pragmas, lib)

		expected := url.Values{}
		expected.Add("_pragma", "busy_timeout=5000")

		if got.Encode() != expected.Encode() {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("mismatch for modernc library")
		}
	})

	//Test with modernc, multiple PRAGMAs.
	t.Run("modernc-multi", func(t *testing.T) {
		lib := sqliteLibraryModernc

		pragmas := []string{
			"PRAGMA busy_timeout = 5000",
			"PRAGMA journal_mode = WAL",
		}

		got := pragmasToURLValues(pragmas, lib)

		expected := url.Values{}
		expected.Add("_pragma", "busy_timeout=5000")
		expected.Add("_pragma", "journal_mode=wal")

		if got.Encode() != expected.Encode() {
			t.Log("Got:", got)
			t.Log("Exp:", expected)
			t.Fatal("mismatch for modernc library")
			return
		}
		if strings.Count(got.Encode(), "_pragma") != len(pragmas) {
			t.Log("Got:", got.Encode())
			t.Log("Exp:", expected.Encode())
			t.Fatal("mismatch for modernc library")
			return
		}
	})
}
