// $ ... | docker exec -i <containerid> psql -U postgres

package gontentful

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
)

const (
	defaultMaxIncludeDepth = 3
)

type PGSQLProcedureColumn struct {
	TableName    string
	ColumnName   string
	Alias        string
	ConTableName string
	Reference    *PGSQLProcedureReference
	JoinAlias    string
	IsAsset      bool
	Localized    bool
	SqlType      string
}

type PGSQLProcedureReference struct {
	TableName    string
	ForeignKey   string
	Columns      []*PGSQLProcedureColumn
	JoinAlias    string
	Localized    bool
	HasLocalized bool
}

type PGSQLProcedure struct {
	TableName    string
	Columns      []*PGSQLProcedureColumn
	HasLocalized bool
}

type PGSQLColumn struct {
	ColumnName string
	ColumnType string
	ColumnDesc string
	Required   bool
	IsIndex    bool
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
	Indices   map[string]string
}

type PGSQLReference struct {
	TableName    string
	ForeignKey   string
	Reference    string
	IsManyToMany bool
}

type PGSQLDependency struct {
	TableName string
	Reference string
}

type PGSQLSchema struct {
	SchemaName     string
	Locales        []*Locale
	Tables         []*PGSQLTable
	ConTables      []*PGSQLTable
	References     []*PGSQLReference
	Dependencies   []*PGSQLDependency
	Functions      []*PGSQLProcedure
	DeleteTriggers []*PGSQLDeleteTrigger
	AssetTableName string
	AssetColumns   []string
	DropTables     bool
	ContentSchema  string
}

type PGSQLDeleteTrigger struct {
	TableName string
	ConTables []string
}

var schemaFuncMap = template.FuncMap{
	"fmtLocale": fmtLocale,
}

