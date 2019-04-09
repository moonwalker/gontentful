// $ ... | docker exec -i <containerid> psql -U postgres

package gontentful

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	Columns   []*PGSQLColumn
}

type PGSQLSchema struct {
	SchemaName string
	Drop       bool
	Space      *Space
	Tables     []*PGSQLTable
}

var funcMap = template.FuncMap{
	"fmtLocale": fmtLocale,
}

func fmtLocale(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", "_"))
}

func NewPGSQLSchema(schemaName string, dropSchema bool, space *Space, items []*ContentType) *PGSQLSchema {
	schema := &PGSQLSchema{
		SchemaName: schemaName,
		Drop:       dropSchema,
		Space:      space,
		Tables:     make([]*PGSQLTable, 0),
	}

	for _, item := range items {
		table := NewPGSQLTable(item)
		schema.Tables = append(schema.Tables, table)
	}

	return schema
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

func NewPGSQLTable(item *ContentType) *PGSQLTable {
	tableName := item.Sys.ID
	table := &PGSQLTable{
		TableName: tableName,
		Columns:   make([]*PGSQLColumn, 0),
		Data:      makeModelData(item),
	}

	for _, field := range item.Fields {
		column := NewPGSQLColumn(field)
		table.Columns = append(table.Columns, column)
		meta := makeMeta(field)
		table.Data.Metas = append(table.Data.Metas, meta)
	}

	return table
}

func NewPGSQLColumn(field *ContentTypeField) *PGSQLColumn {
	column := &PGSQLColumn{
		ColumnName: field.ID,
	}
	column.getColumnDesc(field)
	return column
}

func (c *PGSQLColumn) getColumnDesc(field *ContentTypeField) {
	columnDesc := ""
	if c.isUnique(field.Validations) {
		columnDesc += " unique"
	}
	c.ColumnType = getColumnType(field.Type, field.Items)
	c.ColumnDesc = columnDesc
}

func getColumnType(fieldType string, fieldItems *FieldTypeArrayItem) string {
	switch fieldType {
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
	case "Link":
		return "text"
	case "Array":
		if fieldItems != nil {
			return fmt.Sprintf("%s ARRAY", getColumnType(fieldItems.Type, nil))
		}
		return "text ARRAY"
	case "Object":
		return "jsonb"
	default:
		return "text"
	}
}

func (c *PGSQLColumn) isUnique(validations []*FieldValidation) bool {
	for _, v := range validations {
		if v.Unique {
			return true
		}
	}
	return false
}

func makeModelData(item *ContentType) *PGSQLData {
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
