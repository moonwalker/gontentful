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
  {{ .Name }}({{ .Args }}): {{ .Result }}
  {{- end }}
  {{- end }}
}

{{- range $i := .TypeDefs }}
{{ if $i }}{{ end }}
type {{ .TypeName }} implements Entry {
  sys: EntrySys!
  {{- range $_ := .Fields }}
  {{ .FieldName }}: {{ .FieldType }}
  {{- end }}
}
{{- end }}`

const (
	singleArgs     = "id: ID, locale: String, include: Int, select: String, order: String"
	collectionArgs = "locale: String, skip: Int, limit: Int, include: Int, select: String, order: String, q: String, label: String, routeSlug: String, iconSlug: String, showForUsers: String"
)

type GraphQLResolver struct {
	Name   string
	Args   string
	Result string
}

type GraphQLField struct {
	FieldName string
	FieldType string
}

type GraphQLType struct {
	TypeName  string
	Fields    []GraphQLField
	Resolvers []GraphQLResolver
}

type GraphQLSchema struct {
	TypeDefs []GraphQLType
}

func init() {
	inflection.AddPlural("(bonu)s$", "${1}ses")
	inflection.AddPlural("(hero)$", "${1}es")
}

func NewGraphQLSchema(items []ContentType) GraphQLSchema {
	schema := GraphQLSchema{
		TypeDefs: make([]GraphQLType, 0),
	}

	for _, item := range items {
		typeDef := NewGraphQLTypeDef(item.Sys.ID, item.Fields)
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

func NewGraphQLTypeDef(typeName string, fields []*ContentTypeField) GraphQLType {
	typeDef := GraphQLType{
		TypeName:  strings.Title(typeName),
		Fields:    make([]GraphQLField, 0),
		Resolvers: make([]GraphQLResolver, 0),
	}

	// single
	typeDef.Resolvers = append(typeDef.Resolvers, NewGraphQLResolver(typeName, typeDef.TypeName, false))

	// collection
	pluralName := inflection.Plural(typeName)
	if pluralName != typeName {
		typeDef.Resolvers = append(typeDef.Resolvers, NewGraphQLResolver(pluralName, typeDef.TypeName, true))
	}

	for _, f := range fields {
		field := NewGraphQLField(f)
		typeDef.Fields = append(typeDef.Fields, field)
	}

	return typeDef
}

func NewGraphQLResolver(name string, result string, collection bool) GraphQLResolver {
	args := singleArgs

	if collection {
		args = collectionArgs
		result = fmt.Sprintf("[%s]", result)
	}

	return GraphQLResolver{
		Name:   name,
		Args:   args,
		Result: result,
	}
}

func NewGraphQLField(f *ContentTypeField) GraphQLField {
	return GraphQLField{
		FieldName: f.ID,
		FieldType: isRequired(f.Required, getFieldType(f)),
	}
}

func isRequired(r bool, s string) string {
	if r {
		s += "!"
	}
	return s
}

func getFieldType(field *ContentTypeField) string {
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
		return getArrayType(field)
	case "Link":
		return getLinkType(field)
	case "Object":
		return "String"
	default:
		return "String"
	}
}

func getArrayType(field *ContentTypeField) string {
	if field.Items == nil || len(field.Items.LinkType) == 0 {
		return "[String]"
	}
	return fmt.Sprintf("[%s]", getValidationContentType(field.Items.LinkType, field.Items.Validations))
}

func getLinkType(field *ContentTypeField) string {
	return getValidationContentType(field.LinkType, field.Validations)
}

func getValidationContentType(t string, validations []FieldValidation) string {
	if len(validations) > 0 && len(validations[0].LinkContentType) > 0 {
		t = validations[0].LinkContentType[0]
	}
	return strings.Title(t)
}
