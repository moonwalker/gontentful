package gontentful

import (
	"bytes"
	"text/template"
)

type PGFunctionsPublish struct {
	Schema         *PGSQLSchema
	Locale         string
	FallbackLocale string
}

func NewPGFunctionsPublish(schema *PGSQLSchema, locale string, fallbackLocale string) *PGFunctionsPublish {
	return &PGFunctionsPublish{
		Schema:         schema,
		Locale:         locale,
		FallbackLocale: fallbackLocale,
	}
}

func (s *PGFunctionsPublish) Render() (string, error) {
	tmpl, err := template.New("").Funcs(funcMap).Parse(pgFuncPublishTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	params := struct {
		Functions      []*PGSQLProcedure
		Locale         string
		FallbackLocale string
	}{
		Functions:      s.Schema.Functions,
		Locale:         s.Locale,
		FallbackLocale: s.FallbackLocale,
	}
	err = tmpl.Execute(&buff, params)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}
