package sqldb

import "strings"

//Columns is used to hold columns for a query. This helps in organizing a query you
//are building and is useful for generating the correct placeholders when needed using
//the ForSelect(), ForUpdate(), ForInsert() funcs.
//
//Ex:
//cols := Columns{
//	"users.Fname",
//	"users.Birthday",
//}
type Columns []string

//Bindvars holds the parameters you want to use in a query. This helps in organizing
//a query you are building. You can use the values stored in this slice when running
//a query by providing Bindvars... (ex.: c.Get(&var, q, b...) or stmt.Exec(b...). This
//is typically used when building complex queries and in conjunction with the Columns
//type.
type Bindvars []interface{}

//buildColumnString takes a slice of strings, representing columns, and returns them as
//a string to be used in a sql SELECT, INSERT, or UPDATE. This simply formats the columns
//for the query type correctly (concats them together with a seperator and/or parameter
//placeholder (i.e.: ?)) and returns the parameter placholder string to be used for the
//VALUES clause in an INSERT query as needed. Using this func instead of building column
//list manually ensures column list is formatted correctly and count of parameter
//placeholders match the count of columns.
func (cols Columns) buildColumnString(forUpdate bool) (colString, valString string, err error) {
	//make sure at least one column is provided
	if len(cols) == 0 {
		err = ErrNoColumnsGiven
		return
	}

	//build the strings
	if forUpdate {
		//For an UPDATE query, we just append the parameter placeholder to each column
		//name. The first line here adds the =? to each provided column except the last
		//in the slice, the second line adds the =? to the last column.
		colString = strings.Join(cols, "=?,")
		colString += "=?"

	} else {
		//For a SELECT or INSERT query, we just append a comma to separate each column.
		colString = strings.Join(cols, ",")

		//We also need a list of parameter placeholders, also separated by commas. However,
		//the final comma after the last placeholder needs to be stripped to not cause errors.
		valString = strings.Repeat("?,", len(cols))
		valString = valString[:len(valString)-1]
	}

	//Check for any double commas. This is usually caused by a column name being given with
	//a comma already appended or an empty column was provided.
	if idx := strings.Index(colString, ",,"); idx != -1 {
		err = ErrDoubleCommaInColumnString
		return
	}

	return
}

//ForSelect builds the column string for a SELECT query.
func (c Columns) ForSelect() (colString string, err error) {
	colString, _, err = c.buildColumnString(false)
	return
}

//ForInsert builds the column string for an INSERT query.
func (c Columns) ForInsert() (colString, valString string, err error) {
	colString, valString, err = c.buildColumnString(false)
	return
}

//ForUpdate builds the column string for an UPDATE query.
func (c Columns) ForUpdate() (colString string, err error) {
	colString, _, err = c.buildColumnString(true)
	return
}

//Where is the WHERE statement in a query. This separate type is useful for times when you
//are only passing a WHERE clause into a func and you want a bit more control over what is
//provided.
type Where string

//String converts the Where type into a string type for easier appending of strings
//cannot append Where and string b/c they are different types even though they aren't
func (w Where) String() string {
	return string(w)
}
