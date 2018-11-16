package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
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

func init() {
	schemaCmd.AddCommand(gqlSchemaCmd)
}

var gqlSchemaCmd = &cobra.Command{
	Use:   "gql",
	Short: "Creates graphql schema",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
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
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schema := NewGraphQLSchema(resp.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(str)
	},
}

func NewGraphQLSchema(items []gontentful.ContentType) GraphQLSchema {
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

func NewGraphQLTypeDef(typeName string, fields []*gontentful.ContentTypeField) GraphQLType {
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

func NewGraphQLField(f *gontentful.ContentTypeField) GraphQLField {
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

func getFieldType(field *gontentful.ContentTypeField) string {
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

func getArrayType(field *gontentful.ContentTypeField) string {
	if field.Items == nil || len(field.Items.LinkType) == 0 {
		return "[String]"
	}
	return fmt.Sprintf("[%s]", field.Items.LinkType)
}

func getLinkType(field *gontentful.ContentTypeField) string {
	if len(field.Validations) > 0 && len(field.Validations[0].LinkContentType) > 0 {
		return strings.Title(field.Validations[0].LinkContentType[0])
	}
	return strings.Title(field.LinkType)
}
