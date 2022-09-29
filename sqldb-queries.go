package sqldb

import "strings"

// Columns is used to hold columns for a query. This helps in organizing a query you
// are building and is useful for generating the correct placeholders when needed using
// the ForSelect(), ForUpdate(), ForInsert() funcs.
//
// Ex:
//
//	cols := Columns{
//		"Fname",
//		"Birthday",
//	 "CompanyID",
//	}
//
// colString, valString, _ := cols.ForInsert
// //colString will be "Fname,Birthday,CompanyID"
// //valString will be "?,?,?"
// Use like: "INSERT INTO users (" + colString + ") VALUES (" + valString + ")"
type Columns []string

// Bindvars holds the parameters you want to use in a query. This helps in organizing
// a query you are building. You can use the values stored in this slice when running
// a query by providing Bindvars... (ex.: c.Get(&var, q, b...) or stmt.Exec(b...). This
// is typically used when building complex queries and in conjunction with the Columns
// type.
type Bindvars []interface{}

// buildColumnString takes a slice of strings, representing columns, and returns them as
// a string to be used in a sql SELECT, INSERT, or UPDATE. This simply formats the columns
// for the query type correctly (concats them together with a seperator and/or parameter
// placeholder (i.e.: ?)) and returns the parameter placholder string to be used for the
// VALUES clause in an INSERT query as needed. Using this func instead of building column
// list manually ensures column list is formatted correctly and count of parameter
// placeholders match the count of columns.
//
// Use ForSelect, ForUpdate, or ForInsert instead.
func (cols Columns) buildColumnString(forUpdate bool) (colString, valString string, err error) {
	//Make sure at least one column is provided.
	if len(cols) == 0 {
		err = ErrNoColumnsGiven
		return
	}

	//Build the strings
	if forUpdate {
		//For an UPDATE query, we just append the parameter placeholder to each column
		//name. The first line here adds the =? to each provided column except the last
		//in the slice, the second line adds the =? to the last column.
		colString = strings.Join(cols, "=?,")
		colString += "=?"

	} else {
		//For a SELECT or INSERT query, we just use a comma to separate each provided
		//column. The last column does not have a trailing comma.
		colString = strings.Join(cols, ",")

		//We also need a list of parameter placeholders, also separated by commas.
		//However, we need to make sure that there is trailing comma (the resulting
		//string should end in ?).
		valString = strings.Repeat("?,", len(cols))
		valString = strings.TrimSuffix(valString, ",")
	}

	//Check for any extra commas. This is usually caused by a column name being given
	//with a comma already appended or an empty column was provided.
	doubleCommaIdx := strings.Index(colString, ",,")
	hasTrailingComma := strings.HasSuffix(colString, ",")
	if doubleCommaIdx != -1 || hasTrailingComma {
		err = ErrExtraCommaInColumnString

		//We could set colString equal to "" here but sending back colString is
		//helpful for diagnosing where the double comma occured.

		return
	}

	return
}

// ForSelect builds the column string for a SELECT query.
func (cols Columns) ForSelect() (colString string, err error) {
	colString, _, err = cols.buildColumnString(false)
	return
}

// ForInsert builds the column string for an INSERT query and also returns the
// placholder VALUES() string you should use.
func (cols Columns) ForInsert() (colString, valString string, err error) {
	colString, valString, err = cols.buildColumnString(false)
	return
}

// ForUpdate builds the column string for an UPDATE query.
func (cols Columns) ForUpdate() (colString string, err error) {
	colString, _, err = cols.buildColumnString(true)
	return
}

// Where is the WHERE statement in a query. This separate type is useful for times when
// you are passing a WHERE clause into a func and you want a bit more control over what
// is provided.
type Where string

// String converts the Where type into a string type for easier use.
func (w Where) String() string {
	return string(w)
}
