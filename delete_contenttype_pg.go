package gontentful

import (
	"bytes"
	"text/template"

	"github.com/jmoiron/sqlx"
)

const delContentTypeTemplate = `
DROP TABLE IF EXISTS {{ $.SchemaName }}.{{ $.TableName }} CASCADE;
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}._get_{{ $.TableName }}_items CASCADE;
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.{{ $.TableName }}_items CASCADE;
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.{{ $.TableName }}_query CASCADE;
`

type PGDeleteContentType struct {
	SchemaName string
	TableName  string
	SysID      string
}

func NewPGDeleteContentType(schemaName string, sys *Sys) *PGDeleteContentType {
	return &PGDeleteContentType{
		SchemaName: schemaName,
		TableName:  toSnakeCase(sys.ID),
		SysID:      sys.ID,
	}
}

func (s *PGDeleteContentType) Exec(databaseURL string) error {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}

	defer db.Close()

	tmpl, err := template.New("").Parse(delContentTypeTemplate)

	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	//fmt.Println(buff.String())

	_, err = db.Exec(buff.String())

	return err
}