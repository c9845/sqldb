package sqldb

import "testing"

func TestRunTranslateUpdateFuncs(t *testing.T) {
	//Define config.
	cfg := NewSQLiteConfig(InMemoryFilePathRaceSafe)
	cfg.TranslateUpdateFuncs = []func(string) string{
		TFMySQLToSQLiteBLOB,
	}

	//Test query.
	original := "ALTER TABLE test_table ADD column_test MEDIUMBLOB NOT NULL DEFAULT ''"
	expected := "ALTER TABLE test_table ADD column_test BLOB NOT NULL DEFAULT ''"

	//Translate.
	translated := cfg.runTranslateUpdateFuncs(original)
	if expected != translated {
		t.Fatal("Bad translation.", translated, expected)
	}
}
