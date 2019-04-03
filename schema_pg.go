// $ ... | docker exec -i  <containerid> psql -U postgres

package gontentful

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"
)

type PGSQLColumn struct {
	ColumnName string
	ColumnType string
	ColumnDesc string
}

type PGSQLData struct {
	Label        string
	Description  string
	DisplayField string
	Version      int
	CreatedAt    string
	CreatedBy    string
	UpdatedAt    string
	UpdatedBy    string
	Metas        []*PGSQLMeta
}

type PGSQLMeta struct {
	Name        string
	Label       string
	Type        string
	LinkType    string
	Items       string
	Required    bool
	Localized   bool
	Disabled    bool
	Omitted     bool
	Validations string
}

type PGSQLTable struct {
	TableName string
	Data      *PGSQLData
	Columns   []PGSQLColumn
}

type PGSQLSchema struct {
	SchemaName      string
	Space           *Space
	Tables          []PGSQLTable
	References      []PGSQLTable
	AssetReferences []PGSQLTable
}

var funcMap = template.FuncMap{
	"fmtLocale": fmtLocale,
}

func fmtLocale(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", "_"))
}

func NewPGSQLSchema(schemaName string, assetTableName string, space *Space, items []ContentType) PGSQLSchema {
	schema := PGSQLSchema{
		SchemaName: schemaName,
		Space:      space,
		Tables:     make([]PGSQLTable, 0),
	}

	for _, item := range items {
		table := NewPGSQLTable(item)
		schema.Tables = append(schema.Tables, table)

		schema.collectAlters(item, assetTableName)
	}

	return schema
}

func (s *PGSQLSchema) collectAlters(item ContentType, assetTableName string) {
	alterTable := PGSQLTable{
		TableName: item.Sys.ID,
		Columns:   make([]PGSQLColumn, 0),
	}
	alterAssets := PGSQLTable{
		TableName: item.Sys.ID,
		Columns:   make([]PGSQLColumn, 0),
	}
	for _, field := range item.Fields {
		if field.Items != nil {
			if field.Items.LinkType == "Asset" {
				assetRefColumn := PGSQLColumn{
					ColumnName: field.ID,
					ColumnDesc: assetTableName,
				}
				alterAssets.Columns = append(alterAssets.Columns, assetRefColumn)
			} else if field.Items.LinkType == "Entry" {
				for _, v := range field.Items.Validations {
					if len(v.LinkContentType) > 0 {
						for _, link := range v.LinkContentType {
							refColumn := PGSQLColumn{
								ColumnName: field.ID,
								ColumnDesc: link,
							}
							alterTable.Columns = append(alterTable.Columns, refColumn)
						}
					}
				}
			}
		}
	}
	if len(alterTable.Columns) > 0 {
		s.References = append(s.References, alterTable)
	}
	if len(alterAssets.Columns) > 0 {
		s.AssetReferences = append(s.AssetReferences, alterAssets)
	}
}

func (s *PGSQLSchema) Render() (string, error) {
	tmpl, err := template.New("").Funcs(funcMap).Parse(pgTemplate)
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

func NewPGSQLTable(item ContentType) PGSQLTable {
	tableName := item.Sys.ID
	table := PGSQLTable{
		TableName: tableName,
		Columns:   make([]PGSQLColumn, 0),
		Data:      makeModelData(item),
	}

	for _, field := range item.Fields {
		if field.Items == nil || field.Items.Type != "Link" {
			column := NewPGSQLColumn(*field)
			table.Columns = append(table.Columns, column)
		}
		meta := makeMeta(field)
		table.Data.Metas = append(table.Data.Metas, meta)
	}

	return table
}

func NewPGSQLColumn(field ContentTypeField) PGSQLColumn {
	column := PGSQLColumn{
		ColumnName: field.ID,
	}
	column.getColumnDesc(field)
	return column
}

func (c *PGSQLColumn) getColumnDesc(field ContentTypeField) {
	columnDesc := ""
	if c.isUnique(field.Validations) {
		columnDesc += " unique"
	}
	c.ColumnType = c.getColumnType(field)
	c.ColumnDesc = columnDesc
}

func (c *PGSQLColumn) getColumnType(field ContentTypeField) string {
	switch field.Type {
	case "Symbol":
		return "text"
	case "Text":
		return "text"
	case "Integer":
		return "integer"
	case "Number":
		return "decimal"
	case "Date":
		return "date"
	case "Location":
		return "point"
	case "Boolean":
		return "boolean"
	case "Array":
		return "text ARRAY"
	case "Object":
		return "jsonb"
	default:
		return "text"
	}
}

func (c *PGSQLColumn) isUnique(validations []FieldValidation) bool {
	for _, v := range validations {
		if v.Unique {
			return true
		}
	}
	return false
}

func makeModelData(item ContentType) *PGSQLData {
	data := &PGSQLData{
		Label:        formatText(item.Name),
		Description:  formatText(item.Description),
		DisplayField: item.DisplayField,
		Version:      item.Sys.Revision,
		CreatedAt:    item.Sys.CreatedAt,
		UpdatedAt:    item.Sys.UpdatedAt,
		Metas:        make([]*PGSQLMeta, 0),
	}

	return data
}

func makeMeta(field *ContentTypeField) *PGSQLMeta {
	meta := &PGSQLMeta{
		Name:      field.ID,
		Label:     formatText(field.Name),
		Type:      field.Type,
		LinkType:  field.LinkType,
		Required:  field.Required,
		Localized: field.Localized,
		Disabled:  field.Disabled,
		Omitted:   field.Omitted,
	}
	if field.Items != nil {
		i, err := json.Marshal(field.Items)
		if err == nil {
			meta.Items = formatText(string(i))
		}
	}

	if field.Validations != nil {
		v, err := json.Marshal(field.Validations)
		if err == nil {
			meta.Validations = formatText(string(v))
		}
	}

	return meta
}
