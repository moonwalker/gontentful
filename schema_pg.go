// $ ... | docker exec -i  <containerid> psql -U postgres

package gontentful

import (
	"bytes"
	"text/template"
)

const pgTemplate = `BEGIN;

CREATE SCHEMA {{ .SchemaName }};
{{ range $tblidx, $tbl := .Tables }}
CREATE TABLE {{ $.SchemaName }}.{{ .TableName }} (
  id serial primary key,
  {{- range $colidx, $col := .Columns }}
  {{- if $colidx }},{{- end }}
  {{ .ColumnName }} {{ .ColumnDesc }}
  {{- end }}
);
{{ end -}}
{{ range $refidx, $ref := .References }}
ALTER TABLE {{ $.SchemaName }}.{{ .TableName }}
  {{- range $colidx, $col := .Columns }}
  {{- if $colidx }},{{- end }}
  ADD COLUMN {{ .ColumnName }} integer references {{ $.SchemaName }}.{{ .ColumnDesc }}(id)
  {{- end }};
{{ end }}
COMMIT;`

type PGSQLColumn struct {
	ColumnName string
	ColumnDesc string
}

type PGSQLTable struct {
	TableName string
	Columns   []PGSQLColumn
}

type PGSQLSchema struct {
	SchemaName string
	Tables     []PGSQLTable
	References map[string]PGSQLTable
}

func NewPGSQLSchema(schemaName string, items []ContentType) PGSQLSchema {
	schema := PGSQLSchema{
		SchemaName: schemaName,
		Tables:     make([]PGSQLTable, 0),
		References: make(map[string]PGSQLTable, 0),
	}

	for _, item := range items {
		table := NewPGSQLTable(item.Sys.ID, item.Fields)
		schema.Tables = append(schema.Tables, table)

		schema.collectAlters(item)
	}

	return schema
}

func (s *PGSQLSchema) collectAlters(item ContentType) {
	alterTable := PGSQLTable{
		TableName: item.Sys.ID,
		Columns:   make([]PGSQLColumn, 0),
	}
	for _, field := range item.Fields {
		if field.Items != nil {
			for _, v := range field.Items.Validations {
				if len(v.LinkContentType) > 0 {
					for _, link := range v.LinkContentType {
						refColumn := PGSQLColumn{
							ColumnName: field.ID,
							ColumnDesc: link,
						}
						alterTable.Columns = append(alterTable.Columns, refColumn)
						s.References[item.Sys.ID] = alterTable
					}
				}
			}
		}
	}
}

func (s *PGSQLSchema) Render() (string, error) {
	tmpl, err := template.New("").Parse(pgTemplate)
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

func NewPGSQLTable(tableName string, fields []*ContentTypeField) PGSQLTable {
	table := PGSQLTable{
		TableName: tableName,
		Columns:   make([]PGSQLColumn, 0),
	}

	for _, field := range fields {
		if field.Items == nil || field.Items.Type != "Link" {
			column := NewPGSQLColumn(*field)
			table.Columns = append(table.Columns, column)
		}
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
	columnType := c.getColumnType(field)
	if c.isUnique(field.Validations) {
		columnType += " unique"
	}
	c.ColumnDesc = columnType
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
