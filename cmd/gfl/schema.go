package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

const tpl = `begin;

create schema {{ .SchemaName }};
{{ range $tblidx, $tbl := .Tables }}
create table {{ $.SchemaName }}.{{ .TableName }} (
  id serial primary key,
  {{- range $colidx, $col := .Columns }}
  {{- if $colidx }},{{- end }}
  {{ .ColumnName }} {{ .ColumnDesc }}
  {{- end }}
);
{{ end }}
{{ range $tblidx, $tbl := .Tables }}
alter table {{ $.SchemaName }}.{{ .TableName }} (
  {{- range $refidx, $ref := .Referencing }}
  {{- if $refidx }},{{- end }}
  integer not null references {{ $.SchemaName }}.{{ $ref }} (id)
  {{- end }}
);
{{ end }}
commit;`

type PGSQLTable struct {
	Referencing  []string
	ReferencedBy []string
	TableName    string
	Columns      []PGSQLColumn
}

type PGSQLColumn struct {
	Table      PGSQLTable
	ColumnName string
	ColumnDesc string
}

type PGSQLSchema struct {
	SchemaName string
	Tables     []PGSQLTable
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Creates postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   "cdn.contentful.com",
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		data, err := client.ContentTypes.Get()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		resp := &gontentful.ContentTypes{}
		err = json.Unmarshal(data, resp)

		schema := NewPGSQLSchema(SpaceId, resp.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(str)
	},
}

func (s *PGSQLSchema) Render() (string, error) {
	tmpl, err := template.New("").Parse(tpl)
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

func NewPGSQLSchema(schemaName string, items []gontentful.ContentType) PGSQLSchema {
	schema := PGSQLSchema{
		SchemaName: schemaName,
	}

	tables := make([]PGSQLTable, 0)
	for _, item := range items {
		table := NewPGSQLTable(item.Sys.ID, item.Fields)
		tables = append(tables, table)
	}

	schema.Tables = tables

	return schema
}

func NewPGSQLTable(tableName string, fields []*gontentful.ContentTypeField) PGSQLTable {
	table := PGSQLTable{
		Referencing:  make([]string, 0),
		ReferencedBy: make([]string, 0),
		TableName:    tableName,
	}

	columns := make([]PGSQLColumn, 0)
	for _, field := range fields {
		column := NewPGSQLColumn(table, *field)
		columns = append(columns, column)
	}

	table.Columns = columns

	return table
}

func NewPGSQLColumn(table PGSQLTable, field gontentful.ContentTypeField) PGSQLColumn {
	column := PGSQLColumn{
		Table:      table,
		ColumnName: field.ID,
	}

	column.ColumnDesc = column.getColumnDesc(field)

	return column
}

func (c *PGSQLColumn) getColumnDesc(field gontentful.ContentTypeField) string {
	columnType := c.getColumnType(field)
	if c.isUnique(field.Validations) {
		columnType += " unique"
	}
	return columnType
}

func (c *PGSQLColumn) getColumnType(field gontentful.ContentTypeField) string {
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
	case "Link":
		return c.getLinkType(field.Validations)
	case "Array":
		return c.getArrayType(*field.Items)
	case "Object":
		return "jsonb"
	default:
		return ""
	}
}

func (c *PGSQLColumn) getArrayType(items gontentful.FieldTypeArrayItem) string {
	switch items.Type {
	case "Symbol":
		return "text ARRAY"
	case "Link":
		return c.getLinkType(items.Validations)
	default:
		return ""
	}
}

func (c *PGSQLColumn) getLinkType(validations []gontentful.FieldValidation) string {
	for _, v := range validations {
		// links
		for _, l := range v.LinkContentType {
			c.Table.Referencing = append(c.Table.Referencing, l)
		}
		// mime types ?
		// for _, m := range v.LinkMimetypeGroup {}
	}
	return ""
}

func (c *PGSQLColumn) isUnique(validations []gontentful.FieldValidation) bool {
	for _, v := range validations {
		if v.Unique {
			return true
		}
	}
	return false
}
