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
	Locales          []*Locale
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
		Locales:          space.Locales,
	}

	switch item.Sys.Type {
	case ENTRY:
		contentTypeColumns, columnReferences, localizedColumns := getContentTypeColumns(contentModel)
		contentType := item.Sys.ContentType.Sys.ID
		q.TableName = toSnakeCase(contentType)
		for _, oLoc := range space.Locales {
			loc := strings.ToLower(oLoc.Code)
			fieldValues := make(map[string]interface{})
			id := fmtSysID(item.Sys.ID, true, loc)
			for _, col := range contentTypeColumns {
				prop := toCamelCase(col)
				oLocCode := oLoc.Code
				if !localizedColumns[col] {
					oLocCode = defLocale
				}
				if item.Fields[prop] != nil {
					fieldValue := item.Fields[prop][oLocCode]
					if sv, ok := fieldValue.(string); fieldValue == nil || (ok && sv == "") {
						continue
					}
					fieldValues[col] = convertFieldValue(fieldValue, true, loc)
					if columnReferences[col] != "" {
						appendPublishColCons(q, columnReferences[col], col, fieldValue, item.Sys.ID, id, loc)
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
		for _, oLoc := range space.Locales {
			fieldValues := make(map[string]interface{})
			locTitle := item.Fields["title"][oLoc.Code]
			if locTitle != nil {
				fieldValues["title"] = fmt.Sprintf("'%s'", locTitle)
			}
			locFile := item.Fields["file"][oLoc.Code]
			file, ok := locFile.(map[string]interface{})
			if ok {
				fieldValues["url"] = fmt.Sprintf("'%s'", file["url"])
				fieldValues["file_name"] = fmt.Sprintf("'%s'", file["fileName"])
				fieldValues["content_type"] = fmt.Sprintf("'%s'", file["contentType"])
			}
			if locTitle == nil && locFile == nil {
				continue
			}
			q.Rows = append(q.Rows, newPGPublishRow(item.Sys, assetColumns, fieldValues, strings.ToLower(oLoc.Code)))
		}
		break
	}
	return q
}

func (s *PGPublish) Exec(databaseURL string) error {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}
	tmpl, err := template.New("").Funcs(funcMap).Parse(pgPublishTemplate)
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

func appendPublishColCons(q *PGPublish, columnReference string, col string, fieldValue interface{}, sys_id string, id string, loc string) {
	links, ok := fieldValue.([]interface{})
	addedRefs := make(map[string]bool)
	if ok {
		conTableName := getConTableName(q.TableName, col)
		if q.ConTables[conTableName] == nil {
			q.ConTables[conTableName] = &PGSyncConTable{
				TableName: conTableName,
				Columns:   []string{q.TableName, fmt.Sprintf("%s_sys_id", q.TableName), columnReference, fmt.Sprintf("%s_sys_id", columnReference), "_locale"},
				Rows:      make([][]interface{}, 0),
			}
		}

		for _, e := range links {
			f, ok := e.(map[string]interface{})
			if ok {
				conSysID := convertSysID(f, true)
				conID := convertSys(f, true, loc)
				if id != "" && conID != "" && !addedRefs[conID] {
					conRow := []interface{}{id, fmt.Sprintf("'%s'", sys_id), conID, conSysID, fmt.Sprintf("'%s'", loc)}
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
