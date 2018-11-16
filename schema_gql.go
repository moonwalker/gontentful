package gontentful

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

const gqlTemplate = `{{ range $t := .TypeDefs }}
type {{ .TypeName }} implements Entry {
  sys: EntrySys!
  {{- range $f := .Fields }}
  {{ .FieldName }}: {{ .FieldType }}
  {{- end }}
}
{{ end -}}`

type GraphQLField struct {
	FieldName string
	FieldType string
}

type GraphQLType struct {
	TypeName string
	Fields   []GraphQLField
}

type GraphQLSchema struct {
	TypeDefs []GraphQLType
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
		TypeName: strings.Title(typeName),
		Fields:   make([]GraphQLField, 0),
	}

	for _, f := range fields {
		field := NewGraphQLField(f)
		typeDef.Fields = append(typeDef.Fields, field)
	}

	return typeDef
}

func NewGraphQLField(f *ContentTypeField) GraphQLField {
	field := GraphQLField{
		FieldName: f.ID,
		FieldType: isRequired(f.Required, getFieldType(f)),
	}
	return field
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
