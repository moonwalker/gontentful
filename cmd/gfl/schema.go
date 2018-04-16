package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/moonwalker/gontentful"

	"github.com/cbroglie/mustache"
	"github.com/spf13/cobra"
)

const tpl = `
begin;

create schema {{SchemaName}};

{{#Tables}}
create table {{SchemaName}}.{{TableName}} (
  id serial primary key,
  {{#Columns}}
  {{ColumnName}} {{ColumnDesc}}{{^Last}},{{/Last}}
  {{/Columns}}
);

{{/Tables}}
commit;
`

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
		s, err := mustache.Render(tpl, &schema)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(s)
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}

type PGSQLSchema struct {
	SchemaName string
	Tables     []PGSQLTable
}

func NewPGSQLSchema(schemaName string, items []gontentful.ContentType) PGSQLSchema {
	schema := PGSQLSchema{
		SchemaName: schemaName,
	}

	tables := make([]PGSQLTable, 0)
	for _, item := range items {
		table := NewPGSQLTable(schemaName, item.Sys.ID, item.Fields)
		tables = append(tables, table)
	}

	schema.Tables = tables

	return schema
}

func NewPGSQLTable(schemaName, tableName string, fields []*gontentful.ContentTypeField) PGSQLTable {
	table := PGSQLTable{
		Referencing:  make([]string, 0),
		ReferencedBy: make([]string, 0),
		SchemaName:   schemaName,
		TableName:    tableName,
	}

	columns := make([]PGSQLColumn, 0)
	for _, field := range fields {
		column := NewPGSQLColumn(table, *field)
		columns = append(columns, column)
	}

	table.Columns = columns

	table.Columns[len(table.Columns)-1].Last = true

	return table
}

func NewPGSQLColumn(table PGSQLTable, field gontentful.ContentTypeField) PGSQLColumn {
	column := PGSQLColumn{
		Table:      table,
		SchemaName: table.SchemaName,
		ColumnName: field.ID,
	}

	column.ColumnDesc = column.getColumnDesc(field)

	return column
}

func (c *PGSQLColumn) getColumnDesc(field gontentful.ContentTypeField) string {
	columnType := c.getColumnType(field)
	//if c.isUnique(field.Validations) {
	//	columnType += " unique"
	//}
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
	}
	return ""
}

func (c *PGSQLColumn) getArrayType(items gontentful.FieldTypeArrayItem) string {
	switch items.Type {
	case "Symbol":
		return "text ARRAY"
	case "Link":
		return c.getLinkType(items.Validations)
	}
	return ""
}

func (c *PGSQLColumn) getLinkType(validations []gontentful.FieldValidation) string {
	if len(validations) > 0 {
		if len(validations[0].LinkContentType) != 0 {
			refTable := validations[0].LinkContentType[0]
			c.Table.Referencing = append(c.Table.Referencing, refTable)
			return "integer not null references " + c.SchemaName + "." + refTable + "(id)"
		} else if len(validations[0].LinkMimetypeGroup) != 0 {
			return "text" // image
		}
	} else {
		return "text"
	}
	return ""
}

type PGSQLTable struct {
	Referencing  []string
	ReferencedBy []string
	SchemaName   string
	TableName    string
	Columns      []PGSQLColumn
}

type PGSQLColumn struct {
	Table      PGSQLTable
	SchemaName string
	ColumnName string
	ColumnDesc string
	Last       bool
}
