// $ ... | docker exec -i <containerid> psql -U postgres

package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

type PGSQLColumn struct {
	ColumnName string
	ColumnType string
	ColumnDesc string
	Required   bool
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
	Name      string
	Label     string
	Type      string
	ItemsType string
	LinkType  string
	Required  bool
	Localized bool
	Unique    bool
	Disabled  bool
	Omitted   bool
}

type PGSQLTable struct {
	TableName string
	Data      *PGSQLData
	Columns   []*PGSQLColumn
}

type PGSQLReference struct {
	TableName  string
	ForeignKey string
	Reference  string
}

type PGSQLSchema struct {
	SchemaName    string
	Drop          bool
	Space         *Space
	Tables        []*PGSQLTable
	ConTables     []*PGSQLTable
	References    []*PGSQLReference
	DefaultLocale string
	AssetColumns  []string
}

var schemaFuncMap = template.FuncMap{
	"fmtLocale": fmtLocale,
}

func NewPGSQLSchema(schemaName string, space *Space, items []*ContentType) *PGSQLSchema {
	defaultLocale := ""
	for _, loc := range space.Locales {
		if loc.Default {
			defaultLocale = fmtLocale(loc.Code)
			break
		}
	}
	if defaultLocale == "" {
		if len(space.Locales) > 0 {
			defaultLocale = fmtLocale(space.Locales[0].Code)
		} else {
			defaultLocale = "en"
		}
	}
	schema := &PGSQLSchema{
		SchemaName:    schemaName,
		Space:         space,
		Tables:        make([]*PGSQLTable, 0),
		ConTables:     make([]*PGSQLTable, 0),
		References:    make([]*PGSQLReference, 0),
		DefaultLocale: defaultLocale,
		AssetColumns:  assetColumns,
	}

	for _, item := range items {
		table := NewPGSQLTable(item)
		schema.Tables = append(schema.Tables, table)

		conTables, references := createReferences(item, table, space.Locales, defaultLocale)
		schema.ConTables = append(schema.ConTables, conTables...)
		schema.References = append(schema.References, references...)
	}

	return schema
}

func (s *PGSQLSchema) Exec(databaseURL string) error {
	str, err := s.Render()
	if err != nil {
		return err
	}

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	txn, err := db.Beginx()
	if err != nil {
		return err
	}

	// set schema in use
	_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
	if err != nil {
		return err
	}

	_, err = txn.Exec(str)
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	refs := NewPGReferences(s)
	err = refs.Exec(databaseURL)
	if err != nil {
		return err
	}

	funcs := NewPGFunctions(s.SchemaName)
	err = funcs.Exec(databaseURL)
	if err != nil {
		return err
	}

	return nil
}

func (s *PGSQLSchema) Render() (string, error) {
	tmpl, err := template.New("").Funcs(schemaFuncMap).Parse(pgTemplate)
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
	table := &PGSQLTable{
		TableName: toSnakeCase(item.Sys.ID),
		Columns:   make([]*PGSQLColumn, 0),
		Data:      makeModelData(item),
	}

	for _, field := range item.Fields {
		if !field.Omitted {
			column := NewPGSQLColumn(field)
			table.Columns = append(table.Columns, column)
			meta := makeMeta(field)
			table.Data.Metas = append(table.Data.Metas, meta)
			// } else {
			// 	fmt.Println("Ignoring omitted field", field.ID, "in", table.TableName)
		}
	}

	return table
}

func NewPGSQLColumn(field *ContentTypeField) *PGSQLColumn {
	column := &PGSQLColumn{
		ColumnName: toSnakeCase(field.ID),
	}
	column.getColumnDesc(field)
	return column
}

func (c *PGSQLColumn) getColumnDesc(field *ContentTypeField) {
	columnDesc := ""
	if isUnique(field.Validations) {
		columnDesc += " unique"
	}
	c.Required = field.Required && !field.Omitted
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
		// return "text"
		if fieldItems != nil {
			return fmt.Sprintf("%s ARRAY", getColumnType(fieldItems.Type, nil))
		}
		return "text ARRAY"
	case "Object":
		return "text"
	default:
		return "text"
	}
}

func isUnique(validations []*FieldValidation) bool {
	for _, v := range validations {
		if v.Unique {
			return true
		}
	}
	return false
}

