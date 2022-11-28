package sqldb

import "strings"

// runTranslateUpdateFuncs runs the list of TranslateUpdateFuncs funcs defined on a
// query when  Update() is called.
func (cfg *Config) runTranslateUpdateFuncs(originalQuery string) (translatedQuery string) {
	//Run each translate func. A query may be translated by multiple funcs.
	workingQuery := originalQuery
	for _, f := range cfg.TranslateUpdateFuncs {
		workingQuery = f(workingQuery)
	}

	//Return the completely translated query.
	translatedQuery = workingQuery
	return
}

// TFMySQLToSQLiteBLOB translates TINYBLOB, MEDIUMBLOB, and LONGBLOB to BLOB.
func TFMySQLToSQLiteBLOB(in string) (out string) {
	out = strings.Replace(in, "TINYBLOB", "BLOB", 1)
	out = strings.Replace(out, "MEDIUMBLOB", "BLOB", 1)
	out = strings.Replace(out, "LONGBLOB", "BLOB", 1)
	return out
}
