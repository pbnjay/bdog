package bdog

import (
	"database/sql"
	"errors"
	"strings"

	goplural "github.com/gertd/go-pluralize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	Driver Driver

	// Name of this table in the database
	Name string

	// Key is the set of column names (in order)
	// used to refer to unique rows of this Table.
	Key ColumnSet

	// Columns lists the column names in this Table.
	Columns ColumnSet

	// UniqueColumns lists the column names containing unique values in this Table.
	// NB these are only single columns with a UNIQUE index, no multi-column support.
	UniqueColumns ColumnSet

	// NewData allocates a new map to hold data from this Table.
	NewData func() map[string]interface{}

	// Linked connects table name (string map keys)
	// from columns in this table (ColumnSet map keys on left)
	// to columns in the linked table (ColumnSet values on right)
	Linked map[ColumnSetString]map[string][]ColumnSet

	// RevLinked is a set of table names that link to this Table.
	RevLinked map[string]struct{}
}

var pluralize = goplural.NewClient()

var caser = cases.Title(language.Und, cases.NoLower)

// PluralName contains a pluralized name for this entity type.
func (t *Table) PluralName(titleCase bool) string {
	if pluralize.IsPlural(t.Name) {
		if titleCase {
			return caser.String(t.Name)
		}
		return strings.ToLower(t.Name)
	}
	if titleCase {
		return caser.String(pluralize.Plural(t.Name))
	}
	return pluralize.Plural(t.Name)
}

// SingleName contains a singular name for this entity type.
func (t *Table) SingleName(titleCase bool) string {
	if pluralize.IsSingular(t.Name) {
		if titleCase {
			return caser.String(t.Name)
		}
		return strings.ToLower(t.Name)
	}
	if titleCase {
		return caser.String(pluralize.Singular(t.Name))
	}
	return pluralize.Singular(t.Name)
}

type Model interface {
	ListTableNames() []string
	GetTable(t string) Table
	ListRelatedTableNames(t string) []string
	GetRelatedTableMappings(t1, t2 string) map[ColumnSetString][]ColumnSet
	GetSubqueryMapping(table1, table2 Table, key string, opts map[string][]string)
}

// opts contains options for the query to pass to the driver
//  "_filters" contains a list of column names to filter for the specified values
//  "_where" contains additional WHERE SQL clauses for the query
//  "_args" contains additional SQL query arguments
//
//  "_page" indicates which page to return
//  "_perpage" indicates the number of results per page to display
//  "_sortby" contains SQL query arguments to include in the ORDER BY
//  (column names) contain lists of values for the specified column

type Driver interface {
	Listing(tab Table, opts map[string][]string) ([]interface{}, error)
	Get(tab Table, opts map[string][]string) (map[string]interface{}, error)
	Insert(tab Table, opts map[string][]string) (interface{}, error)
	Update(tab Table, opts map[string][]string) (interface{}, error)
	Delete(tab Table, opts map[string][]string) error
}
type RawDriver interface {
	QueryPlaceholders(args ...interface{}) []string
	Query(sql99 string, args ...interface{}) (*sql.Rows, error)
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
