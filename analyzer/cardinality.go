package analyzer

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/pbnjay/bdog"
)

type Cardinality struct {
	mod bdog.Model

	// row counts per table
	rows map[string]int

	// values per column per table
	cols map[string]map[string]ColStats
}

// Uniqueness describes the non-empty values of a column.
type Uniqueness string

const (
	// KeyValues indicates that 100.0% of values are unique to each row.
	KeyValues Uniqueness = "key"
	// MostlyUniqueValues indicates that at least 67% of values are unique.
	MostlyUniqueValues Uniqueness = "mostly unique"
	// StandardizedValues indicates that 67-34% of values are unique.
	StandardizedValues Uniqueness = "standardized"
	// ClassValues indicates that less than 34% of values are unique.
	ClassValues Uniqueness = "classes"
)

// Ubiquity describes the frequency of non-empty values in a column.
type Ubiquity string

const (
	// RequiredValues indicates 100% of rows have values.
	RequiredValues Ubiquity = "required"
	// CommonValues indicates 99-67% of rows have values.
	CommonValues Ubiquity = "common"
	// UncommonValues indicates 66-34% of rows have values.
	UncommonValues Ubiquity = "uncommon"
	// RareValues indicates at least 1 row, up to 33% of rows have values.
	RareValues Ubiquity = "rare"
	// UnknownValues indicates no rows had observed values.
	UnknownValues Ubiquity = "unknown"
)

type ColStats struct {
	Name string

	Uniques int
	Blanks  int
	Nulls   int
	Zeros   int

	// UniquePercent is the proportion of unique values, ignoring empty values.
	// e.g. If there are 40 unique values, 2 of which represent 30 empty values,
	//      over 200 rows then UniquePercent = (40-2)/(200-30) = 22.35%
	UniquePercent float64 // 0-100.0

	// UniqueClass classifies the uniqueness of values in the column.
	UniqueClass Uniqueness

	// PresentPercent is the proportion of non-empty values compared to number of rows.
	// e.g. If there are 30 empty values over 200 rows,
	//      then PresentPercent = 170/200 = 85%
	PresentPercent float64 // 0-100.0

	// PresenceClass classifies the ubiquity of non-empty values in the column.
	PresenceClass Ubiquity
}

func (c *Cardinality) String() string {

	maxtabname := 0
	maxcolname := 11 // "### columns"

	for tabName, counts := range c.cols {
		if len(tabName) > maxtabname {
			maxtabname = len(tabName)
		}
		for colName := range counts {
			if len(colName) > maxcolname {
				maxcolname = len(colName)
			}
		}
	}

	lines := []string{
		fmt.Sprintf(" Cardinality Analysis of %d tables", len(c.rows)),
	}

	for tabName, rowCount := range c.rows {
		tab := c.mod.GetTable(tabName)

		lines = append(lines, strings.Repeat("-", maxtabname+maxcolname+24))
		colCounts := c.cols[tabName]
		lines = append(lines, fmt.Sprintf(" %*s %*s %6d rows (PK: %s)",
			-maxtabname, tabName, -maxcolname,
			fmt.Sprintf("%d columns", len(colCounts)), rowCount,
			strings.Join(tab.Key[:], ",")))

		cols := make([]ColStats, 0, len(colCounts))
		for _, cs := range colCounts {
			cols = append(cols, cs)
		}
		sort.Slice(cols, func(i, j int) bool {
			d := (cols[i].PresentPercent - cols[j].PresentPercent)
			if d < 0.1 && d > -0.1 {
				return cols[i].UniquePercent > cols[j].UniquePercent
			}
			return cols[i].PresentPercent > cols[j].PresentPercent
		})
		for _, cs := range cols {
			lines = append(lines, fmt.Sprintf(" %*s %*s %6d unique values (%6.2f%% unique, %6.2f%% present, %s, %s)",
				-maxtabname, " ", -maxcolname, cs.Name,
				cs.Uniques, cs.UniquePercent, cs.PresentPercent, cs.UniqueClass, cs.PresenceClass))
			if others, ok := tab.Linked[bdog.ColumnSetAsString(bdog.ColumnSet{cs.Name})]; ok {
				for other, otherColSets := range others {
					lines = append(lines, fmt.Sprintf(" %*s ^ FOREIGN KEY %s (%s)",
						-maxtabname, " ", other, strings.Join(otherColSets[0], ","),
					))
				}
			}
			for _, uqname := range tab.UniqueColumns {
				if uqname == cs.Name {
					lines = append(lines, fmt.Sprintf(" %*s ^ UNIQUE CONSTRAINT",
						-maxtabname, " ",
					))
				}
			}
			/*if cs.PresentPercent < 100.0 {
				lines = append(lines, fmt.Sprintf(" %*s %*s [ %d blanks, %d nulls, %d zeros ]", -maxtabname, " ", -maxcolname, " ",
					cs.Blanks, cs.Nulls, cs.Zeros))
			}*/
		}
	}

	return strings.Join(lines, "\n")
}

