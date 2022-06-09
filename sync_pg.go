package gontentful

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	defaultLocale = "en"
)

var (
	metaColumns           = []string{"_locale", "_version", "_created_at", "_created_by", "_updated_at", "_updated_by"}
	localizedAssetColumns = map[string]bool{
		"title":       true,
		"description": true,
		"file":        true,
	}
)

type PGSyncRow struct {
	ID           string
	SysID        string
	FieldColumns []string
	FieldValues  map[string]interface{}
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

type PGDeletedTable struct {
	TableName string
	SysIDs    []string
}

type PGSyncSchema struct {
	SchemaName       string
	Locales          []*Locale
	DefaultLocale    string
	Tables           map[string]*PGSyncTable
	Deleted          map[string]*PGDeletedTable
	ConTables        map[string]*PGSyncConTable
	DeletedConTables map[string]*PGSyncConTable
	InitSync         bool
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

func NewPGSyncSchema(schemaName string, space *Space, types []*ContentType, entries []*Entry, initSync bool) *PGSyncSchema {

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
		SchemaName:       schemaName,
		Locales:          space.Locales,
		DefaultLocale:    defLocale,
		Tables:           make(map[string]*PGSyncTable),
		Deleted:          make(map[string]*PGDeletedTable),
		ConTables:        make(map[string]*PGSyncConTable),
		DeletedConTables: make(map[string]*PGSyncConTable),
		InitSync:         initSync,
	}

	columnsByContentType := getColumnsByContentType(types)

	for _, item := range entries {
		switch item.Sys.Type {
		case ENTRY:
			contentType := item.Sys.ContentType.Sys.ID
			tableName := toSnakeCase(contentType)
			appendTables(schema, item, tableName, columnsByContentType[contentType].fieldColumns, columnsByContentType[contentType].columnReferences, columnsByContentType[contentType].localizedColumns, !initSync)
			break
		case ASSET:
			appendTables(schema, item, assetTableName, assetColumns, nil, localizedAssetColumns, !initSync)
			break
			// case DELETED_ENTRY:
			// 	contentType := item.Sys.ContentType.Sys.ID
			// 	tableName := toSnakeCase(contentType)
			// 	if schema.Deleted[tableName] == nil {
			// 		schema.Deleted[tableName] = &PGDeletedTable{
			// 			TableName: tableName,
			// 			SysIDs:    make([]string, 0),
			// 		}
			// 		schema.Deleted[tableName].SysIDs = append(schema.Deleted[tableName].SysIDs, item.Sys.ID)
			// 	}
			// 	break
			// case DELETED_ASSET:
			// 	if schema.Deleted[assetTableName] == nil {
			// 		schema.Deleted[assetTableName] = &PGDeletedTable{
			// 			TableName: assetTableName,
			// 			SysIDs:    make([]string, 0),
			// 		}
			// 		schema.Deleted[assetTableName].SysIDs = append(schema.Deleted[assetTableName].SysIDs, item.Sys.ID)
			// 	}
			// 	break
		}
	}

	return schema
}

func newPGSyncTable(tableName string, fieldColumns []string) *PGSyncTable {
	columns := []string{"_id", "_sys_id"}
	columns = append(columns, fieldColumns...)
	columns = append(columns, metaColumns...)

	return &PGSyncTable{
		TableName: tableName,
		Columns:   columns,
		Rows:      make([]*PGSyncRow, 0),
	}
}

func newPGSyncRow(item *Entry, fieldColumns []string, fieldValues map[string]interface{}, locale string) *PGSyncRow {
	row := &PGSyncRow{
		ID:           fmt.Sprintf("%s_%s", item.Sys.ID, locale),
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

func (r *PGSyncRow) Fields() []interface{} {
	values := []interface{}{
		r.ID,
		r.SysID,
	}
	for _, fieldColumn := range r.FieldColumns {
		values = append(values, r.FieldValues[fieldColumn])
	}
	values = append(values, r.Locale, r.Version, r.CreatedAt, "sync", r.UpdatedAt, "sync")
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
		// _, err := txn.Exec("SET session_replication_role=replica")
		// if err != nil {
		// 	return err
		// }

		// bulk insert
		return s.bulkInsert(txn)
	}

	// insert and/or delete changes
	return s.deltaSync(txn)
}

func (s *PGSyncSchema) Render() (string, error) {
	tmpl, err := template.New("").Parse(pgSyncTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
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
			_, err = stmt.Exec(row.Fields()...)
			if err != nil {
				fmt.Println("stmt.Exec error", tbl.TableName, row)
				return err
			}
		}

		err = stmt.Close()
		if err != nil {
			fmt.Println("stmt.Close error", tbl.TableName)
			for _, r := range tbl.Rows {
				fmt.Println("row", fmt.Sprintf("%+v", r))
			}
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
			a := make(map[string]string)
			for _, r := range tbl.Rows {
				sys := r[0].(string)
				id := r[1].(string)
				if a[sys] == "" {
					a[sys] = id
				} else {
					fmt.Println(tbl.TableName, sys, id)
					break
				}
			}
			ioutil.WriteFile("/tmp/"+tbl.TableName, []byte(fmt.Sprintf("%+v", tbl.Rows)), 0644)
			return err
		}

		err = stmt.Close()
		if err != nil {
			fmt.Println("stmt.Close error", tbl.TableName)
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
