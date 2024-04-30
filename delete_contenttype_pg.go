package gontentful

import (
	"bytes"
	"fmt"
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
--
DELETE FROM {{ $.SchemaName }}.{{ $.SchemaTableName }} WHERE table_name = {{ $.TableName }};
`

type PGDeleteContentType struct {
	SchemaName      string
	TableName       string
	SysID           string
	SchemaTableName string
}

func NewPGDeleteContentType(schemaName string, sys *Sys) *PGDeleteContentType {
	return &PGDeleteContentType{
		SchemaName:      schemaName,
		TableName:       toSnakeCase(sys.ID),
		SysID:           sys.ID,
		SchemaTableName: SCHEMA_TABLE_NAME,
	}
}

func (s *PGDeleteContentType) Exec(databaseURL string) error {
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
	defer txn.Rollback()

	if s.SchemaName != "" {
		// set schema in use
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", s.SchemaName))
		if err != nil {
			return err
		}
	}

	// os.WriteFile("/tmp/func", buff.Bytes(), 0644)

	_, err = txn.Exec(str)
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *PGDeleteContentType) Render() (string, error) {
	tmpl, err := template.New("").Parse(delContentTypeTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return "", err
	}

	// fmt.Println(buff.String())

	return buff.String(), nil
}
