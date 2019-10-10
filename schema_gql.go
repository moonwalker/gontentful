package gontentful

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jinzhu/inflection"
)

const gqlTemplate = `schema {
  query: Query
}

type Query {
  {{- range $_ := .TypeDefs }}
  {{- range $_ := .Resolvers }}
  {{ .Name }}(
	  {{- range $i, $_ := .Args -}}
	  {{ if $i }}, {{ end }}{{ .ArgName }}: {{ .ArgType }}
	  {{- end }}): {{ .Result }}
  {{- end }}
  {{- end }}
}

interface Sys {
  id: ID!
  createdAt: String!
  updatedAt: String!
}

interface Entry {
  sys: EntrySys!
}

type EntrySys implements Sys {
  id: ID!
  createdAt: String!
  updatedAt: String!
  contentTypeId: ID!
}

{{- range $t := .TypeDefs }}
{{ if $t }}{{ end }}
type {{ .TypeName }} implements Entry {
  sys: EntrySys!
  {{- range $_ := .Fields }}
  {{ .FieldName }}: {{ .FieldType }}
  {{- end }}
}
{{- end }}

type FileDetailsImage {
  height: Int
  width: Int
}

type FileDetails {
  size: Int
  image: FileDetailsImage
}

type File {
  contentType: String
  fileName: String
  url: String
  details: FileDetails
}

type AssetSys implements Sys {
  id: ID!
  createdAt: String!
  updatedAt: String!
}

type Asset {
  sys: AssetSys!
  title: String
  description: String
  url: String
  file: File
}`

var (
	singleArgs = []*GraphQLResolverArg{
		&GraphQLResolverArg{"_id", "ID"},
		&GraphQLResolverArg{"_locale", "String"},
		&GraphQLResolverArg{"_include", "Int"},
		&GraphQLResolverArg{"_select", "String"},
	}
	singleIdentityFields = []*GraphQLResolverArg{
		&GraphQLResolverArg{"slug", "String"},
		&GraphQLResolverArg{"code", "String"},
		&GraphQLResolverArg{"name", "String"},
		&GraphQLResolverArg{"key", "String"},
	}
	collectionArgs = []*GraphQLResolverArg{
		&GraphQLResolverArg{"_locale", "String"},
		&GraphQLResolverArg{"_skip", "Int"},
		&GraphQLResolverArg{"_limit", "Int"},
		&GraphQLResolverArg{"_include", "Int"},
		&GraphQLResolverArg{"_select", "String"},
		&GraphQLResolverArg{"_order", "String"},
		&GraphQLResolverArg{"_q", "String"},
	}
)

type GraphQLResolver struct {
	Name   string
	Args   []*GraphQLResolverArg
	Result string
}

type GraphQLResolverArg struct {
	ArgName string
	ArgType string
}

type GraphQLField struct {
	FieldName string
	FieldType string
}

type GraphQLType struct {
	Schema    GraphQLSchema
	TypeName  string
	Fields    []*GraphQLField
	Resolvers []*GraphQLResolver
}

type GraphQLSchema struct {
	Items    []*ContentType
	TypeDefs []*GraphQLType
}

func init() {
	inflection.AddPlural("(bonu)s$", "${1}ses")
	inflection.AddPlural("(hero)$", "${1}es")
}

func NewGraphQLSchema(items []*ContentType) *GraphQLSchema {
	schema := &GraphQLSchema{
		Items:    items,
		TypeDefs: make([]*GraphQLType, 0),
	}

	for _, item := range items {
		typeDef := NewGraphQLTypeDef(schema, item.Sys.ID, item.Fields)
		schema.TypeDefs = append(schema.TypeDefs, typeDef)
	}

	return schema
}