func getFieldLinkType(linkType string, validations []*FieldValidation) string {
	if linkType == ASSET {
		return assetTableName
	}
	if linkType == ENTRY {
		for _, v := range validations {
			if v.LinkContentType != nil {
				return toSnakeCase(v.LinkContentType[0])
			}
		}
	}
	return linkType
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
		Name:      toSnakeCase(field.ID),
		Label:     formatText(field.Name),
		Type:      field.Type,
		Required:  field.Required,
		Localized: field.Localized,
		Unique:    isUnique(field.Validations),
		Disabled:  field.Disabled,
		Omitted:   field.Omitted,
	}
	if field.LinkType != "" {
		linkType := getFieldLinkType(field.LinkType, field.Validations)
		if linkType != "" {
			meta.LinkType = linkType
		}
	}
	if field.Items != nil {
		meta.ItemsType = field.Items.Type
		linkType := getFieldLinkType(field.Items.LinkType, field.Items.Validations)
		if linkType != "" {
			meta.LinkType = linkType
		}
	}

	return meta
}

func createReferences(item *ContentType, table *PGSQLTable, locales []*Locale, defaultLocale string) ([]*PGSQLTable, []*PGSQLReference) {
	conTables := make([]*PGSQLTable, 0)
	references := make([]*PGSQLReference, 0)
	for _, field := range item.Fields {
		if !field.Omitted {
			if field.LinkType != "" {
				references = addOneTOne(references, table.TableName, field, locales, defaultLocale)
			} else if field.Items != nil {
				conTables, references = addManyToMany(conTables, references, table.TableName, field, locales, defaultLocale)
			}
		}
	}
	return conTables, references
}

func NewPGSQLCon(tableName string, reference string, locale string) *PGSQLTable {
	return &PGSQLTable{
		TableName: getConTableName(tableName, reference, locale),
		Columns:   getConTableColumns(tableName, reference),
	}
}

func getConTableName(tableName string, reference string, locale string) string {
	return fmt.Sprintf("%s__%s__%s__con", tableName, reference, locale)
}
func getConTableColumns(tableName string, reference string) []*PGSQLColumn {
	return []*PGSQLColumn{
		&PGSQLColumn{
			ColumnName: tableName,
		},
		&PGSQLColumn{
			ColumnName: reference,
		},
	}
}

func addReference(references []*PGSQLReference, tableName string, reference string, foreignKey string, locale string) []*PGSQLReference {
	return append(references, &PGSQLReference{
		TableName:  tableName,
		Reference:  fmt.Sprintf("%s__%s", reference, fmtLocale(locale)),
		ForeignKey: foreignKey,
	})
}

func addOneTOne(references []*PGSQLReference, tableName string, field *ContentTypeField, locales []*Locale, defaultLocale string) []*PGSQLReference {
	linkType := getFieldLinkType(field.LinkType, field.Validations)
	if linkType != "" && linkType != ENTRY {
		foreignKey := toSnakeCase(field.ID)
		if field.Localized {
			for _, loc := range locales {
				locale := fmtLocale(loc.Code)
				references = addReference(references, fmt.Sprintf("%s__%s", tableName, locale), linkType, foreignKey, locale)
			}
		} else {
			references = addReference(references, fmt.Sprintf("%s__%s", tableName, defaultLocale), linkType, foreignKey, defaultLocale)
		}
	}
	return references
}

func addManyToMany(conTables []*PGSQLTable, references []*PGSQLReference, tableName string, field *ContentTypeField, locales []*Locale, defaultLocale string) ([]*PGSQLTable, []*PGSQLReference) {
	linkType := getFieldLinkType(field.Items.LinkType, field.Items.Validations)
	if linkType != "" && linkType != ENTRY {
		if field.Localized {
			for _, loc := range locales {
				locale := fmtLocale(loc.Code)
				conTable := NewPGSQLCon(tableName, linkType, locale)
				conTables = append(conTables, conTable)
				references = addReference(references, conTable.TableName, tableName, tableName, locale)
				references = addReference(references, conTable.TableName, linkType, linkType, locale)
			}
		} else {
			conTable := NewPGSQLCon(tableName, linkType, defaultLocale)
			conTables = append(conTables, conTable)
			references = addReference(references, conTable.TableName, tableName, tableName, defaultLocale)
			references = addReference(references, conTable.TableName, linkType, linkType, defaultLocale)
		}
	}
	return conTables, references
}