func NewCardinality(m bdog.Model) (*Cardinality, error) {
	tabNames := m.ListTableNames()
	c := &Cardinality{
		mod:  m,
		rows: make(map[string]int, len(tabNames)),
		cols: make(map[string]map[string]ColStats, len(tabNames)),
	}
	for _, tableName := range tabNames {
		tab := m.GetTable(tableName)
		drv, ok := (tab.Driver).(bdog.RawDriver)
		if !ok {
			return nil, errors.New("unable to analyze tables")
		}

		/// count total rows in the table
		n := quickCount(drv, `SELECT COUNT(*) FROM `+tab.Name)
		c.rows[tab.Name] = n

		/// count total unique values in each column
		c.cols[tab.Name] = make(map[string]ColStats, len(tab.Columns))
		for _, colName := range tab.Columns {
			un := quickCount(drv, `SELECT COUNT(DISTINCT `+colName+`) FROM `+tab.Name)
			nn := quickCount(drv, `SELECT COUNT(1) FROM `+tab.Name+` WHERE `+colName+` IS NULL`)
			bn := quickCount(drv, `SELECT COUNT(1) FROM `+tab.Name+` WHERE TRIM(`+colName+`)=''`)
			zn := quickCount(drv, `SELECT COUNT(1) FROM `+tab.Name+` WHERE `+colName+`=0`)

			uqempty := 0
			if bn > 0 {
				uqempty++
			}
			if nn > 0 {
				uqempty++
			}
			if zn > 0 {
				uqempty++
			}
			empty := bn + nn + zn
			upct := float64((un-uqempty)*100) / float64(n-empty)
			ppct := float64((n-empty)*100) / float64(n)

			var preClass Ubiquity = UnknownValues
			switch { // present values (e.g non-"empty")
			case empty == zn: // special case, if the only "empty" values are 0s then we call it required
				preClass = RequiredValues
			case ppct >= 67.0:
				preClass = CommonValues
			case ppct >= 34.0:
				preClass = UncommonValues
			case ppct > 0.0:
				preClass = RareValues
			}

			var valClass Uniqueness
			switch { // unique values
			case upct == 100.0:
				valClass = KeyValues
			case upct >= 67.0:
				valClass = MostlyUniqueValues
			case upct >= 34.0:
				valClass = StandardizedValues
			case upct < 34.0:
				valClass = ClassValues
			}

			c.cols[tab.Name][colName] = ColStats{
				Name:           colName,
				Uniques:        un,
				Blanks:         bn,
				Nulls:          nn,
				Zeros:          zn,
				UniquePercent:  upct,
				UniqueClass:    valClass,
				PresentPercent: ppct,
				PresenceClass:  preClass,
			}
		}
	}

	return c, nil
}

func quickCount(drv bdog.RawDriver, q string) int {
	n := 0
	rows, err := drv.Query(q)
	if err != nil {
		return 0
	}
	rows.Next()
	rows.Scan(&n)
	rows.Close()
	return n
}
