package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const localesQueryFormat = `SELECT code FROM %s._locales`

type PGSyncRow struct {
	SysID            string
	FieldColumns     []string
	FieldValues      map[string]interface{}
	MetaColumns      []string
	Version          int
	PublishedVersion int
	CreatedAt        string
	UpdatedAt        string
	PublishedAt      string
}

type PGSyncTable struct {
	TableName string
	Columns   []string
	Rows      []*PGSyncRow
}

type PGSyncSchema struct {
	SchemaName string
	Tables     map[string]*PGSyncTable
	Deleted    []string
	InitSync   bool
	Locales    []string
}

type PGSyncField struct {
	Type  string
	Value interface{}
}

func NewPGSyncSchema(schemaName string, types []*ContentType, entries []*Entry, initSync bool) *PGSyncSchema {
	schema := &PGSyncSchema{
		SchemaName: schemaName,
		Tables:     make(map[string]*PGSyncTable, 0),
		Deleted:    make([]string, 0),
		InitSync:   initSync,
	}

	// create a "global" entries table to store all entries with sys_id for later delete
	entriesTable := newPGSyncTable("_entries", []string{"table_name"}, []string{})
	appendToEntries := func(tableName string, sysID string, templateFormat bool) {
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

	for _, item := range entries {
		switch item.Sys.Type {
		case ENTRY:
			contentType := item.Sys.ContentType.Sys.ID
			fieldColumns := getFieldColumns(types, contentType)
			baseName := toSnakeCase(contentType)
			appendTables(schema.Tables, item, baseName, fieldColumns, !initSync)
			// append to "global" entries table
			appendToEntries(baseName, item.Sys.ID, !initSync)
			break
		case ASSET:
			baseName := assetTableName
			appendTables(schema.Tables, item, baseName, assetColumns, !initSync)
			// append to "global" entries table
			appendToEntries(baseName, item.Sys.ID, !initSync)
			break
		case DELETED_ENTRY, DELETED_ASSET:
			schema.Deleted = append(schema.Deleted, item.Sys.ID)
			break
		}
	}

	// append the "global" entries table to the tables
	schema.Tables["_entries"] = entriesTable

	return schema
}

func newPGSyncTable(tableName string, fieldColumns []string, metaColumns []string) *PGSyncTable {
	columns := []string{"sys_id"}
	columns = append(columns, fieldColumns...)
	columns = append(columns, metaColumns...)

	return &PGSyncTable{
		TableName: tableName,
		Columns:   columns,
		Rows:      make([]*PGSyncRow, 0),
	}
}

func newPGSyncRow(item *Entry, fieldColumns []string, fieldValues map[string]interface{}, metaColumns []string) *PGSyncRow {
	row := &PGSyncRow{
		SysID:            item.Sys.ID,
		FieldColumns:     fieldColumns,
		FieldValues:      fieldValues,
		MetaColumns:      metaColumns,
		Version:          item.Sys.Version,
		CreatedAt:        item.Sys.CreatedAt,
		UpdatedAt:        item.Sys.UpdatedAt,
		PublishedVersion: item.Sys.PublishedVersion,
		PublishedAt:      item.Sys.PublishedAt,
	}
	if row.Version == 0 {
		row.Version = item.Sys.Revision
	}
	if row.PublishedVersion == 0 {
		row.PublishedVersion = row.Version
	}
	if len(row.UpdatedAt) == 0 {
		row.UpdatedAt = row.CreatedAt
	}
	if len(row.PublishedAt) == 0 {
		row.PublishedAt = row.UpdatedAt
	}
	return row
}

func (r *PGSyncRow) Fields() []interface{} {
	values := []interface{}{
		r.SysID,
	}
	for _, fieldColumn := range r.FieldColumns {
		values = append(values, r.FieldValues[fieldColumn])
	}
	for _, metaColumn := range r.MetaColumns {
		switch metaColumn {
		case "version":
			values = append(values, r.Version)
		case "created_at":
			values = append(values, r.CreatedAt)
		case "created_by":
			values = append(values, "sync")
		case "updated_at":
			values = append(values, r.UpdatedAt)
		case "updated_by":
			values = append(values, "sync")
		case "published_at":
			values = append(values, r.PublishedAt)
		case "published_by":
			values = append(values, "sync")
		}
	}
	return values
}

func (r *PGSyncRow) GetFieldValue(fieldColumn string) string {
	switch fieldColumn {
	case "version":
		return fmt.Sprintf("%d", r.Version)
	case "created_at":
		if r.CreatedAt != "" {
			return fmt.Sprintf("to_timestamp('%s','YYYY-MM-DDThh24:mi:ss.mssZ')", r.CreatedAt)
		}
		return "now()"
	case "created_by":
		return "'sync'"
	case "updated_at":
		if r.UpdatedAt != "" {
			return fmt.Sprintf("to_timestamp('%s','YYYY-MM-DDThh24:mi:ss.mssZ')", r.UpdatedAt)
		}
		return "now()"
	case "updated_by":
		return "'sync'"
	case "published_at":
		if r.PublishedAt != "" {
			return fmt.Sprintf("to_timestamp('%s','YYYY-MM-DDThh24:mi:ss.mssZ')", r.PublishedAt)
		}
		return "now()"
	case "published_by":
		return "'sync'"
	}

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

	// set schema name
	_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
	if err != nil {
		return err
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

	rows, err := txn.Query(fmt.Sprintf(localesQueryFormat, s.SchemaName))
	if err != nil {
		return err
	}
	defer rows.Close()
	s.Locales = make([]string, 0)
	for rows.Next() {
		code := ""
		err := rows.Scan(&code)
		if err != nil {
			return err
		}
		s.Locales = append(s.Locales, fmtLocale(code))
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
			return err
		}

		for _, row := range tbl.Rows {
			_, err = stmt.Exec(row.Fields()...)
			if err != nil {
				return err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
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

	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	return txn.Commit()
}
