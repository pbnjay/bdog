package sqlite3

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pbnjay/bdog"
)

func Open(dbName string) (bdog.Model, error) {
	conn, err := sql.Open("sqlite3", dbName)
	if err == nil {
		err = conn.Ping()
	}
	if err != nil {
		return nil, err
	}

	/////
	// step 1: list all tables
	// step 2: list all columns in each table
	// step 3: list all primary keys for each table
	// step 4: list all foreign keys in each table

	mod := &sModel{conn: conn, tabs: make(map[string]bdog.Table, 10)}
	/////
	rows, err := conn.Query(schemaDumpSQL)
	if err != nil {
		conn.Close()
		return nil, err
	}
	var tabName, colName = "", ""
	var notNull bool
	var pkOrder int
	for rows.Next() {
		err = rows.Scan(&tabName, &colName, &notNull, &pkOrder)
		if err != nil {
			rows.Close()
			conn.Close()
			return nil, err
		}
		tab, found := mod.tabs[tabName]
		if !found {
			tab.Name = tabName
		}
		tab.Columns = append(tab.Columns, colName)
		if pkOrder > 0 {
			tab.Key = append(tab.Key, colName)
		}

		mod.tabs[tabName] = tab
		//log.Println(tabName, colName, notNull, pkOrder)
	}
	rows.Close()

	///
	// check for any unique keys
	rows, err = conn.Query(uniqueColsDumpSQL)
	if err != nil {
		conn.Close()
		return nil, err
	}
	for rows.Next() {
		err := rows.Scan(&tabName, &colName)
		if err != nil {
			rows.Close()
			return nil, err
		}
		tab := mod.tabs[tabName]
		tab.UniqueColumns = append(tab.UniqueColumns, colName)
		mod.tabs[tabName] = tab
	}
	rows.Close()
	///
	rows, err = conn.Query(foreignKeyDumpSQL)
	if err != nil {
		conn.Close()
		return nil, err
	}
	var otherTabName, otherColName = "", ""
	fkID := ""
	type fkinfo struct {
		srcTable  string
		destTable string
		src       bdog.ColumnSet
		dest      bdog.ColumnSet
	}
	fkData := make(map[string]*fkinfo, 5)

	for rows.Next() {
		err = rows.Scan(&fkID, &tabName, &colName, &otherTabName, &otherColName, &pkOrder)
		if err != nil {
			rows.Close()
			conn.Close()
			return nil, err
		}

		if pkOrder == 0 {
			fki := &fkinfo{srcTable: tabName, destTable: otherTabName}
			fkData[tabName+":"+fkID] = fki
		}
		fkData[tabName+":"+fkID].src = append(fkData[tabName+":"+fkID].src, colName)
		fkData[tabName+":"+fkID].dest = append(fkData[tabName+":"+fkID].dest, otherColName)

		//log.Println(fkID, tabName, colName, otherTabName, otherColName, pkOrder)
	}
	rows.Close()

	for _, fk := range fkData {
		tab, found := mod.tabs[fk.srcTable]
		if !found {
			log.Println(tabName)
			panic("corrupted database schema")
		}
		otherTab, found := mod.tabs[fk.destTable]
		if !found {
			log.Println(fk.destTable)
			panic("corrupted database schema")
		}
		if tab.Linked == nil {
			tab.Linked = make(map[bdog.ColumnSetString]map[string][]bdog.ColumnSet)
		}
		if otherTab.RevLinked == nil {
			otherTab.RevLinked = make(map[string]struct{})
		}
		fkcss := bdog.ColumnSetAsString(fk.src)
		if _, ok := tab.Linked[fkcss]; !ok {
			tab.Linked[fkcss] = make(map[string][]bdog.ColumnSet)
		}
		otherTab.RevLinked[fk.srcTable] = struct{}{}
		tab.Linked[fkcss][fk.destTable] = append(tab.Linked[fkcss][fk.destTable], fk.dest)
		mod.tabs[fk.srcTable] = tab
		mod.tabs[fk.destTable] = otherTab
	}

	return mod, nil
}

var schemaDumpSQL = `
SELECT 
	m.name as table_name, 
	p.name as column_name,
	p."notnull" as null_allowed,
	p.pk as pk_order
FROM 
	sqlite_master AS m
JOIN 
	pragma_table_info(m.name) AS p
ORDER BY 
	m.name, 
	p.cid
`

var foreignKeyDumpSQL = `
SELECT 
  f.id,
  m.name as table_name, 
  f."from" as column_name,
  f."table" as fk_table,
  f."to" as fk_column,
	f.seq as fk_seq
FROM 
  sqlite_master AS m
JOIN 
  pragma_foreign_key_list(m.name) AS f
ORDER BY 
  m.name, 
  f.id
`

// dumps ONLY UNIQUE indexes with exactly one column in the table.
var uniqueColsDumpSQL = `
SELECT m.name, MAX(x.name)
  FROM sqlite_master AS m
  JOIN pragma_index_list(m.name) AS i
  JOIN pragma_index_info(i.name) AS x
 WHERE i.origin='u' AND i."unique"=1
GROUP BY m.name, i.name
HAVING MAX(x.seqno)=0
`
