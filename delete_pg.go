package gontentful

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

const deleteTemplate = "DELETE FROM {{ .SchemaName }}.{{ .TableName }} WHERE _sys_id = '{{ .SysID }}';"

type PGDelete struct {
	SchemaName string
	TableName  string
	SysID      string
}

func NewPGDelete(schemaName string, tableName string, sysID string) *PGDelete {
	return &PGDelete{
		SchemaName: schemaName,
		TableName:  tableName,
		SysID:      sysID,
	}
}

func (s *PGDelete) Exec(databaseURL string) error {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return err
	}

	defer db.Close()

	tmpl, err := template.New("").Parse(deleteTemplate)

	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	fmt.Println(buff.String())

	_, err = db.Exec(buff.String())

	return err
}