func NewPGSQLSchema(schemaName string, space *Space, contentTypeFilter string, items []*ContentType, includeDepth int64) *PGSQLSchema {
	schema := &PGSQLSchema{
		SchemaName:     schemaName,
		Locales:        space.Locales,
		Tables:         make([]*PGSQLTable, 0),
		ConTables:      make([]*PGSQLTable, 0),
		References:     make([]*PGSQLReference, 0),
		Dependencies:   make([]*PGSQLDependency, 0),
		Functions:      make([]*PGSQLProcedure, 0),
		DeleteTriggers: make([]*PGSQLDeleteTrigger, 0),
		AssetColumns:   assetColumns,
	}

	itemsMap := make(map[string]*ContentType)
	for _, item := range items {
		itemsMap[item.Sys.ID] = item
	}

	for _, item := range items {
		if len(contentTypeFilter) > 0 && contentTypeFilter != item.Sys.ID {
			continue
		}

		table, conTables, references, dependencies, proc := NewPGSQLTable(item, itemsMap, includeDepth)

		schema.Tables = append(schema.Tables, table)
		schema.ConTables = append(schema.ConTables, conTables...)
		schema.References = append(schema.References, references...)
		schema.Dependencies = append(schema.Dependencies, dependencies...)
		schema.Functions = append(schema.Functions, proc)
	}
	schema.DeleteTriggers = getDeleteTriggers(schema.References)

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
	if s.SchemaName != "" {
		// set schema in use
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
		if err != nil {
			return err
		}
	}

	// ioutil.WriteFile("/tmp/schema", []byte(str), 0644)

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

	// funcs := NewPGFunctions(s)
	// err = funcs.Exec(databaseURL)
	// if err != nil {
	// 	return err
	// }

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

func NewPGSQLTable(item *ContentType, items map[string]*ContentType, includeDepth int64) (*PGSQLTable, []*PGSQLTable, []*PGSQLReference, []*PGSQLDependency, *PGSQLProcedure) {
	table := &PGSQLTable{
		TableName: toSnakeCase(item.Sys.ID),
		Columns:   make([]*PGSQLColumn, 0),
		Data:      makeModelData(item),
	}
	conTables := make([]*PGSQLTable, 0)
	references := make([]*PGSQLReference, 0)
	dependencies := make([]*PGSQLDependency, 0)
	proc := &PGSQLProcedure{
		TableName: table.TableName,
		Columns:   make([]*PGSQLProcedureColumn, 0),
	}
	include := includeDepth
	if include == 0 {
		include = defaultMaxIncludeDepth
	}

	for _, field := range item.Fields {
		if !field.Omitted {
			column := NewPGSQLColumn(field)
			table.Columns = append(table.Columns, column)
			procColumn := NewPGSQLProcedureColumn(column.ColumnName, field, items, table.TableName, include, 0, "")

			if field.LinkType != "" {
				references, dependencies = addOneTOne(references, dependencies, table.TableName, field)
			} else if field.Items != nil {
				conTables, references, dependencies = addManyToMany(conTables, references, dependencies, table.TableName, field)
			}
			proc.Columns = append(proc.Columns, procColumn)
			if procColumn.Localized {
				proc.HasLocalized = true
			}

			// } else {
			// 	fmt.Println("Ignoring omitted field", field.ID, "in", table.TableName)
		}
	}

	return table, conTables, references, dependencies, proc
}

func NewPGSQLColumn(field *ContentTypeField) *PGSQLColumn {
	column := &PGSQLColumn{
		ColumnName: toSnakeCase(field.ID),
		IsIndex:    isIndex(field.ID),
	}
	column.getColumnDesc(field)
	return column
}

func isIndex(fieldName string) bool {
	return fieldName == "slug" || fieldName == "code" || fieldName == "key"
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
		return "jsonb"
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

func getFieldLinkContentType(validations []*FieldValidation) string {
	for _, v := range validations {
		if v.LinkContentType != nil {
			return v.LinkContentType[0]
		}
	}
	return ""
}

func getFieldLinkType(linkType string, validations []*FieldValidation) string {
	if linkType == ASSET {
		return assetTableName
	}
	if linkType == ENTRY {
		lct := getFieldLinkContentType(validations)
		if lct != "" {
			return toSnakeCase(lct)
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

func NewPGSQLCon(tableName string, fieldName string, reference string) *PGSQLTable {
	return &PGSQLTable{
		TableName: getConTableName(tableName, fieldName),
		Columns:   getConTableColumns(tableName, reference),
		Indices:   map[string]string{"id_locale": fmt.Sprintf("%s_sys_id,_locale", tableName), "sys_id_locale": fmt.Sprintf("%s_sys_id,_locale", reference)},
	}
}

func getConTableName(tableName string, fieldName string) string {
	return fmt.Sprintf("%.63s", fmt.Sprintf("%s__%s", tableName, fieldName))
}

func getConTableColumns(tableName string, reference string) []*PGSQLColumn {
	return []*PGSQLColumn{
		&PGSQLColumn{
			ColumnName: tableName,
		},
		&PGSQLColumn{
			ColumnName: fmt.Sprintf("%s_sys_id", tableName),
		},
		&PGSQLColumn{
			ColumnName: reference,
		},
		&PGSQLColumn{
			ColumnName: fmt.Sprintf("%s_sys_id", reference),
		},
		&PGSQLColumn{
			ColumnName: "_locale",
		},
	}
}

func addOneTOne(references []*PGSQLReference, dependencies []*PGSQLDependency, tableName string, field *ContentTypeField) ([]*PGSQLReference, []*PGSQLDependency) {
	linkType := getFieldLinkType(field.LinkType, field.Validations)
	if linkType != "" && linkType != ENTRY {
		foreignKey := toSnakeCase(field.ID)
		references = append(references, &PGSQLReference{
			TableName:    tableName,
			Reference:    linkType,
			ForeignKey:   foreignKey,
			IsManyToMany: false,
		})
		dependencies = append(dependencies, &PGSQLDependency{
			TableName: tableName,
			Reference: linkType,
		})
	}
	return references, dependencies
}

func addManyToMany(conTables []*PGSQLTable, references []*PGSQLReference, dependencies []*PGSQLDependency, tableName string, field *ContentTypeField) ([]*PGSQLTable, []*PGSQLReference, []*PGSQLDependency) {
	linkType := getFieldLinkType(field.Items.LinkType, field.Items.Validations)
	if linkType != "" && linkType != ENTRY {
		conTable := NewPGSQLCon(tableName, toSnakeCase(field.ID), linkType)
		conTables = append(conTables, conTable)
		references = append(references, &PGSQLReference{
			TableName:    conTable.TableName,
			Reference:    tableName,
			ForeignKey:   tableName,
			IsManyToMany: true,
		}, &PGSQLReference{
			TableName:    conTable.TableName,
			Reference:    linkType,
			ForeignKey:   linkType,
			IsManyToMany: true,
		})
		dependencies = append(dependencies, &PGSQLDependency{
			TableName: tableName,
			Reference: linkType,
		})
	}
	return conTables, references, dependencies
}

func NewPGSQLProcedureColumn(columnName string, field *ContentTypeField, items map[string]*ContentType, tableName string, maxIncludeDepth int64, includeDepth int64, path string) *PGSQLProcedureColumn {
	col := &PGSQLProcedureColumn{
		TableName:  tableName,
		ColumnName: columnName,
		Alias:      field.ID,
		Localized:  field.Localized,
		SqlType:    mapFieldType(columnName, field.Type, field.Items, field),
	}

	if field.LinkType == ASSET {
		col.IsAsset = true
		assetJoinAlias := getJoinAlias(path, columnName, assetTableName)
		if path == "" {
			col.JoinAlias = tableName
		} else {
			col.JoinAlias = assetJoinAlias
		}
		col.Reference = &PGSQLProcedureReference{
			TableName:  assetTableName,
			ForeignKey: toSnakeCase(field.ID),
			JoinAlias:  assetJoinAlias,
			Localized:  col.Localized,
		}
	} else if field.LinkType != "" {
		linkType := getFieldLinkContentType(field.Validations)
		linkTableName := toSnakeCase(linkType)
		if linkType != "" && linkType != ENTRY {
			joinAlias := getJoinAlias(path, columnName, linkTableName)
			if path == "" {
				col.JoinAlias = tableName
			} else {
				col.JoinAlias = joinAlias
			}
			col.Reference = &PGSQLProcedureReference{
				TableName:  linkTableName,
				ForeignKey: toSnakeCase(field.ID),
				Columns:    make([]*PGSQLProcedureColumn, 0),
				JoinAlias:  joinAlias,
				Localized:  col.Localized,
			}

			if includeDepth <= maxIncludeDepth && items[linkType] != nil {
				itemTableName := toSnakeCase(items[linkType].Sys.ID)
				for _, f := range items[linkType].Fields {
					if !f.Omitted {
						fieldColumnName := toSnakeCase(f.ID)
						procColumn := NewPGSQLProcedureColumn(fieldColumnName, f, items, itemTableName, maxIncludeDepth, includeDepth+1, getPath(path, columnName))
						procColumn.JoinAlias = joinAlias
						col.Reference.Columns = append(col.Reference.Columns, procColumn)
					}
				}
			}
		}
	} else if field.Items != nil {
		if field.Items.LinkType == ASSET {
			col.ConTableName = getConTableName(tableName, toSnakeCase(field.ID))
			assetJoinAlias := getJoinAlias(path, columnName, assetTableName)
			if path == "" {
				col.JoinAlias = tableName
			} else {
				col.JoinAlias = assetJoinAlias
			}
			col.IsAsset = true
			col.Reference = &PGSQLProcedureReference{
				TableName:  assetTableName,
				ForeignKey: toSnakeCase(field.ID),
				JoinAlias:  assetJoinAlias,
				Localized:  col.Localized,
			}
		} else if field.Items.LinkType != "" {
			conLinkType := getFieldLinkContentType(field.Items.Validations)
			if conLinkType != "" && conLinkType != ENTRY {
				col.ConTableName = getConTableName(tableName, toSnakeCase(field.ID))
				conLinkTableName := toSnakeCase(conLinkType)
				conJoinAlias := getJoinAlias(path, columnName, conLinkTableName)
				if path == "" {
					col.JoinAlias = tableName
				} else {
					col.JoinAlias = conJoinAlias
				}
				col.Reference = &PGSQLProcedureReference{
					TableName:  conLinkTableName,
					ForeignKey: toSnakeCase(field.ID),
					Columns:    make([]*PGSQLProcedureColumn, 0),
					JoinAlias:  conJoinAlias,
					Localized:  col.Localized,
				}
				if includeDepth <= maxIncludeDepth && items[conLinkType] != nil {
					itemTableName := toSnakeCase(items[conLinkType].Sys.ID)
					for _, f := range items[conLinkType].Fields {
						if !f.Omitted {
							fieldColumnName := toSnakeCase(f.ID)
							procColumn := NewPGSQLProcedureColumn(fieldColumnName, f, items, itemTableName, maxIncludeDepth, includeDepth+1, getPath(path, columnName))
							procColumn.JoinAlias = conJoinAlias
							col.Reference.Columns = append(col.Reference.Columns, procColumn)
						}
					}
				}
			}
		}
	}

	if col.Reference != nil {
		col.Reference.HasLocalized = getHasLocalized(col.Reference)
	}

	return col
}

func mapFieldType(fieldName string, fieldType string, fieldItems *FieldTypeArrayItem, field *ContentTypeField) string {
	switch fieldType {
	case "Integer":
		return "integer"
	case "Number":
		return "decimal"
	case "Date":
		return "date"
	case "Location":
		return "point"
	case "Object":
		return "jsonb"
	case "Symbol":
		return "text"
	case "Link":
		if field.LinkType == "Entry" && len(field.Validations) == 0 {
			return "text"
		}
		return "json"
	case "Text":
		return "text"
	case "Boolean":
		return "boolean"
	case "Array":
		if fieldItems != nil {
			switch fieldItems.Type {
			case "Link":
				if fieldItems.LinkType == "Entry" && len(fieldItems.Validations) == 0 {
					return "text[]"
				}
				return "json"
			default:
				return fmt.Sprintf("%s[]", mapFieldType(fieldName, fieldItems.Type, nil, nil))
			}
		}
		return "text[]"
	default:
		return "text"
	}
}

func getHasLocalized(ref *PGSQLProcedureReference) bool {
	for _, col := range ref.Columns {
		if col.Localized || col.IsAsset {
			if col.Reference != nil {
				col.Reference.HasLocalized = true
			}
			return true
		}
		if col.Reference != nil {
			col.Reference.HasLocalized = getHasLocalized(col.Reference)
			if col.Reference.HasLocalized {
				return true
			}
		}
	}
	return false
}

func getJoinAlias(path string, columnName, tableName string) string {
	if len(path) == 0 {
		return fmt.Sprintf("%s__%s", columnName, tableName)
	}
	return truncatePath(fmt.Sprintf("%s__%s__%s", truncatePath(path), truncateColumn(columnName), tableName))
}

func getPath(path string, columnName string) string {
	if len(path) == 0 {
		return columnName
	}
	return fmt.Sprintf("%s__%s", truncatePath(path), truncateColumn(columnName))

}

func truncatePath(path string) string {
	idx := strings.LastIndex(path, "__")
	if idx == -1 {
		return path
	}
	re := regexp.MustCompile(`_(\S)[^_]*`)
	return fmt.Sprintf("%s__%s", path[:idx], re.ReplaceAllString(path[idx+1:], "$1"))
}

func truncateColumn(column string) string {
	items := strings.Split(column, "_")
	if len(items) < 3 {
		return column
	}
	for idx, item := range items {
		if idx < (len(items) - 2) {
			items[idx] = item[:1]
		}
	}
	return strings.Join(items, "_")
}

func getDeleteTriggers(references []*PGSQLReference) []*PGSQLDeleteTrigger {
	res := make([]*PGSQLDeleteTrigger, 0)
	delTriggerMap := make(map[string][]string, 0)
	for _, ref := range references {
		if !ref.IsManyToMany {
			continue
		}
		if delTriggerMap[ref.Reference] == nil {
			delTriggerMap[ref.Reference] = make([]string, 0)
		}
		delTriggerMap[ref.Reference] = append(delTriggerMap[ref.Reference], ref.TableName)
	}
	for tn, ct := range delTriggerMap {
		res = append(res, &PGSQLDeleteTrigger{TableName: tn, ConTables: ct})
	}
	return res
}
