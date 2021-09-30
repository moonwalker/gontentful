package gontentful

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type PGPublish struct {
	SchemaName       string
	TableName        string
	Columns          []string
	Rows             []*PGSyncRow
	ConTables        map[string]*PGSyncConTable
	DeletedConTables map[string]*PGSyncConTable
}

func NewPGPublish(schemaName string, space *Space, contentModel *ContentType, item *PublishedEntry) *PGPublish {

	defLocale := defaultLocale
	if len(space.Locales) > 0 {
		defLocale = space.Locales[0].Code
		for _, loc := range space.Locales {
			if loc.Default {
				defLocale = loc.Code
			}
		}
	}

	q := &PGPublish{
		SchemaName:       schemaName,
		Rows:             make([]*PGSyncRow, 0),
		ConTables:        make(map[string]*PGSyncConTable),
		DeletedConTables: make(map[string]*PGSyncConTable),
	}

	switch item.Sys.Type {
	case ENTRY:
		contentTypeColumns, columnReferences := getContentTypeColumns(contentModel)
		contentType := item.Sys.ContentType.Sys.ID
		q.TableName = toSnakeCase(contentType)
		for _, oLoc := range space.Locales {
			loc := strings.ToLower(oLoc.Code)
			fallback := strings.ToLower(oLoc.FallbackCode)
			fieldValues := make(map[string]interface{})
			id := fmtSysID(item.Sys.ID, true, loc)
			for _, col := range contentTypeColumns {
				prop := toCamelCase(col)
				if item.Fields[prop] != nil {
					fieldValue := item.Fields[prop][oLoc.Code]
					if sv, ok := fieldValue.(string); !ok || sv == "" {
						if sv, ok = item.Fields[prop][fallback].(string); ok && sv != "" {
							fieldValue = item.Fields[prop][fallback]
						} else {
							fieldValue = item.Fields[prop][defLocale]
						}
					}
					fieldValues[col] = convertFieldValue(fieldValue, true, loc)
					if columnReferences[col] != "" {
						appendPublishColCons(q, columnReferences[col], col, fieldValue, id, loc)
					}
				} else if _, ok := columnReferences[col]; ok {
					appendDeletedColCons(q, col, id)
				}
			}
			q.Rows = append(q.Rows, newPGPublishRow(item.Sys, contentTypeColumns, fieldValues, loc))
		}
		break
	case ASSET:
		q.TableName = assetTableName
		for _, loc := range locales {
			fieldValues := make(map[string]interface{})
			locTitle := item.Fields["title"][loc]
			if locTitle == nil {
				locTitle = item.Fields["title"][defLocale]
			}
			fieldValues["title"] = fmt.Sprintf("'%s'", locTitle)
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
			q.Rows = append(q.Rows, newPGPublishRow(item.Sys, assetColumns, fieldValues, strings.ToLower(loc)))
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
		if q.ConTables[conTableName] == nil {
			q.ConTables[conTableName] = &PGSyncConTable{
				TableName: conTableName,
				Columns:   []string{q.TableName, columnReference},
				Rows:      make([][]interface{}, 0),
			}
		}

		for _, e := range links {
			f, ok := e.(map[string]interface{})
			if ok {
				conID := convertSys(f, true, loc)
				if id != "" && conID != "" && !addedRefs[conID] {
					conRow := []interface{}{id, conID}
					q.ConTables[conTableName].Rows = append(q.ConTables[conTableName].Rows, conRow)
					addedRefs[conID] = true
				} else {
					fmt.Println(q.TableName, id, col, conID)
				}
			}
		}
	}
}

func appendDeletedColCons(q *PGPublish, col string, id string) {
	conTableName := getConTableName(q.TableName, col)
	if q.DeletedConTables[conTableName] == nil {
		q.DeletedConTables[conTableName] = &PGSyncConTable{
			TableName: conTableName,
			Columns:   []string{q.TableName},
			Rows:      make([][]interface{}, 0),
		}
	}

	if id != "" {
		conRow := []interface{}{id}
		q.DeletedConTables[conTableName].Rows = append(q.DeletedConTables[conTableName].Rows, conRow)
	}
}