func (s *GraphQLSchema) Render() (string, error) {
	tmpl, err := template.New("").Parse(gqlTemplate)
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

func NewGraphQLTypeDef(schema *GraphQLSchema, typeName string, fields []*ContentTypeField) *GraphQLType {
	typeDef := &GraphQLType{
		TypeName:  strings.Title(typeName),
		Fields:    make([]*GraphQLField, 0),
		Resolvers: make([]*GraphQLResolver, 0),
	}

	// single
	args := getResolverArgs(false, fields)
	typeDef.Resolvers = append(typeDef.Resolvers, NewGraphQLResolver(false, typeName, args, typeDef.TypeName))

	// collection
	pluralName := inflection.Plural(typeName)
	if pluralName != typeName {
		args = getResolverArgs(true, fields)
		typeDef.Resolvers = append(typeDef.Resolvers, NewGraphQLResolver(true, pluralName, args, typeDef.TypeName))
	}

	for _, f := range fields {
		if f.Disabled || f.Omitted {
			continue
		}
		field := NewGraphQLField(schema, f)
		typeDef.Fields = append(typeDef.Fields, field)
	}

	return typeDef
}

func NewGraphQLResolver(collection bool, name string, args []*GraphQLResolverArg, result string) *GraphQLResolver {
	if collection {
		result = fmt.Sprintf("[%s]", result)
	}

	return &GraphQLResolver{
		Name:   name,
		Args:   args,
		Result: result,
	}
}

func getResolverArgs(collection bool, fields []*ContentTypeField) []*GraphQLResolverArg {
	if collection {
		return getCollectionArgs(fields)
	}
	return getSingleArgs(fields)
}

func getSingleArgs(fields []*ContentTypeField) []*GraphQLResolverArg {
	args := singleArgs
	for _, a := range singleExtraArgs {
		if hasField(fields, a.ArgName) {
			args = append(args, a)
		}
	}
	return args
}

func getCollectionArgs(fields []*ContentTypeField) []*GraphQLResolverArg {
	args := collectionArgs
	for _, f := range fields {
		t := isOwnField(f)
		if len(t) > 0 {
			args = append(args, &GraphQLResolverArg{
				ArgName: f.ID,
				ArgType: t,
			})
		}
	}
	return args
}

func isOwnField(field *ContentTypeField) string {
	if field.Type == "Link" || field.Type == "Array" {
		return ""
	}
	switch field.Type {
	case "Integer":
		return "Int"
	case "Number":
		return "Float"
	case "Boolean":
		return "Boolean"
	default:
		return "String"
	}
}

func hasField(fields []*ContentTypeField, id string) bool {
	for _, f := range fields {
		if f.ID == id {
			return true
		}
	}
	return false
}

func NewGraphQLField(schema *GraphQLSchema, f *ContentTypeField) *GraphQLField {
	return &GraphQLField{
		FieldName: f.ID,
		FieldType: isRequired(f.Required, getFieldType(schema, f)),
	}
}

func isRequired(r bool, s string) string {
	if r {
		s += "!"
	}
	return s
}

func getFieldType(schema *GraphQLSchema, field *ContentTypeField) string {
	switch field.Type {
	case "Symbol":
		return "String"
	case "Text":
		return "String"
	case "Integer":
		return "Int"
	case "Number":
		return "Float"
	case "Date":
		return "String"
	case "Location":
		return "String"
	case "Boolean":
		return "Boolean"
	case "Array":
		return getArrayType(schema, field)
	case "Link":
		return getLinkType(schema, field)
	case "Object":
		return "String"
	default:
		return "String"
	}
}

func getArrayType(schema *GraphQLSchema, field *ContentTypeField) string {
	if field.Items == nil || len(field.Items.LinkType) == 0 {
		return "[String]"
	}
	return fmt.Sprintf("[%s]", getValidationContentType(schema, field.Items.LinkType, field.Items.Validations))
}

func getLinkType(schema *GraphQLSchema, field *ContentTypeField) string {
	return getValidationContentType(schema, field.LinkType, field.Validations)
}

func getValidationContentType(schema *GraphQLSchema, t string, validations []*FieldValidation) string {
	if len(validations) > 0 && len(validations[0].LinkContentType) > 0 {
		vt := validations[0].LinkContentType[0]
		// check if validation content type exists
		for _, item := range schema.Items {
			if item.Sys.ID == vt {
				t = vt
				break
			}
		}
	}
	return strings.Title(t)
}
