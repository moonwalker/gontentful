package gontentful

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type PGPublish struct {
	SchemaName string
	TableName  string
	Columns    []string
	Rows       []*PGSyncRow
	ConTables  []*PGSyncConTable
}

func NewPGPublish(schemaName string, space *Space, contentModel *ContentType, item *PublishedEntry) *PGPublish {

	defLocale := defaultLocale
	locales := make([]string, 0)
	if len(space.Locales) > 0 {
		defLocale = space.Locales[0].Code
		for _, loc := range space.Locales {
			locales = append(locales, loc.Code)
			if loc.Default {
				defLocale = loc.Code
			}
		}
	}

	q := &PGPublish{
		SchemaName: schemaName,
		Rows:       make([]*PGSyncRow, 0),
		ConTables:  make([]*PGSyncConTable, 0),
	}

	contentTypeColumns, columnReferences := getContentTypeColumns(contentModel)
	switch item.Sys.Type {
	case ENTRY:
		contentType := item.Sys.ContentType.Sys.ID
		q.TableName = toSnakeCase(contentType)
		for _, oLoc := range locales {
			loc := strings.ToLower(oLoc)
			fieldValues := make(map[string]interface{})
			id := fmtSysID(item.Sys.ID, true, loc)
			for _, col := range contentTypeColumns {
				prop := toCamelCase(col)
				if item.Fields[prop] != nil {
					fieldValue := item.Fields[prop][oLoc]
					if fieldValue == nil {
						fieldValue = item.Fields[prop][defLocale]
					}
					fieldValues[col] = convertFieldValue(fieldValue, true, loc)
					if columnReferences[col] != "" {
						appendPublishColCons(q, columnReferences[col], col, fieldValue, id, loc)
					}
				}
			}
			q.Rows = append(q.Rows, newPGPublishRow(item.Sys, contentTypeColumns, fieldValues, loc))
		}
		break
	case ASSET:
		q.TableName = assetTableName
		for _, loc := range locales {
			fieldValues := make(map[string]interface{})
			locFile := item.Fields["file"][loc]
			if locFile == nil {
				locFile = item.Fields["file"][defLocale]
			}
			file, ok := locFile.(map[string]interface{})
			if ok {
				fieldValues["url"] = fmt.Sprintf("'%s'", file["url"])
				fieldValues["file_name"] = fmt.Sprintf("'%s'", file["fileName"])
				fieldValues["content_type"] = fmt.Sprintf("'%s'", file["contentType"])
			}
			q.Rows = append(q.Rows, newPGPublishRow(item.Sys, assetColumns, fieldValues, loc))
		}
		break
	}
	return q
}

func (s *PGPublish) Exec(databaseURL string) error {
	tmpl, err := template.New("").Parse(pgPublishTemplate)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}
	// fmt.Println(buff.String())

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

	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}

func newPGPublishRow(sys *Sys, fieldColumns []string, fieldValues map[string]interface{}, locale string) *PGSyncRow {
	row := &PGSyncRow{
		SysID:        sys.ID,
		FieldColumns: fieldColumns,
		FieldValues:  fieldValues,
		Locale:       locale,
		Version:      sys.Version,
		CreatedAt:    sys.CreatedAt,
		UpdatedAt:    sys.UpdatedAt,
	}
	if row.Version == 0 {
		row.Version = sys.Revision
	}
	if len(row.UpdatedAt) == 0 {
		row.UpdatedAt = row.CreatedAt
	}
	return row
}

func appendPublishColCons(q *PGPublish, columnReference string, col string, fieldValue interface{}, id string, loc string) {
	links, ok := fieldValue.([]interface{})
	addedRefs := make(map[string]bool)
	if ok {
		conTableName := getConTableName(q.TableName, col)
		colConTable := &PGSyncConTable{
			TableName: conTableName,
			Columns:   []string{q.TableName, columnReference},
			Rows:      make([][]interface{}, 0),
		}

		for _, e := range links {
			f, ok := e.(map[string]interface{})
			if ok {
				conID := convertSys(f, true, loc)
				if id != "" && conID != "" && !addedRefs[conID] {
					conRow := []interface{}{id, conID}
					colConTable.Rows = append(colConTable.Rows, conRow)
					addedRefs[conID] = true
				} else {
					fmt.Println(q.TableName, id, col, conID)
				}
			}
		}
		q.ConTables = append(q.ConTables, colConTable)
	}
}
