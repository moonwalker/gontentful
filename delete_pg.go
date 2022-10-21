package gontentful

import (
	"bytes"
	"text/template"

	"github.com/jmoiron/sqlx"
)

const deleteTemplate = "DELETE FROM {{ .SchemaName }}.{{ .TableName }} WHERE _sys_id = '{{ .SysID }}';"

type PGDelete struct {
	SchemaName string
	TableName  string
	SysID      string
}

func NewPGDelete(schemaName string, sys *Sys) *PGDelete {
	tableName := ""
	if sys.Type == DELETED_ENTRY {
		tableName = toSnakeCase(sys.ContentType.Sys.ID)
	} else if sys.Type == DELETED_ASSET {
		tableName = assetTableName
	}
	return &PGDelete{
		SchemaName: schemaName,
		TableName:  tableName,
		SysID:      sys.ID,
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

	// fmt.Println(buff.String())

	_, err = db.Exec(buff.String())

	return err
}
