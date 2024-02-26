package sqldb

import "testing"

func TestBuildColumnString(t *testing.T) {
	//Good state for inserting.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, valString, err := c.buildColumnString(false)
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID,Fname,Bday" {
		t.Fatal("colString not build right", colString)
		return
	}
	if valString != "?,?,?" {
		t.Fatal("valString not build right", valString)
		return
	}

	//Good state for updating.
	colString, valString, err = c.buildColumnString(true)
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID=?,Fname=?,Bday=?" {
		t.Fatal("colString not build right", colString)
		return
	}
	if valString != "" {
		t.Fatal("valString should be blank", valString)
		return
	}

	//Provide extra comma in column.
	c = append(c, "Lname,")
	colString, _, err = c.buildColumnString(false)
	if err != ErrExtraCommaInColumnString {
		t.Fatal("Error about double comma should have occured.", colString)
		return
	}

	//Test with no columns
	c = Columns{}
	_, _, err = c.buildColumnString(false)
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestForSelect(t *testing.T) {
	//Good test.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, err := c.ForSelect()
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID,Fname,Bday" {
		t.Fatal("colString not build right", colString)
		return
	}

	//No columns bad test.
	c = Columns{}
	_, err = c.ForSelect()
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestForInsert(t *testing.T) {
	//Good test.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, valString, err := c.ForInsert()
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID,Fname,Bday" {
		t.Fatal("colString not build right", colString)
		return
	}
	if valString != "?,?,?" {
		t.Fatal("valString not build right", valString)
		return
	}

	//No columns bad test.
	c = Columns{}
	_, _, err = c.ForInsert()
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestForUpdate(t *testing.T) {
	//Good test.
	c := Columns{
		"ID",
		"Fname",
		"Bday",
	}
	colString, err := c.ForUpdate()
	if err != nil {
		t.Fatal(err)
		return
	}
	if colString != "ID=?,Fname=?,Bday=?" {
		t.Fatal("colString not build right", colString)
		return
	}

	//No columns bad test.
	c = Columns{}
	_, err = c.ForUpdate()
	if err != ErrNoColumnsGiven {
		t.Fatal("Error about no columns should have occured.")
		return
	}
}

func TestString(t *testing.T) {
	s := "hello"
	w := Where(s)
	if w.String() != s {
		t.Fatal("Mismatch", w.String(), s)
		return
	}
}
