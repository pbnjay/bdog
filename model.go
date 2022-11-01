package bdog

import (
	"errors"
	"strings"
)

// ColumnSet is an ordered list of column names
type ColumnSet []string

// ColumnSetString is the above type joined by ","
type ColumnSetString string

// IsEqual returns true if the receiver has the same columns
// as the argument ColumnSet / ColumnSetString value
func (c ColumnSet) IsEqual(colset interface{}) bool {
	cs, ok := colset.(ColumnSet)
	if ok {
		if len(cs) != len(c) {
			return false
		}
		for i, x := range cs {
			if x != c[i] {
				return false
			}
		}
		return true
	}
	css, ok := colset.(ColumnSetString)
	if ok {
		return css == ColumnSetAsString(c)
	}
	panic("ColumnSet.IsEqual but not ColumnSet(String)")
}

// IsEqual returns true if the receiver has the same columns
// as the argument ColumnSet / ColumnSetString value
func (c ColumnSetString) IsEqual(colset interface{}) bool {
	css, ok := colset.(ColumnSetString)
	if ok {
		return css == c
	}
	cs, ok := colset.(ColumnSet)
	if ok {
		return c == ColumnSetAsString(cs)
	}

	panic("ColumnSetString.IsEqual but not ColumnSet(String)")
}

// ColumnSetAsString converts a ColumnSet to a ColumnSetString.
func ColumnSetAsString(cs ColumnSet) ColumnSetString {
	return ColumnSetString(strings.Join(cs, ","))
}

// StringAsColumnSet converts a ColumnSetString to a ColumnSet.
func StringAsColumnSet(cs ColumnSetString) ColumnSet {
	return ColumnSet(strings.Split(string(cs), ","))
}

// Table represents a table in the database schema.
type Table struct {
	Driver interface{}

	// Name of this table in the database
	Name string

	// Key is the set of column names (in order)
	// used to refer to unique rows of this Table.
	Key ColumnSet

	// Columns lists the unique column names in this Table.
	Columns ColumnSet

	// NewData allocates a new map to hold data from this Table.
	NewData func() map[string]interface{}

	// Linked connects table name (string map keys)
	// from columns in this table (ColumnSet map keys on left)
	// to columns in the linked table (ColumnSet values on right)
	Linked map[ColumnSetString]map[string][]ColumnSet

	// RevLinked is a set of table names that link to this Table.
	RevLinked map[string]struct{}
}

type Model interface {
	ListTableNames() []string
	GetTable(t string) Table
	ListRelatedTableNames(t string) []string
	GetRelatedTableMappings(t1, t2 string) map[ColumnSetString][]ColumnSet
}

type Driver interface {
	Listing(tab Table, opts map[string][]string) ([]interface{}, error)
	Get(tab Table, opts map[string][]string) (interface{}, error)
	Insert(tab Table, opts map[string][]string) (interface{}, error)
	Update(tab Table, opts map[string][]string) (interface{}, error)
	Delete(tab Table, opts map[string][]string) error
}

var (
	// ErrNotFound is returned by Driver.Get when a record is not found.
	ErrNotFound = errors.New("bdog: not found")
	// ErrInsertFailed is returned by Driver.Insert when record creation fails.
	ErrInsertFailed = errors.New("bdog: insert failed")
	// ErrInvalidInclude is returned by Driver.Get when an invalid "include" is requested.
	ErrInvalidInclude = errors.New("bdog: invalid include")
	// ErrInvalidFilter is returned by Driver.Listing when an invalid "filter" is requested.
	ErrInvalidFilter = errors.New("bdog: invalid filter")
)
