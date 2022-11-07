package sqlite3

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pbnjay/bdog"
)

const (
	DefaultPerPage = 10
	MaxPerPage     = 500
)

type sModel struct {
	conn *sql.DB

	tabs map[string]bdog.Table
}

func (m *sModel) ListTableNames() []string {
	var res []string
	for tn := range m.tabs {
		res = append(res, tn)
	}
	return res
}

func (m *sModel) GetTable(t string) bdog.Table {
	tab, ok := m.tabs[t]
	if ok {
		return tab
	}
	return bdog.Table{}
}

func (m *sModel) ListRelatedTableNames(t string) []string {
	tab, ok := m.tabs[t]
	if !ok {
		return nil
	}
	var res []string
	rels := make(map[string]struct{})
	for _, others := range tab.Linked {
		for tn := range others {
			if _, found := rels[tn]; !found {
				res = append(res, tn)
				rels[tn] = struct{}{}
			}
		}
	}
	for tn := range tab.RevLinked {
		if _, found := rels[tn]; !found {
			res = append(res, tn)
			rels[tn] = struct{}{}
		}
	}
	return res
}

func (m *sModel) GetRelatedTableMappings(t1, t2 string) map[bdog.ColumnSetString][]bdog.ColumnSet {
	tab, ok := m.tabs[t1]
	otherTab, ok2 := m.tabs[t2]
	if !ok || !ok2 {
		return nil
	}
	res := make(map[bdog.ColumnSetString][]bdog.ColumnSet)

	if len(tab.Linked) > 0 {
		for fromCols, others := range tab.Linked {
			res[fromCols] = append(res[fromCols], others[t2]...)
		}
	}
	if len(tab.RevLinked) > 0 {
		if _, isReverse := tab.RevLinked[t2]; isReverse {
			for destCols, others := range otherTab.Linked {
				toCols := bdog.StringAsColumnSet(destCols)
				for _, srcCols := range others[t1] {
					fromCols := bdog.ColumnSetAsString(srcCols)
					res[fromCols] = append(res[fromCols], toCols)
				}
			}
		}
	}
	return res
}

func getData(rows *sql.Rows) (map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		log.Println("err columns", err)
		return nil, err
	}

	args := make([]interface{}, len(cols))
	for i := range cols {
		args[i] = &sql.NullString{}
	}

	/////
	err = rows.Scan(args...)
	if err != nil {
		log.Println("err scan", err)
		return nil, err
	}

	data := make(map[string]interface{})
	for i, arg := range args {
		data[cols[i]] = arg.(*sql.NullString).String
	}
	return data, nil
}

