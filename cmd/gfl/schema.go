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
{{ end -}}
{{ range $refidx, $ref := .References }}
alter table {{ $.SchemaName }}.{{ .TableName }} (
  {{- range $colidx, $col := .Columns }}
  {{- if $colidx }},{{- end }}
  {{ .ColumnName }} integer not null references {{ $.SchemaName }}.{{ .ColumnDesc }} (id)
  {{- end }}
);
{{ end }}
commit;`

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

func NewPGSQLSchema(schemaName string, items []gontentful.ContentType) PGSQLSchema {
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

func (s *PGSQLSchema) collectAlters(item gontentful.ContentType) {
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

func NewPGSQLTable(tableName string, fields []*gontentful.ContentTypeField) PGSQLTable {
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

func NewPGSQLColumn(field gontentful.ContentTypeField) PGSQLColumn {
	column := PGSQLColumn{
		ColumnName: field.ID,
	}
	column.getColumnDesc(field)
	return column
}

func (c *PGSQLColumn) getColumnDesc(field gontentful.ContentTypeField) {
	columnType := c.getColumnType(field)
	if c.isUnique(field.Validations) {
		columnType += " unique"
	}
	c.ColumnDesc = columnType
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
	case "Array":
		return "text ARRAY"
	case "Object":
		return "jsonb"
	default:
		return "text"
	}
}

func (c *PGSQLColumn) isUnique(validations []gontentful.FieldValidation) bool {
	for _, v := range validations {
		if v.Unique {
			return true
		}
	}
	return false
}
