package gontentful

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	entriesTableName = "_entries"
	defaultLocale    = "en"
)

var (
	metaColumns = []string{"_locale", "_version", "_created_at", "_created_by", "_updated_at", "_updated_by"}
)

type PGSyncRow struct {
	SysID        string
	FieldColumns []string
	FieldValues  map[string]interface{}
	MetaColumns  []string
	Locale       string
	Version      int
	CreatedAt    string
	UpdatedAt    string
}

type PGSyncTable struct {
	TableName string
	Columns   []string
	Rows      []*PGSyncRow
}

type PGSyncSchema struct {
	SchemaName    string
	Locales       []*Locale
	DefaultLocale string
	Tables        map[string]*PGSyncTable
	ConTables     map[string]*PGSyncConTable
	Deleted       []string
	InitSync      bool
}

type PGSyncField struct {
	Type  string
	Value interface{}
}

type PGSyncConTable struct {
	TableName string
	Columns   []string
	Rows      [][]interface{}
}

func NewPGSyncSchema(schemaName string, space *Space, types []*ContentType, entries []*Entry, initSync bool, withMetaData bool) *PGSyncSchema {

	defLocale := defaultLocale
	if len(space.Locales) > 0 {
		defLocale = space.Locales[0].Code
		for _, loc := range space.Locales {
			if loc.Default {
				defLocale = strings.ToLower(loc.Code)
				break
			}
		}
	}

	schema := &PGSyncSchema{
		SchemaName:    schemaName,
		Locales:       space.Locales,
		DefaultLocale: defLocale,
		Tables:        make(map[string]*PGSyncTable, 0),
		ConTables:     make(map[string]*PGSyncConTable, 0),
		Deleted:       make([]string, 0),
		InitSync:      initSync,
	}

	var entriesTable *PGSyncTable

	if withMetaData {
		// create a "global" entries table to store all entries with sys_id for later delete
		entriesTable = newPGSyncTable(entriesTableName, []string{"table_name"})

		// append the "global" entries table to the tables
		schema.Tables[entriesTableName] = entriesTable
	}

	columnsByContentType := getColumnsByContentType(types)

	for _, item := range entries {
		switch item.Sys.Type {
		case ENTRY:
			contentType := item.Sys.ContentType.Sys.ID
			tableName := toSnakeCase(contentType)
			appendTables(schema, item, tableName, columnsByContentType[contentType].fieldColumns, columnsByContentType[contentType].columnReferences, !initSync)
			if withMetaData {
				// append to "global" entries table
				appendToEntries(entriesTable, tableName, item.Sys.ID, !initSync)
			}
			break
		case ASSET:
			appendTables(schema, item, assetTableName, assetColumns, nil, !initSync)
			if withMetaData {
				// append to "global" entries table
				appendToEntries(entriesTable, assetTableName, item.Sys.ID, !initSync)
			}
			break
		case DELETED_ENTRY, DELETED_ASSET:
			schema.Deleted = append(schema.Deleted, item.Sys.ID)
			break
		}
	}

	return schema
}

func appendToEntries(entriesTable *PGSyncTable, tableName string, sysID string, templateFormat bool) {
	fieldValue := tableName
	if templateFormat {
		fieldValue = fmt.Sprintf("'%s'", tableName)
	}
	enrtiesRow := &PGSyncRow{
		SysID:        sysID,
		FieldColumns: []string{"table_name"},
		FieldValues: map[string]interface{}{
			"table_name": fieldValue,
		},
	}
	entriesTable.Rows = append(entriesTable.Rows, enrtiesRow)
}

func newPGSyncTable(tableName string, fieldColumns []string) *PGSyncTable {
	columns := []string{"_sys_id"}
	columns = append(columns, fieldColumns...)
	if tableName != entriesTableName {
		columns = append(columns, metaColumns...)
	}

	return &PGSyncTable{
		TableName: tableName,
		Columns:   columns,
		Rows:      make([]*PGSyncRow, 0),
	}
}