func (m *sModel) Listing(tab bdog.Table, opts map[string][]string) ([]interface{}, error) {
	var res []interface{}

	queryString := "SELECT * FROM " + tab.Name
	var args []interface{}
	if opts != nil {
		if f, ok := opts["_filters"]; ok {
			for _, varname := range f {
				opts["_where"] = append(opts["_where"], fmt.Sprintf("%s=$%d", varname, len(opts["_args"])+1))
				opts["_args"] = append(opts["_args"], opts[varname][0])
			}
		}

		if w, ok := opts["_where"]; ok {
			queryString += " WHERE " + strings.Join(w, " AND ")
		}
		if a, ok := opts["_args"]; ok {
			for _, ax := range a {
				args = append(args, ax)
			}
		}
	}

	offset := 0
	perPage := DefaultPerPage
	if ppg, ok := opts["_perpage"]; ok {
		n, err := strconv.ParseInt(ppg[0], 10, 64)
		if err == nil {
			perPage = int(n)
		}
		if perPage > MaxPerPage || perPage < 1 {
			perPage = 10
		}
	}

	if pg, ok := opts["_page"]; ok {
		n, err := strconv.ParseInt(pg[0], 10, 64)
		if err == nil {
			offset = (int(n) - 1) * perPage
		}
		if offset < 1 {
			offset = 0
		}
	}
	sortKey := strings.Join(tab.Key, ", ")
	if sb, ok := opts["_sortby"]; ok {
		sortKey = ""
		sortNames := strings.Split(sb[0], ",")
		for _, sk := range sortNames {
			for _, cn := range tab.Columns {
				if cn == sk {
					if sortKey != "" {
						sortKey += ", "
					}
					sortKey += cn
				}
			}
		}
	}

	queryString += fmt.Sprintf(" ORDER BY %s LIMIT %d OFFSET %d", sortKey, perPage, offset)
	//log.Println(queryString, args)
	rows, err := m.conn.Query(queryString, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		data, err := getData(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		res = append(res, data)
	}
	rows.Close()

	return res, err
}

func (m *sModel) getLinkedFrom(data map[string]interface{}, tab1, tab2 bdog.Table) error {
	opts := make(map[string][]string)

	for srcCols, others := range tab1.Linked {
		if toColSets, ok := others[tab2.Name]; ok {
			for _, toCols := range toColSets {
				if !tab2.Key.IsEqual(toCols) {
					continue
				}

				fkCS := bdog.StringAsColumnSet(srcCols)
				for i, fk := range fkCS {
					opts[tab2.Key[i]] = append(opts[tab2.Key[i]], fmt.Sprint(data[fk]))
				}
			}
		}
	}
	if len(opts) == 0 {
		return bdog.ErrInvalidInclude
	}

	newData, err := m.Get(tab2, opts)
	if err != nil {
		return err
	}
	data[tab2.Name] = newData
	return nil
}

func (m *sModel) Get(tab bdog.Table, opts map[string][]string) (interface{}, error) {
	var where []string
	var args []interface{}
	for colname, vals := range opts {
		if colname[:1] == "_" {
			continue
		}
		where = append(where, fmt.Sprintf("%s=$%d", colname, len(args)+1))
		args = append(args, vals[0])
	}
	squery := "SELECT * FROM " + tab.Name
	if len(where) > 0 {
		squery += " WHERE " + strings.Join(where, " AND ")
	}
	rows, err := m.conn.Query(squery, args...)
	var data map[string]interface{}
	if err == nil {
		if rows.Next() {
			data, err = getData(rows)
		}
		rows.Close()
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if data == nil {
		return nil, bdog.ErrNotFound
	}

	if nestData, ok := opts["_nest"]; ok && len(nestData) > 0 {
		if tab.Linked == nil && tab.RevLinked == nil {
			// errors.New("invalid nest value(s): " + strings.Join(nestData, ", "))
			return nil, bdog.ErrInvalidInclude
		}
		for _, other := range nestData {
			tab2 := m.GetTable(other)
			err = m.getLinkedFrom(data, tab, tab2)
			if err != nil {
				return nil, err
			}
		}
	}

	return data, err
}

func (m *sModel) Delete(tab bdog.Table, opts map[string][]string) error {
	var where []string
	var args []interface{}
	for i, colname := range tab.Key {
		where = append(where, fmt.Sprintf("%s=$%d", colname, 1+i))
		args = append(args, opts[colname][0])
	}
	squery := "DELETE FROM " + tab.Name
	if len(where) > 0 {
		squery += " WHERE " + strings.Join(where, " AND ")
	} else {
		// invalid LACK OF a filter...
		return bdog.ErrInvalidFilter
	}

	res, err := m.conn.Exec(squery, args...)
	if err != nil {
		log.Println(err)
		return err
	}

	if n, err := res.RowsAffected(); err != nil {
		log.Println(err)
		return err
	} else if n == 0 {
		return bdog.ErrNotFound
	}

	return nil
}

func (m *sModel) Update(tab bdog.Table, opts map[string][]string) (interface{}, error) {
	var where []string
	var args []interface{}
	squery := "UPDATE " + tab.Name + " SET "
	first := true
	for _, colname := range tab.Columns {
		vals, ok := opts[colname]
		if !ok {
			continue
		}
		// skip if colname is in the Key
		skipCol := false
		for _, keycolname := range tab.Key {
			if keycolname == colname {
				skipCol = true
				break
			}
		}
		if skipCol {
			continue
		}

		if first {
			first = false
		} else {
			squery += ", "
		}
		squery += fmt.Sprintf("%s=$%d", colname, len(args)+1)
		args = append(args, vals[0])
	}
	for _, colname := range tab.Key {
		where = append(where, fmt.Sprintf("%s=$%d", colname, len(args)+1))
		args = append(args, opts[colname][0])
	}

	squery += " WHERE " + strings.Join(where, " AND ")
	squery += " RETURNING *"

	rows, err := m.conn.Query(squery, args...)
	var data map[string]interface{}
	if err == nil {
		if rows.Next() {
			data, err = getData(rows)
		}
		rows.Close()
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if data == nil {
		return nil, bdog.ErrNotFound
	}

	return data, err
}

func (m *sModel) Insert(tab bdog.Table, opts map[string][]string) (interface{}, error) {
	var colnames []string
	var placeholders []string
	var args []interface{}
	for _, colname := range tab.Columns {
		if x, ok := opts[colname]; ok && len(x) > 0 {
			colnames = append(colnames, colname)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(placeholders)+1))
			args = append(args, x[0])
		}
	}
	squery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tab.Name, strings.Join(colnames, ","), strings.Join(placeholders, ","))
	squery += " RETURNING *"

	rows, err := m.conn.Query(squery, args...)
	var data map[string]interface{}
	if err == nil {
		if rows.Next() {
			data, err = getData(rows)
		}
		rows.Close()
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if data == nil {
		return nil, bdog.ErrNotFound
	}

	return data, err
}

func (m *sModel) GetSubqueryMapping(table1, table2 bdog.Table, key string, opts map[string][]string) {
	colmaps := m.GetRelatedTableMappings(table1.Name, table2.Name)
	didAdd := false

	for left, rights := range colmaps {
		for _, right := range rights {
			for i, x := range bdog.StringAsColumnSet(left) {
				if !didAdd {
					didAdd = true
					opts["_args"] = append(opts["_args"], key)
				}
				whereClause := fmt.Sprintf("%s.%s IN (SELECT %s FROM %s WHERE %s=$%d)", table2.Name, right[i], x, table1.Name, table1.Key[0], len(opts["_args"]))
				opts["_where"] = append(opts["_where"], whereClause)
			}
		}
	}
}