func newPGSyncRow(item *Entry, fieldColumns []string, fieldValues map[string]interface{}, locale string) *PGSyncRow {
	row := &PGSyncRow{
		SysID:        item.Sys.ID,
		FieldColumns: fieldColumns,
		FieldValues:  fieldValues,
		Locale:       locale,
		Version:      item.Sys.Version,
		CreatedAt:    item.Sys.CreatedAt,
		UpdatedAt:    item.Sys.UpdatedAt,
	}
	if row.Version == 0 {
		row.Version = item.Sys.Revision
	}
	if len(row.UpdatedAt) == 0 {
		row.UpdatedAt = row.CreatedAt
	}
	return row
}

func (r *PGSyncRow) Fields(addMeta bool) []interface{} {
	values := []interface{}{
		r.SysID,
	}
	for _, fieldColumn := range r.FieldColumns {
		values = append(values, r.FieldValues[fieldColumn])
	}
	if addMeta {
		values = append(values, r.Locale, r.Version, r.CreatedAt, "sync", r.UpdatedAt, "sync")
	}
	return values
}

func (r *PGSyncRow) GetFieldValue(fieldColumn string) string {
	if r.FieldValues[fieldColumn] != nil {
		return fmt.Sprintf("%v", r.FieldValues[fieldColumn])
	}
	return "NULL"
}

func (s *PGSyncSchema) Exec(databaseURL string) error {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	txn, err := db.Beginx()
	if err != nil {
		return err
	}

	if s.SchemaName != "" {
		// set schema name
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
		if err != nil {
			return err
		}
	}

	// init sync
	if s.InitSync {
		// disable triggers for the current session
		_, err := txn.Exec("SET session_replication_role=replica")
		if err != nil {
			return err
		}

		// bulk insert
		return s.bulkInsert(txn)
	}

	// insert and/or delete changes
	return s.deltaSync(txn)
}

func (s *PGSyncSchema) bulkInsert(txn *sqlx.Tx) error {
	for _, tbl := range s.Tables {
		if len(tbl.Rows) == 0 {
			continue
		}
		stmt, err := txn.Preparex(pq.CopyIn(tbl.TableName, tbl.Columns...))
		if err != nil {
			fmt.Println("txn.Preparex error", tbl.TableName)
			return err
		}
		for _, row := range tbl.Rows {
			_, err = stmt.Exec(row.Fields(tbl.TableName != entriesTableName)...)
			if err != nil {
				fmt.Println("stmt.Exec error", tbl.TableName, row)
				return err
			}
		}

		err = stmt.Close()
		if err != nil {
			return err
		}
	}
	for _, tbl := range s.ConTables {
		if len(tbl.Rows) == 0 {
			continue
		}

		stmt, err := txn.Preparex(pq.CopyIn(tbl.TableName, tbl.Columns...))
		if err != nil {
			fmt.Println("txn.Preparex error", tbl.TableName)
			return err
		}

		for _, row := range tbl.Rows {
			_, err = stmt.Exec(row...)
			if err != nil {
				fmt.Println("stmt.Exec error", tbl.TableName, row)
				return err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			fmt.Println("stmt.Exec", tbl.TableName)
			a := make(map[string]map[string]string)
			for _, r := range tbl.Rows {
				sys := r[0].(string)
				id := r[1].(string)
				loc := r[2].(string)
				if a[sys] == nil {
					a[sys] = make(map[string]string)
				} else if a[sys][id] == "" {
					a[sys][id] = loc
				} else {
					fmt.Println(tbl.TableName, sys, id, loc)
					break
				}
			}
			// ioutil.WriteFile("/tmp/"+tbl.TableName, []byte(fmt.Sprintf("%+v", tbl.Rows)), 0644)
			return err
		}

		err = stmt.Close()
		if err != nil {
			return err
		}
	}

	return txn.Commit()
}

func (s *PGSyncSchema) deltaSync(txn *sqlx.Tx) error {
	tmpl, err := template.New("").Parse(pgSyncTemplate)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	// ioutil.WriteFile("/tmp/deltaSync", buff.Bytes(), 0644)

	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	return txn.Commit()
}
